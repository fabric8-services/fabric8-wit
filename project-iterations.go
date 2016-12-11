package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// ProjectIterationsController implements the project-iterations resource.
type ProjectIterationsController struct {
	*goa.Controller
	db application.DB
}

// NewProjectIterationsController creates a project-iterations controller.
func NewProjectIterationsController(service *goa.Service, db application.DB) *ProjectIterationsController {
	return &ProjectIterationsController{Controller: service.NewController("ProjectIterationsController"), db: db}
}

// Create runs the create action.
func (c *ProjectIterationsController) Create(ctx *app.CreateProjectIterationsContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	projectID, err := uuid.FromString(ctx.ID)
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
		_, err = appl.Projects().Load(ctx, projectID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}

		newItr := iteration.Iteration{
			ProjectID: projectID,
			Name:      *reqIter.Attributes.Name,
			StartAt:   reqIter.Attributes.StartAt,
			EndAt:     reqIter.Attributes.EndAt,
		}

		err = appl.Iterations().Create(ctx, &newItr)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.IterationSingle{
			Data: ConvertIteration(ctx.RequestData, &newItr),
		}
		ctx.ResponseData.Header().Set("Location", AbsoluteURL(ctx.RequestData, app.IterationHref(res.Data.ID)))
		return ctx.Created(res)
	})
}

// List runs the list action.
func (c *ProjectIterationsController) List(ctx *app.ListProjectIterationsContext) error {
	projectID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {

		_, err = appl.Projects().Load(ctx, projectID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}

		iterations, err := appl.Iterations().List(ctx, projectID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.IterationList{}
		res.Data = ConvertIterations(ctx.RequestData, iterations)

		return ctx.OK(res)
	})
}
