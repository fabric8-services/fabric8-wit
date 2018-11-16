package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// Following const are keys to be used in Meta Relationship for counters of Work Item
const (
	KeyTotalWorkItems  = "total"
	KeyClosedWorkItems = "closed"
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

// verifyUser checks if user is a space owner or a collaborator
func verifyUser(ctx context.Context, currentUser uuid.UUID, sp *space.Space) (bool, bool, error) {
	authorized, err := authz.Authorize(ctx, sp.ID.String())
	if err != nil {
		return false, false, err
	}
	var spaceOwner bool
	if uuid.Equal(currentUser, sp.OwnerID) {
		spaceOwner = true
	}
	return authorized, spaceOwner, nil
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
	var parentItr *iteration.Iteration
	var itrSpace *space.Space
	err = application.Transactional(c.db, func(appl application.Application) error {
		parentItr, err = appl.Iterations().Load(ctx, parentID)
		if err != nil {
			return err
		}
		itrSpace, err = appl.Spaces().Load(ctx, parentItr.SpaceID)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	authorized, spaceOwner, err := verifyUser(ctx, *currentUser, itrSpace)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	if !authorized && !spaceOwner {
		log.Error(ctx, map[string]interface{}{
			"space_id":     itrSpace.ID,
			"space_owner":  itrSpace.OwnerID,
			"current_user": *currentUser,
		}, "user is not the space owner")
		if !authorized {
			// unauthorized
			return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError("user is not allowed to create an iteration in this space"))
		}
		// forbidden
		// Ideally we never hit following error
		// But written following line to make it verbose 401 vs 403
		return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not allowed to create an iteration in this space"))
	}
	reqItr := ctx.Payload.Data
	if reqItr.Attributes.Name == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil"))
	}

	if ctx.Payload.Data.Attributes.UserActive != nil {
		reqItr.Attributes.UserActive = ctx.Payload.Data.Attributes.UserActive
	} else {
		userActive := false
		reqItr.Attributes.UserActive = &userActive
	}
	itr := &iteration.Iteration{
		SpaceID:     parentItr.SpaceID,
		Name:        *reqItr.Attributes.Name,
		Description: reqItr.Attributes.Description,
		StartAt:     reqItr.Attributes.StartAt,
		EndAt:       reqItr.Attributes.EndAt,
		UserActive:  *reqItr.Attributes.UserActive,
	}
	if reqItr.ID != nil {
		itr.ID = *reqItr.ID
	} else {
		itr.ID = uuid.NewV4()
	}
	itr.MakeChildOf(*parentItr)

	itrMap := make(iterationIDMap)
	err = application.Transactional(c.db, func(appl application.Application) error {
		err = appl.Iterations().Create(ctx, itr)
		if err != nil {
			return err
		}
		// For create, count will always be zero hence no need to query
		// by passing empty map, updateIterationsWithCounts will be able to put zero values
		parentItrs, err := appl.Iterations().LoadMultiple(ctx, itr.Path.ParentPath())
		if err != nil {
			return err
		}
		for _, itr := range parentItrs {
			itrMap[itr.ID] = itr
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	wiCounts := make(map[string]workitem.WICountsPerIteration)
	responseData := ConvertIteration(ctx.Request, *itr, parentPathResolver(itrMap), updateIterationsWithCounts(wiCounts))
	res := &app.IterationSingle{
		Data: responseData,
	}
	ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.Request, app.IterationHref(res.Data.ID)))
	return ctx.Created(res)
}

// Show runs the show action.
func (c *IterationController) Show(ctx *app.ShowIterationContext) error {
	id, err := uuid.FromString(ctx.IterationID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}
	var itr *iteration.Iteration
	err = application.Transactional(c.db, func(appl application.Application) error {
		itr, err = appl.Iterations().Load(ctx, id)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.ConditionalRequest(*itr, c.config.GetCacheControlIteration, func() error {
		var wiCounts map[string]workitem.WICountsPerIteration
		var parentItrs []iteration.Iteration
		err := application.Transactional(c.db, func(appl application.Application) error {
			var err error
			wiCounts, err = appl.WorkItems().GetCountsForIteration(ctx, itr)
			if err != nil {
				return err
			}
			parentItrs, err = appl.Iterations().LoadMultiple(ctx, itr.Path)
			return err
		})
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		itrMap := make(iterationIDMap)
		for _, itr := range parentItrs {
			itrMap[itr.ID] = itr
		}
		return ctx.OK(&app.IterationSingle{
			Data: ConvertIteration(ctx.Request, *itr, parentPathResolver(itrMap), updateIterationsWithCounts(wiCounts)),
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

	var itr *iteration.Iteration
	var sp *space.Space
	err = application.Transactional(c.db, func(appl application.Application) error {
		itr, err = appl.Iterations().Load(ctx.Context, id)
		if err != nil {
			return err
		}
		sp, err = appl.Spaces().Load(ctx, itr.SpaceID)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	authorized, spaceOwner, err := verifyUser(ctx, *currentUser, sp)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	if !authorized && !spaceOwner {
		log.Error(ctx, map[string]interface{}{
			"space_id":     sp.ID,
			"space_owner":  sp.OwnerID,
			"current_user": *currentUser,
		}, "user is not the space owner")
		if !authorized {
			// unauthorized
			return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError("user is not allowed to create an iteration in this space"))
		}
		// forbidden:
		// Ideally we never hit following error
		// But written following line to make it verbose 401 vs 403
		return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not allowed to create an iteration in this space"))
	}
	var iterations []iteration.Iteration
	var wiCounts map[string]workitem.WICountsPerIteration
	err = application.Transactional(c.db, func(appl application.Application) error {
		if ctx.Payload.Data.Attributes.Name != nil {
			itr.Name = *ctx.Payload.Data.Attributes.Name
		}
		if ctx.Payload.Data.Attributes.StartAt != nil {
			if ctx.Payload.Data.Attributes.StartAt.IsZero() {
				itr.StartAt = nil
			} else {
				itr.StartAt = ctx.Payload.Data.Attributes.StartAt
			}
		}
		if ctx.Payload.Data.Attributes.EndAt != nil {
			if ctx.Payload.Data.Attributes.EndAt.IsZero() {
				itr.EndAt = nil
			} else {
				itr.EndAt = ctx.Payload.Data.Attributes.EndAt
			}
		}
		if ctx.Payload.Data.Attributes.Description != nil {
			itr.Description = ctx.Payload.Data.Attributes.Description
		}
		if ctx.Payload.Data.Attributes.State != nil {
			if *ctx.Payload.Data.Attributes.State == iteration.StateStart.String() {
				res, err := appl.Iterations().CanStart(ctx, itr)
				if res == false && err != nil {
					return err
				}
			}
			itr.State = iteration.State(*ctx.Payload.Data.Attributes.State)
		}
		if ctx.Payload.Data.Attributes.UserActive != nil {
			itr.UserActive = *ctx.Payload.Data.Attributes.UserActive
		}
		var oldSubtree []iteration.Iteration
		if ctx.Payload.Data.Relationships != nil && ctx.Payload.Data.Relationships.Parent != nil {
			// update parent of Iteration
			// do not allow root-iteraiton to update its parent
			if itr.IsRoot(itr.SpaceID) {
				return errors.NewForbiddenError("Parent of root iteration can not be updated")
			}
			newParentID := ctx.Payload.Data.Relationships.Parent.Data.ID
			if newParentID == nil {
				return errors.NewBadParameterError("Data.Relationships.Parent.ID", newParentID).Expected("not nil")
			}
			pid, err := uuid.FromString(*newParentID)
			if err != nil {
				return errors.NewBadParameterError("Data.Relationships.Parent.ID", newParentID)
			}
			// Iteration itself can not be parent of self
			if pid == itr.ID {
				return errors.NewForbiddenError("Parent must be different than subject iteration")
			}
			newParentIteration, err := appl.Iterations().Load(ctx.Context, pid)
			if err != nil {
				return err
			}
			// New parent iteraiton must be from same sapce as that of subject iteration
			if newParentIteration.SpaceID != itr.SpaceID {
				return errors.NewForbiddenError("Parent must be from same space")
			}
			// we need subtree to update later
			oldSubtree, err = appl.Iterations().LoadChildren(ctx, itr.ID)
			if err != nil {
				return err
			}
			// New parent must not be one of existing children of subject iteration
			for _, childItr := range oldSubtree {
				if newParentIteration.ID == childItr.ID {
					return errors.NewForbiddenError("Parent must not be existing child")
				}
			}
			itr.MakeChildOf(*newParentIteration)
		}
		itr, err = appl.Iterations().Save(ctx.Context, *itr)
		if err != nil {
			return err
		}
		if ctx.Payload.Data.Relationships != nil && ctx.Payload.Data.Relationships.Parent != nil {
			// update all child iterations's parent as well
			for _, x := range oldSubtree {
				x.MakeChildOf(*itr)
				_, err = appl.Iterations().Save(ctx.Context, x)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"iteration_id": x.ID,
						"err":          err.Error(),
					}, "unable to update child iteration from subtree")
					return err
				}
			}
		}
		wiCounts, err = appl.WorkItems().GetCountsForIteration(ctx, itr)
		if err != nil {
			return err
		}
		allParentsUUIDs := itr.Path
		iterations, err = appl.Iterations().LoadMultiple(ctx, allParentsUUIDs)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	itrMap := make(iterationIDMap)
	for _, itr := range iterations {
		itrMap[itr.ID] = itr
	}
	responseData := ConvertIteration(ctx.Request, *itr, parentPathResolver(itrMap), updateIterationsWithCounts(wiCounts))
	return ctx.OK(&app.IterationSingle{
		Data: responseData,
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
			return err
		}
		s, err := appl.Spaces().Load(ctx, itr.SpaceID)
		if err != nil {
			return goa.ErrNotFound(err.Error())
		}
		if !uuid.Equal(*currentUser, s.OwnerID) {
			errorMsg := fmt.Sprintf("only the space owner can delete an iteration and %s is not the space owner of %s",
				*currentUser, s.ID)
			log.Warn(ctx, map[string]interface{}{
				"space_id":     s.ID,
				"space_owner":  s.OwnerID,
				"current_user": *currentUser,
			}, errorMsg)
			return errors.NewForbiddenError(errorMsg)
		}
		if itr.IsRoot(s.ID) {
			log.Warn(ctx, map[string]interface{}{
				"space_id":     s.ID,
				"iteration_id": itr.ID,
			}, "cannot delete root iteration")
			return errors.NewForbiddenError("can not delete root iteration")
		}
		subtree, err := appl.Iterations().LoadChildren(ctx, ctx.IterationID)
		if err != nil {
			return err
		}
		// Fetch parent iteration to which work items will get attached
		parentID := itr.Path.ParentID()
		if parentID == uuid.Nil {
			return goa.ErrNotFound("can not find parent iteration")
		}
		parentIteration, err := appl.Iterations().Load(ctx, parentID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"iteration_id": parentID,
				"err":          err.Error(),
			}, "unable to load parent iteration of iteration %s", parentID)
			return err
		}
		// delete all children along with given iteration
		subtree = append(subtree, *itr)
		for _, child := range subtree {
			// fetch associated work items
			wis, err := appl.WorkItems().LoadByIteration(ctx, child.ID)
			if err != nil {
				return err
			}
			// update iteration on all associated work items
			for _, wi := range wis {
				// move WI to parent iteration
				wi.Fields[workitem.SystemIteration] = parentIteration.ID.String()
				_, _, err = appl.WorkItems().Save(ctx, wi.SpaceID, *wi, *currentUser)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"workitem_id": wi.ID,
						"err":         err.Error(),
					}, "unable to update iteration for work item")
					return err
				}
			}
			// now, remove the iteration
			err = appl.Iterations().Delete(ctx.Context, child.ID)
			if err != nil {
				log.Error(ctx, map[string]interface{}{
					"iteration_id": child.ID,
					"err":          err.Error(),
				}, "unable to delete iteration")
				return err
			}
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.NoContent()
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
	pathToTopMostParent := itr.Path.ParentPath().String()
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
			State:        itr.State.StringPtr(),
			ParentPath:   &pathToTopMostParent,
			UserActive:   &itr.UserActive,
			ActiveStatus: &activeStatus,
			Number:       &itr.Number,
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
	if !itr.Path.ParentPath().IsEmpty() {
		parentID := itr.Path.ParentID().String()
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

// ConvertIterationSimple converts a simple Iteration ID into a Generic
// Relationship data+links element
func ConvertIterationSimple(request *http.Request, id interface{}) (*app.GenericData, *app.GenericLinks) {
	t := iteration.APIStringTypeIteration
	i := fmt.Sprint(id)
	data := &app.GenericData{
		Type: &t,
		ID:   &i,
	}
	relatedURL := rest.AbsoluteURL(request, app.IterationHref(i))
	links := &app.GenericLinks{
		Self:    &relatedURL,
		Related: &relatedURL,
	}
	return data, links
}

// iterationIDMap contains a map that will hold iteration's ID as its key
type iterationIDMap map[uuid.UUID]iteration.Iteration

func parentPathResolver(itrMap iterationIDMap) IterationConvertFunc {
	return func(request *http.Request, itr *iteration.Iteration, appIteration *app.Iteration) {
		parentUUIDs := itr.Path.ParentPath()
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
		appIteration.Relationships.Workitems.Meta[KeyTotalWorkItems] = counts.Total
		appIteration.Relationships.Workitems.Meta[KeyClosedWorkItems] = counts.Closed
	}
}
