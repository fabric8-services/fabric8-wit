package main

import (
	"log"

	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/models"
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
			switch err.(type) {
			case models.NotFoundError:
				log.Printf("not found, id=%s", ctx.Name)
				return goa.ErrNotFound(err.Error())
			default:
				return err
			}
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
			switch err := err.(type) {
			case models.BadParameterError, models.ConversionError:
				return goa.ErrBadRequest(err.Error())
			default:
				return goa.ErrInternal(err.Error())
			}
		}
		ctx.ResponseData.Header().Set("Location", app.WorkitemtypeHref(wit.Name))
		return ctx.Created(wit)
	})
}

// List runs the list action
func (c *WorkitemtypeController) List(ctx *app.ListWorkitemtypeContext) error {
	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		return goa.ErrBadRequest(fmt.Sprintf("could not parse paging: %s", err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.WorkItemTypes().List(ctx.Context, start, &limit)
		if err != nil {
			return goa.ErrInternal(fmt.Sprintf("Error listing work item types: %s", err.Error()))
		}
		return ctx.OK(result)
	})
}
