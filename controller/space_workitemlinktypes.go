package controller

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/workitem/link"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// SpaceWorkitemlinktypesController implements the space_workitemlinktypes resource.
type SpaceWorkitemlinktypesController struct {
	*goa.Controller
	db application.DB
}

// NewSpaceWorkitemlinktypesController creates a space_workitemlinktypes controller.
func NewSpaceWorkitemlinktypesController(service *goa.Service, db application.DB) *SpaceWorkitemlinktypesController {
	return &SpaceWorkitemlinktypesController{Controller: service.NewController("SpaceWorkitemlinktypesController"), db: db}
}

// Create runs the create action.
func (c *SpaceWorkitemlinktypesController) Create(ctx *app.CreateSpaceWorkitemlinktypesContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	// WorkItemLinkTypeController_Create: start_implement
	// Convert payload from app to model representation
	model := link.WorkItemLinkType{}
	in := app.WorkItemLinkTypeSingle{
		Data: ctx.Payload.Data,
	}
	err = link.ConvertLinkTypeToModel(in, &model)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(err.Error()))
		return ctx.BadRequest(jerrors)
	}
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		//linkType, err := appl.WorkItemLinkTypes().Create(ctx.Context, model.Name, model.Description, model.SourceTypeID, model.TargetTypeID, model.ForwardName, model.ReverseName, model.Topology, model.LinkCategoryID, model.SpaceID)
		//TODO: It confusing if whether the Space data will come from the ctx or payload
		linkType, err := appl.WorkItemLinkTypes().Create(ctx.Context, model.Name, model.Description, model.SourceTypeID, model.TargetTypeID, model.ForwardName, model.ReverseName, model.Topology, model.LinkCategoryID, spaceID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		// Enrich
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkTypeHref, currentUserIdentityID)
		err = enrichLinkTypeSingle(linkCtx, linkType)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal("Failed to enrich link type: %s", err.Error()))
			return ctx.InternalServerError(jerrors)
		}
		ctx.ResponseData.Header().Set("Location", app.WorkItemLinkTypeHref(linkType.Data.ID))
		return ctx.Created(linkType)
	})
	// WorkItemLinkTypeController_Create: end_implement
}

// List runs the list action.
func (c *SpaceWorkitemlinktypesController) List(ctx *app.ListSpaceWorkitemlinktypesContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	// WorkItemLinkTypeController_List: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.WorkItemLinkTypes().List(ctx.Context, spaceID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		// Enrich
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkTypeHref, nil)
		err = enrichLinkTypeList(linkCtx, result)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal("Failed to enrich link types: %s", err.Error()))
			return ctx.InternalServerError(jerrors)
		}
		return ctx.OK(result)
	})
}

// enrichLinkTypeList includes related resources in the list's "included" array
func enrichLinkTypeList(ctx *workItemLinkContext, list *app.WorkItemLinkTypeList) error {
	// Add "links" element
	for _, data := range list.Data {
		selfURL := rest.AbsoluteURL(ctx.RequestData, ctx.LinkFunc(*data.ID))
		data.Links = &app.GenericLinks{
			Self: &selfURL,
		}
	}
	// Build our "set" of distinct category IDs already converted as strings
	categoryIDMap := map[uuid.UUID]bool{}
	for _, typeData := range list.Data {
		categoryIDMap[typeData.Relationships.LinkCategory.Data.ID] = true
	}
	// Now include the optional link category data in the work item link type "included" array
	for categoryID := range categoryIDMap {
		linkCat, err := ctx.Application.WorkItemLinkCategories().Load(ctx.Context, categoryID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		list.Included = append(list.Included, linkCat.Data)
	}

	// Build our "set" of distinct space IDs already converted as strings
	spaceIDMap := map[uuid.UUID]bool{}
	for _, typeData := range list.Data {
		spaceIDMap[*typeData.Relationships.Space.Data.ID] = true
	}
	// Now include the optional link space data in the work item link type "included" array
	for spaceID := range spaceIDMap {
		space, err := ctx.Application.Spaces().Load(ctx.Context, spaceID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		spaceSingle := &app.SpaceSingle{
			Data: ConvertSpace(ctx.RequestData, space),
		}
		list.Included = append(list.Included, spaceSingle.Data)
	}
	return nil
}
