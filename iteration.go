package main

import (
	"fmt"
	"strings"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// IterationController implements the iteration resource.
type IterationController struct {
	*goa.Controller
	db application.DB
}

// NewIterationController creates a iteration controller.
func NewIterationController(service *goa.Service, db application.DB) *IterationController {
	return &IterationController{Controller: service.NewController("IterationController"), db: db}
}

// CreateChild runs the create-child action.
func (c *IterationController) CreateChild(ctx *app.CreateChildIterationContext) error {
	_, err := login.ContextIdentity(ctx)
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

		reqIter := ctx.Payload.Data
		if reqIter.Attributes.Name == nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil"))
		}

		parentPath := iteration.ConvertToLtreeFormat(parentID.String())
		if parent.ParentPath != "" {
			parentPath = parent.ParentPath + iteration.PathSepInDatabase + parentPath
		}

		newItr := iteration.Iteration{
			SpaceID:    parent.SpaceID,
			ParentPath: parentPath,
			Name:       *reqIter.Attributes.Name,
			StartAt:    reqIter.Attributes.StartAt,
			EndAt:      reqIter.Attributes.EndAt,
		}

		err = appl.Iterations().Create(ctx, &newItr)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		var responseData *app.Iteration
		if newItr.ParentPath != "" {
			allParents := strings.Split(ConvertFromLtreeFormat(newItr.ParentPath), iteration.PathSepInDatabase)
			allParentsUUIDs := []uuid.UUID{}
			for _, x := range allParents {
				id, _ := uuid.FromString(x) // we can safely ignore this error.
				allParentsUUIDs = append(allParentsUUIDs, id)
			}
			iterations, error := appl.Iterations().LoadMultiple(ctx, allParentsUUIDs)
			if error != nil {
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			itrMap := make(iterationIDMap)
			for _, itr := range iterations {
				itrMap[itr.ID] = itr
			}
			responseData = ConvertIteration(ctx.RequestData, &newItr, parentPathResolver(itrMap))
		} else {
			responseData = ConvertIteration(ctx.RequestData, &newItr)
		}
		res := &app.IterationSingle{
			Data: responseData,
		}
		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.RequestData, app.IterationHref(res.Data.ID)))
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
		c, err := appl.Iterations().Load(ctx, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.IterationSingle{}
		res.Data = ConvertIteration(
			ctx.RequestData,
			c)

		return ctx.OK(res)
	})
}

// Update runs the update action.
func (c *IterationController) Update(ctx *app.UpdateIterationContext) error {
	_, err := login.ContextIdentity(ctx)
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
				res, err := appl.Iterations().CanStartIteration(ctx, itr)
				if res == false && err != nil {
					return jsonapi.JSONErrorResponse(ctx, err)
				}
			}
			itr.State = *ctx.Payload.Data.Attributes.State
		}
		itr, err = appl.Iterations().Save(ctx.Context, *itr)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		response := app.IterationSingle{
			Data: ConvertIteration(ctx.RequestData, itr),
		}

		return ctx.OK(&response)
	})
}

// IterationConvertFunc is a open ended function to add additional links/data/relations to a Iteration during
// conversion from internal to API
type IterationConvertFunc func(*goa.RequestData, *iteration.Iteration, *app.Iteration)

// ConvertIterations converts between internal and external REST representation
func ConvertIterations(request *goa.RequestData, Iterations []*iteration.Iteration, additional ...IterationConvertFunc) []*app.Iteration {
	var is = []*app.Iteration{}
	for _, i := range Iterations {
		is = append(is, ConvertIteration(request, i, additional...))
	}
	return is
}

// ConvertIteration converts between internal and external REST representation
func ConvertIteration(request *goa.RequestData, itr *iteration.Iteration, additional ...IterationConvertFunc) *app.Iteration {
	iterationType := iteration.APIStringTypeIteration
	spaceType := "spaces"

	spaceID := itr.SpaceID.String()

	selfURL := rest.AbsoluteURL(request, app.IterationHref(itr.ID))
	spaceSelfURL := rest.AbsoluteURL(request, app.SpaceHref(spaceID))
	workitemsRelatedURL := rest.AbsoluteURL(request, app.WorkitemHref("?filter[iteration]="+itr.ID.String()))
	pathToTopMostParent := iteration.PathSepInService + iteration.ConvertFromLtreeFormat(itr.ParentPath) // /uuid1/uuid2/uuid3s

	i := &app.Iteration{
		Type: iterationType,
		ID:   &itr.ID,
		Attributes: &app.IterationAttributes{
			Name:        &itr.Name,
			StartAt:     itr.StartAt,
			EndAt:       itr.EndAt,
			Description: itr.Description,
			State:       &itr.State,
			ParentPath:  &pathToTopMostParent,
		},
		Relationships: &app.IterationRelations{
			Space: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: &spaceType,
					ID:   &spaceID,
				},
				Links: &app.GenericLinks{
					Self: &spaceSelfURL,
				},
			},
			Workitems: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &workitemsRelatedURL,
				},
			},
		},
		Links: &app.GenericLinks{
			Self: &selfURL,
		},
	}
	if itr.ParentPath != "" {
		allParents := strings.Split(ConvertFromLtreeFormat(itr.ParentPath), iteration.PathSepInDatabase)
		parentID := allParents[len(allParents)-1]
		parentSelfURL := rest.AbsoluteURL(request, app.IterationHref(parentID))
		i.Relationships.Parent = &app.RelationGeneric{
			Data: &app.GenericData{
				Type: &iterationType,
				ID:   &parentID,
			},
			Links: &app.GenericLinks{
				Self: &parentSelfURL,
			},
		}
	}
	for _, add := range additional {
		add(request, itr, i)
	}
	return i
}

// ConvertIterationSimple converts a simple Iteration ID into a Generic Reletionship
func ConvertIterationSimple(request *goa.RequestData, id interface{}) *app.GenericData {
	t := iteration.APIStringTypeIteration
	i := fmt.Sprint(id)
	return &app.GenericData{
		Type:  &t,
		ID:    &i,
		Links: createIterationLinks(request, id),
	}
}

func createIterationLinks(request *goa.RequestData, id interface{}) *app.GenericLinks {
	selfURL := rest.AbsoluteURL(request, app.IterationHref(id))
	return &app.GenericLinks{
		Self: &selfURL,
	}
}

// ConvertFromLtreeFormat converts data to UUID format from ltree format.
func ConvertFromLtreeFormat(uuid string) string {
	// Ltree allows only "_" as a special character.
	return strings.Replace(uuid, "_", "-", -1)
}

// iterationIDMap contains a map that will hold iteration's ID as its key
type iterationIDMap map[uuid.UUID]*iteration.Iteration

func parentPathResolver(itrMap iterationIDMap) IterationConvertFunc {
	return func(request *goa.RequestData, itr *iteration.Iteration, appIteration *app.Iteration) {
		parentUUIDStrings := strings.Split(iteration.ConvertFromLtreeFormat(itr.ParentPath), iteration.PathSepInService)
		parentUUIDs := convertToUUID(parentUUIDStrings)
		pathResolved := ""
		for _, id := range parentUUIDs {
			if i, ok := itrMap[id]; ok {
				pathResolved += iteration.PathSepInService + i.Name
			}
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

// UpdateIterationsWithCounts accepts map of 'iterationID to a workitem.WICountsPerIteration instance'.
// This function returns function of type IterationConvertFunc
// Inner function is able to access `wiCounts` in closure and it is responsible
// for adding 'closed' and 'total' count of WI in relationship's meta for every given iteration.
func UpdateIterationsWithCounts(wiCounts map[string]workitem.WICountsPerIteration) IterationConvertFunc {
	return func(request *goa.RequestData, itr *iteration.Iteration, appIteration *app.Iteration) {
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
