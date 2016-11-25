package main

import (
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/goadesign/goa"
)

// WorkitemtypeController implements the workitemtype resource.
type WorkitemtypeController struct {
	*goa.Controller
	db application.DB
}

// NewWorkitemtypeController creates a workitemtype controller.
func NewWorkitemtypeController(service *goa.Service, db application.DB) *WorkitemtypeController {
	return &WorkitemtypeController{
		Controller: service.NewController("WorkitemtypeController"),
		db:         db,
	}
}

// Show runs the show action.
func (c *WorkitemtypeController) Show(ctx *app.ShowWorkitemtypeContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		res, err := appl.WorkItemTypes().Load(ctx.Context, ctx.Name)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ConvertErrorFromModelToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		return ctx.OK(res)
	})
}

// Create runs the create action.
func (c *WorkitemtypeController) Create(ctx *app.CreateWorkitemtypeContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		var fields = map[string]app.FieldDefinition{}

		for key, fd := range ctx.Payload.Fields {
			fields[key] = *fd
		}
		wit, err := appl.WorkItemTypes().Create(ctx.Context, ctx.Payload.ExtendedTypeName, ctx.Payload.Name, fields)

		if err != nil {
			jerrors, httpStatusCode := jsonapi.ConvertErrorFromModelToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		ctx.ResponseData.Header().Set("Location", app.WorkitemtypeHref(wit.Name))
		return ctx.Created(wit)
	})
}

// List runs the list action
func (c *WorkitemtypeController) List(ctx *app.ListWorkitemtypeContext) error {
	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		jerrors, _ := jsonapi.ConvertErrorFromModelToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("could not parse paging: %s", err.Error())))
		return ctx.BadRequest(jerrors)
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.WorkItemTypes().List(ctx.Context, start, &limit)
		if err != nil {
			jerrors, _ := jsonapi.ConvertErrorFromModelToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error listing work item types: %s", err.Error())))
			return ctx.BadRequest(jerrors)
		}
		return ctx.OK(result)
	})
}
