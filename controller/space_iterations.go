package controller

import (
	"github.com/Sirupsen/logrus"
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

// SpaceIterationsControllerConfiguration configuration for the SpaceIterationsController

type SpaceIterationsControllerConfiguration interface {
	GetCacheControlIteration() string
}

// SpaceIterationsController implements the space-iterations resource.
type SpaceIterationsController struct {
	*goa.Controller
	db     application.DB
	config SpaceIterationsControllerConfiguration
}

// NewSpaceIterationsController creates a space-iterations controller.
func NewSpaceIterationsController(service *goa.Service, db application.DB, config SpaceIterationsControllerConfiguration) *SpaceIterationsController {
	return &SpaceIterationsController{Controller: service.NewController("SpaceIterationsController"), db: db, config: config}
}

// Create runs the create action.
func (c *SpaceIterationsController) Create(ctx *app.CreateSpaceIterationsContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	// Validate Request
	if ctx.Payload.Data == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data", nil).Expected("not nil"))
	}
	reqIter := ctx.Payload.Data
	if reqIter.Attributes.Name == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil"))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		_, err = appl.Spaces().Load(ctx, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		// Put iteration under root iteration
		rootIteration, err := appl.Iterations().Root(ctx, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		childPath := append(rootIteration.Path, rootIteration.ID)
		newItr := iteration.Iteration{
			SpaceID: spaceID,
			Name:    *reqIter.Attributes.Name,
			StartAt: reqIter.Attributes.StartAt,
			EndAt:   reqIter.Attributes.EndAt,
			Path:    childPath,
		}
		if reqIter.Attributes.Description != nil {
			newItr.Description = reqIter.Attributes.Description
		}
		err = appl.Iterations().Create(ctx, &newItr)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		// For create, count will always be zero hence no need to query
		// by passing empty map, updateIterationsWithCounts will be able to put zero values
		wiCounts := make(map[string]workitem.WICountsPerIteration)
		logrus.Info("wicounts for created iteration ", newItr.ID.String(), " -> ", wiCounts)

		var responseData *app.Iteration
		if newItr.Path.IsEmpty() == false {
			allParentsUUIDs := newItr.Path
			iterations, error := appl.Iterations().LoadMultiple(ctx, allParentsUUIDs)
			if error != nil {
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			itrMap := make(iterationIDMap)
			for _, itr := range iterations {
				itrMap[itr.ID] = itr
			}
			responseData = ConvertIteration(ctx.RequestData, newItr, parentPathResolver(itrMap), updateIterationsWithCounts(wiCounts))
		} else {
			responseData = ConvertIteration(ctx.RequestData, newItr, updateIterationsWithCounts(wiCounts))
		}
		res := &app.IterationSingle{
			Data: responseData,
		}
		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.RequestData, app.IterationHref(res.Data.ID)))
		return ctx.Created(res)
	})
}

// List runs the list action.
func (c *SpaceIterationsController) List(ctx *app.ListSpaceIterationsContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {

		_, err = appl.Spaces().Load(ctx, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		iterations, err := appl.Iterations().List(ctx, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalEntities(iterations, c.config.GetCacheControlIteration, func() error {
			itrMap := make(iterationIDMap)
			for _, itr := range iterations {
				itrMap[itr.ID] = itr
			}
			// fetch extra information(counts of WI in each iteration of the space) to be added in response
			wiCounts, err := appl.WorkItems().GetCountsPerIteration(ctx, spaceID)
			logrus.Info("Retrieving wicounts for spaceID ", spaceID.String(), " -> ", wiCounts)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			res := &app.IterationList{}
			res.Data = ConvertIterations(ctx.RequestData, iterations, updateIterationsWithCounts(wiCounts), parentPathResolver(itrMap))
			return ctx.OK(res)
		})
	})
}
