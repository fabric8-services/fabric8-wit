package controller

import (
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/path"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// IterationController implements the iteration resource.
type IterationController struct {
	*goa.Controller
	db     application.DB
	config IterationControllerConfiguration
}

// IterationControllerConfiguration configuration for the IterationController

type IterationControllerConfiguration interface {
	GetCacheControlIterations() string
	GetCacheControlIteration() string
}

// NewIterationController creates a iteration controller.
func NewIterationController(service *goa.Service, db application.DB, config IterationControllerConfiguration) *IterationController {
	return &IterationController{Controller: service.NewController("IterationController"), db: db, config: config}
}

// CreateChild runs the create-child action.
func (c *IterationController) CreateChild(ctx *app.CreateChildIterationContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	parentID, err := uuid.FromString(ctx.IterationID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {

		parent, err := appl.Iterations().Load(ctx, parentID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		s, err := appl.Spaces().Load(ctx, parent.SpaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		if !uuid.Equal(*currentUser, s.OwnerId) {
			log.Warn(ctx, map[string]interface{}{
				"space_id":     s.ID,
				"space_owner":  s.OwnerId,
				"current_user": *currentUser,
			}, "user is not the space owner")
			return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not the space owner"))
		}

		reqIter := ctx.Payload.Data
		if reqIter.Attributes.Name == nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil"))
		}

		childPath := append(parent.Path, parent.ID)

		if ctx.Payload.Data.Attributes.UserActive != nil {
			reqIter.Attributes.UserActive = ctx.Payload.Data.Attributes.UserActive
		} else {
			userActive := false
			reqIter.Attributes.UserActive = &userActive

		}

		newItr := iteration.Iteration{
			SpaceID:    parent.SpaceID,
			Path:       childPath,
			Name:       *reqIter.Attributes.Name,
			StartAt:    reqIter.Attributes.StartAt,
			EndAt:      reqIter.Attributes.EndAt,
			UserActive: *reqIter.Attributes.UserActive,
		}

		err = appl.Iterations().Create(ctx, &newItr)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		// For create, count will always be zero hence no need to query
		// by passing empty map, updateIterationsWithCounts will be able to put zero values
		wiCounts := make(map[string]workitem.WICountsPerIteration)
		var responseData *app.Iteration
		allParentsUUIDs := newItr.Path
		iterations, error := appl.Iterations().LoadMultiple(ctx, allParentsUUIDs)
		if error != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		itrMap := make(iterationIDMap)
		for _, itr := range iterations {
			itrMap[itr.ID] = itr
		}
		responseData = ConvertIteration(ctx.Request, newItr, parentPathResolver(itrMap), updateIterationsWithCounts(wiCounts))
		res := &app.IterationSingle{
			Data: responseData,
		}
		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.Request, app.IterationHref(res.Data.ID)))
		return ctx.Created(res)
	})
}

// Show runs the show action.
func (c *IterationController) Show(ctx *app.ShowIterationContext) error {
	id, err := uuid.FromString(ctx.IterationID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		iter, err := appl.Iterations().Load(ctx, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalRequest(*iter, c.config.GetCacheControlIteration, func() error {
			wiCounts, err := appl.WorkItems().GetCountsForIteration(ctx, iter)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			var responseData *app.Iteration
			allParentsUUIDs := iter.Path
			iterations, error := appl.Iterations().LoadMultiple(ctx, allParentsUUIDs)
			if error != nil {
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			itrMap := make(iterationIDMap)
			for _, itr := range iterations {
				itrMap[itr.ID] = itr
			}
			responseData = ConvertIteration(ctx.Request, *iter, parentPathResolver(itrMap), updateIterationsWithCounts(wiCounts))
			res := &app.IterationSingle{
				Data: responseData,
			}
			return ctx.OK(res)
		})
	})
}

// Update runs the update action.
func (c *IterationController) Update(ctx *app.UpdateIterationContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	id, err := uuid.FromString(ctx.IterationID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		itr, err := appl.Iterations().Load(ctx.Context, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		s, err := appl.Spaces().Load(ctx, itr.SpaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		if !uuid.Equal(*currentUser, s.OwnerId) {
			log.Warn(ctx, map[string]interface{}{
				"space_id":     s.ID,
				"space_owner":  s.OwnerId,
				"current_user": *currentUser,
			}, "user is not the space owner")
			return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not the space owner"))
		}
		if ctx.Payload.Data.Attributes.Name != nil {
			itr.Name = *ctx.Payload.Data.Attributes.Name
		}
		if ctx.Payload.Data.Attributes.StartAt != nil {
			itr.StartAt = ctx.Payload.Data.Attributes.StartAt
		}
		if ctx.Payload.Data.Attributes.EndAt != nil {
			itr.EndAt = ctx.Payload.Data.Attributes.EndAt
		}
		if ctx.Payload.Data.Attributes.Description != nil {
			itr.Description = ctx.Payload.Data.Attributes.Description
		}
		if ctx.Payload.Data.Attributes.State != nil {
			if *ctx.Payload.Data.Attributes.State == iteration.IterationStateStart {
				res, err := appl.Iterations().CanStart(ctx, itr)
				if res == false && err != nil {
					return jsonapi.JSONErrorResponse(ctx, err)
				}
			}
			itr.State = *ctx.Payload.Data.Attributes.State
		}
		if ctx.Payload.Data.Attributes.UserActive != nil {
			itr.UserActive = *ctx.Payload.Data.Attributes.UserActive
		}
		itr, err = appl.Iterations().Save(ctx.Context, *itr)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		wiCounts, err := appl.WorkItems().GetCountsForIteration(ctx, itr)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		var responseData *app.Iteration
		allParentsUUIDs := itr.Path
		iterations, error := appl.Iterations().LoadMultiple(ctx, allParentsUUIDs)
		if error != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		itrMap := make(iterationIDMap)
		for _, itr := range iterations {
			itrMap[itr.ID] = itr
		}
		responseData = ConvertIteration(ctx.Request, *itr, parentPathResolver(itrMap), updateIterationsWithCounts(wiCounts))
		res := &app.IterationSingle{
			Data: responseData,
		}
		return ctx.OK(res)
	})
}

// Delete runs the delete action.
func (c *IterationController) Delete(ctx *app.DeleteIterationContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	err = application.Transactional(c.db, func(appl application.Application) error {
		itr, err := appl.Iterations().Load(ctx.Context, ctx.IterationID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		s, err := appl.Spaces().Load(ctx, itr.SpaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		if !uuid.Equal(*currentUser, s.OwnerId) {
			log.Warn(ctx, map[string]interface{}{
				"space_id":     s.ID,
				"space_owner":  s.OwnerId,
				"current_user": *currentUser,
			}, "user is not the space owner")
			return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not the space owner"))
		}
		is := c.IsRoot(itr, s)
		if is {
			log.Warn(ctx, map[string]interface{}{
				"space_id":     s.ID,
				"iteration_id": itr.ID,
			}, "can not delte root iteration")
			return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("can not delete root iteration"))
		}
		subtree, err := appl.Iterations().LoadChildren(ctx, ctx.IterationID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}
		// delete all children along with given iteration
		subtree = append(subtree, *itr)
		for _, child := range subtree {
			err = appl.Iterations().Delete(ctx.Context, child.ID)
			if err != nil {
				log.Warn(ctx, map[string]interface{}{
					"iteration_id": child.ID,
					"err":          err.Error(),
				}, "unable to delete iteration")
				return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
			}
			// fetch associated work items
			wis, err := appl.WorkItems().LoadByIteration(ctx, child.ID)
			if err != nil {
				return err
			}
			// set root iteration to associated work items
			for _, wi := range wis {
				// to be safe, load root iteration of this WI and then assign it
				ri, err := appl.Iterations().Root(ctx, wi.SpaceID)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"space_id": wi.SpaceID,
						"err":      err.Error(),
					}, "unable to get root iteration for space")
					return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
				}
				// move WI to root iteration
				wi.Fields[workitem.SystemIteration] = ri.ID.String()
				_, err = appl.WorkItems().Save(ctx, wi.SpaceID, *wi, *currentUser)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"workitem_id": wi.ID,
						"err":         err.Error(),
					}, "unable to update iteration for work item")
					return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return ctx.OK([]byte{})
}

// IsRoot Checks if given iteration is a root iteration or not
func (c *IterationController) IsRoot(itr *iteration.Iteration, sp *space.Space) bool {
	return (itr.SpaceID == sp.ID && itr.Path.String() == path.SepInService)
}

// IterationConvertFunc is a open ended function to add additional links/data/relations to a Iteration during
// conversion from internal to API
type IterationConvertFunc func(*http.Request, *iteration.Iteration, *app.Iteration)

// ConvertIterations converts between internal and external REST representation
func ConvertIterations(request *http.Request, Iterations []iteration.Iteration, additional ...IterationConvertFunc) []*app.Iteration {
	var is = []*app.Iteration{}
	for _, i := range Iterations {
		is = append(is, ConvertIteration(request, i, additional...))
	}
	return is
}

// ConvertIteration converts between internal and external REST representation
func ConvertIteration(request *http.Request, itr iteration.Iteration, additional ...IterationConvertFunc) *app.Iteration {
	iterationType := iteration.APIStringTypeIteration
	spaceID := itr.SpaceID.String()
	relatedURL := rest.AbsoluteURL(request, app.IterationHref(itr.ID))
	spaceRelatedURL := rest.AbsoluteURL(request, app.SpaceHref(spaceID))
	workitemsRelatedURL := rest.AbsoluteURL(request, app.WorkitemHref("?filter[iteration]="+itr.ID.String()))
	pathToTopMostParent := itr.Path.String()
	activeStatus := itr.IsActive()
	i := &app.Iteration{
		Type: iterationType,
		ID:   &itr.ID,
		Attributes: &app.IterationAttributes{
			Name:         &itr.Name,
			CreatedAt:    &itr.CreatedAt,
			UpdatedAt:    &itr.UpdatedAt,
			StartAt:      itr.StartAt,
			EndAt:        itr.EndAt,
			Description:  itr.Description,
			State:        &itr.State,
			ParentPath:   &pathToTopMostParent,
			UserActive:   &itr.UserActive,
			ActiveStatus: &activeStatus,
		},
		Relationships: &app.IterationRelations{
			Space: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: &space.SpaceType,
					ID:   &spaceID,
				},
				Links: &app.GenericLinks{
					Self:    &spaceRelatedURL,
					Related: &spaceRelatedURL,
				},
			},
			Workitems: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &workitemsRelatedURL,
				},
			},
		},
		Links: &app.GenericLinks{
			Self:    &relatedURL,
			Related: &relatedURL,
		},
	}
	if itr.Path.IsEmpty() == false {
		parentID := itr.Path.This().String()
		parentRelatedURL := rest.AbsoluteURL(request, app.IterationHref(parentID))
		i.Relationships.Parent = &app.RelationGeneric{
			Data: &app.GenericData{
				Type: &iterationType,
				ID:   &parentID,
			},
			Links: &app.GenericLinks{
				Self:    &parentRelatedURL,
				Related: &parentRelatedURL,
			},
		}
	}
	for _, add := range additional {
		add(request, &itr, i)
	}
	return i
}

// ConvertIterationSimple converts a simple Iteration ID into a Generic Reletionship
func ConvertIterationSimple(request *http.Request, id interface{}) *app.GenericData {
	t := iteration.APIStringTypeIteration
	i := fmt.Sprint(id)
	return &app.GenericData{
		Type:  &t,
		ID:    &i,
		Links: createIterationLinks(request, id),
	}
}

func createIterationLinks(request *http.Request, id interface{}) *app.GenericLinks {
	relatedURL := rest.AbsoluteURL(request, app.IterationHref(id))
	return &app.GenericLinks{
		Self:    &relatedURL,
		Related: &relatedURL,
	}
}

// iterationIDMap contains a map that will hold iteration's ID as its key
type iterationIDMap map[uuid.UUID]iteration.Iteration

func parentPathResolver(itrMap iterationIDMap) IterationConvertFunc {
	return func(request *http.Request, itr *iteration.Iteration, appIteration *app.Iteration) {
		parentUUIDs := itr.Path
		pathResolved := ""
		for _, id := range parentUUIDs {
			if i, ok := itrMap[id]; ok {
				pathResolved += iteration.PathSepInService + i.Name
			}
		}
		if pathResolved == "" {
			pathResolved = iteration.PathSepInService
		}
		appIteration.Attributes.ResolvedParentPath = &pathResolved
	}
}

func convertToUUID(uuidStrings []string) []uuid.UUID {
	var uUIDs []uuid.UUID

	for i := 0; i < len(uuidStrings); i++ {
		uuidString, _ := uuid.FromString(uuidStrings[i])
		uUIDs = append(uUIDs, uuidString)
	}
	return uUIDs
}

// updateIterationsWithCounts accepts map of 'iterationID to a workitem.WICountsPerIteration instance'.
// This function returns function of type IterationConvertFunc
// Inner function is able to access `wiCounts` in closure and it is responsible
// for adding 'closed' and 'total' count of WI in relationship's meta for every given iteration.
func updateIterationsWithCounts(wiCounts map[string]workitem.WICountsPerIteration) IterationConvertFunc {
	return func(request *http.Request, itr *iteration.Iteration, appIteration *app.Iteration) {
		var counts workitem.WICountsPerIteration
		if _, ok := wiCounts[appIteration.ID.String()]; ok {
			counts = wiCounts[appIteration.ID.String()]
		} else {
			counts = workitem.WICountsPerIteration{}
		}
		if appIteration.Relationships == nil {
			appIteration.Relationships = &app.IterationRelations{}
		}
		if appIteration.Relationships.Workitems == nil {
			appIteration.Relationships.Workitems = &app.RelationGeneric{}
		}
		if appIteration.Relationships.Workitems.Meta == nil {
			appIteration.Relationships.Workitems.Meta = map[string]interface{}{}
		}
		appIteration.Relationships.Workitems.Meta["total"] = counts.Total
		appIteration.Relationships.Workitems.Meta["closed"] = counts.Closed
	}
}
