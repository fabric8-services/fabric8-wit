package controller

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/rest"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// SpaceAreasController implements the space-Areas resource.
type SpaceAreasController struct {
	*goa.Controller
	db application.DB
}

// NewSpaceAreasController creates a space-Areas controller.
func NewSpaceAreasController(service *goa.Service, db application.DB) *SpaceAreasController {
	return &SpaceAreasController{Controller: service.NewController("SpaceAreasController"), db: db}
}

// Create runs the create action.
func (c *SpaceAreasController) Create(ctx *app.CreateSpaceAreasContext) error {

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

		newArea := area.Area{
			SpaceID: spaceID,
			Name:    *reqIter.Attributes.Name,
		}

		err = appl.Areas().Create(ctx, &newArea)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.AreaSingle{
			Data: ConvertArea(appl, ctx.RequestData, &newArea, addResolvedPath),
		}
		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.RequestData, app.AreaHref(res.Data.ID)))
		return ctx.Created(res)
	})
}

// List runs the list action.
func (c *SpaceAreasController) List(ctx *app.ListSpaceAreasContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {

		_, err = appl.Spaces().Load(ctx, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}

		areas, err := appl.Areas().List(ctx, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.AreaList{}
		res.Data = ConvertAreas(appl, ctx.RequestData, areas, addResolvedPath)

		return ctx.OK(res)
	})
}
