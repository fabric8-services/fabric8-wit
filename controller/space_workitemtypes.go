package controller

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// SpaceWorkitemtypesController implements the space_workitemtypes resource.
type SpaceWorkitemtypesController struct {
	*goa.Controller
	db application.DB
}

// NewSpaceWorkitemtypesController creates a space_workitemtypes controller.
func NewSpaceWorkitemtypesController(service *goa.Service, db application.DB) *SpaceWorkitemtypesController {
	return &SpaceWorkitemtypesController{Controller: service.NewController("SpaceWorkitemtypesController"), db: db}
}

// Create runs the create action.
func (c *SpaceWorkitemtypesController) Create(ctx *app.CreateSpaceWorkitemtypesContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	//TODO: It confusing if whether the Space data will come from the ctx or payload

	return application.Transactional(c.db, func(appl application.Application) error {
		var fields = map[string]app.FieldDefinition{}
		for key, fd := range ctx.Payload.Data.Attributes.Fields {
			fields[key] = *fd
		}
		//wit, err := appl.WorkItemTypes().Create(ctx.Context, *ctx.Payload.Data.Relationships.Space.Data.ID, ctx.Payload.Data.ID, ctx.Payload.Data.Attributes.ExtendedTypeName, ctx.Payload.Data.Attributes.Name, ctx.Payload.Data.Attributes.Description, ctx.Payload.Data.Attributes.Icon, fields)
		wit, err := appl.WorkItemTypes().Create(ctx.Context, spaceID, ctx.Payload.Data.ID, ctx.Payload.Data.Attributes.ExtendedTypeName, ctx.Payload.Data.Attributes.Name, ctx.Payload.Data.Attributes.Description, ctx.Payload.Data.Attributes.Icon, fields)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		ctx.ResponseData.Header().Set("Location", app.WorkitemtypeHref(wit.Data.ID))
		return ctx.Created(wit)
	})
}

// List runs the list action.
func (c *SpaceWorkitemtypesController) List(ctx *app.ListSpaceWorkitemtypesContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Could not parse paging"))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.WorkItemTypes().List(ctx.Context, spaceID, start, &limit)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Error listing work item types"))
		}
		return ctx.OK(result)
	})
}
