package controller

import (
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/space"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
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
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		res, err := appl.WorkItemTypes().Load(ctx.Context, spaceID, ctx.WitID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.OK(res)
	})
}

// Create runs the create action.
func (c *WorkitemtypeController) Create(ctx *app.CreateWorkitemtypeContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		var fields = map[string]app.FieldDefinition{}
		for key, fd := range ctx.Payload.Data.Attributes.Fields {
			fields[key] = *fd
		}

		// Set the space to the Payload
		if ctx.Payload.Data != nil && ctx.Payload.Data.Relationships != nil {
			// We overwrite or use the space ID in the URL to set the space of this WI
			spaceSelfURL := rest.AbsoluteURL(goa.ContextRequest(ctx), app.SpaceHref(spaceID.String()))
			ctx.Payload.Data.Relationships.Space = space.NewSpaceRelation(spaceID, spaceSelfURL)
		}

		wit, err := appl.WorkItemTypes().Create(ctx.Context, *ctx.Payload.Data.Relationships.Space.Data.ID, ctx.Payload.Data.ID, ctx.Payload.Data.Attributes.ExtendedTypeName, ctx.Payload.Data.Attributes.Name, ctx.Payload.Data.Attributes.Description, ctx.Payload.Data.Attributes.Icon, fields)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		ctx.ResponseData.Header().Set("Location", app.WorkitemtypeHref(*ctx.Payload.Data.Relationships.Space.Data.ID, wit.Data.ID))
		return ctx.Created(wit)
	})
}

// List runs the list action
func (c *WorkitemtypeController) List(ctx *app.ListWorkitemtypeContext) error {
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

// ListSourceLinkTypes runs the list-source-link-types action.
func (c *WorkitemtypeController) ListSourceLinkTypes(ctx *app.ListSourceLinkTypesWorkitemtypeContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		// Test that work item type exists
		_, err := appl.WorkItemTypes().Load(ctx.Context, spaceID, ctx.WitID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		// Fetch all link types where this work item type can be used in the
		// source of the link
		res, err := appl.WorkItemLinkTypes().ListSourceLinkTypes(ctx.Context, ctx.WitID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		// Enrich link types
		hrefFunc := func(obj interface{}) string {
			return fmt.Sprintf(app.WorkItemLinkTypeHref(spaceID, "%v"), obj)
		}
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, hrefFunc, nil)
		err = enrichLinkTypeList(linkCtx, res)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.OK(res)
	})
}

// ListTargetLinkTypes runs the list-target-link-types action.
func (c *WorkitemtypeController) ListTargetLinkTypes(ctx *app.ListTargetLinkTypesWorkitemtypeContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		// Test that work item type exists
		_, err := appl.WorkItemTypes().Load(ctx.Context, spaceID, ctx.WitID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		// Fetch all link types where this work item type can be used in the
		// target of the linkg
		res, err := appl.WorkItemLinkTypes().ListTargetLinkTypes(ctx.Context, ctx.WitID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		// Enrich link types
		hrefFunc := func(obj interface{}) string {
			return fmt.Sprintf(app.WorkItemLinkTypeHref(spaceID, "%v"), obj)
		}
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, hrefFunc, nil)
		err = enrichLinkTypeList(linkCtx, res)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.OK(res)
	})
}
