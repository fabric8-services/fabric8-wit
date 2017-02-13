package controllers

import (
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/goadesign/goa"
)

const (
	sourceLinkTypesRouteEnd = "/source-link-types"
	targetLinkTypesRouteEnd = "/target-link-types"
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
			return jsonapi.JSONErrorResponse(ctx, err)
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
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
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
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("could not parse paging: %s", err.Error())))
		return ctx.BadRequest(jerrors)
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.WorkItemTypes().List(ctx.Context, start, &limit)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error listing work item types: %s", err.Error())))
			return ctx.BadRequest(jerrors)
		}
		return ctx.OK(result)
	})
}

// ListSourceLinkTypes runs the list-source-link-types action.
func (c *WorkitemtypeController) ListSourceLinkTypes(ctx *app.ListSourceLinkTypesWorkitemtypeContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		// Test that work item type exists
		_, err := appl.WorkItemTypes().Load(ctx.Context, ctx.Name)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		// Fetch all link types where this work item type can be used in the
		// source of the link
		res, err := appl.WorkItemLinkTypes().ListSourceLinkTypes(ctx.Context, ctx.Name)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		// Enrich link types
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkTypeHref)
		err = enrichLinkTypeList(linkCtx, res)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.OK(res)
	})
}

// ListTargetLinkTypes runs the list-target-link-types action.
func (c *WorkitemtypeController) ListTargetLinkTypes(ctx *app.ListTargetLinkTypesWorkitemtypeContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		// Test that work item type exists
		_, err := appl.WorkItemTypes().Load(ctx.Context, ctx.Name)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		// Fetch all link types where this work item type can be used in the
		// target of the linkg
		res, err := appl.WorkItemLinkTypes().ListTargetLinkTypes(ctx.Context, ctx.Name)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		// Enrich link types
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkTypeHref)
		err = enrichLinkTypeList(linkCtx, res)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.OK(res)
	})
}
