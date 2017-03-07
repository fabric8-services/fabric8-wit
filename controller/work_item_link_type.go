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

// WorkItemLinkTypeController implements the work-item-link-type resource.
type WorkItemLinkTypeController struct {
	*goa.Controller
	db application.DB
}

// NewWorkItemLinkTypeController creates a work-item-link-type controller.
func NewWorkItemLinkTypeController(service *goa.Service, db application.DB) *WorkItemLinkTypeController {
	if db == nil {
		panic("db must not be nil")
	}
	return &WorkItemLinkTypeController{
		Controller: service.NewController("WorkItemLinkTypeController"),
		db:         db,
	}
}

// enrichLinkTypeSingle includes related resources in the single's "included" array
func enrichLinkTypeSingle(ctx *workItemLinkContext, single *app.WorkItemLinkTypeSingle) error {
	// Add "links" element
	selfURL := rest.AbsoluteURL(ctx.RequestData, ctx.LinkFunc(*single.Data.ID))
	single.Data.Links = &app.GenericLinks{
		Self: &selfURL,
	}

	// Now include the optional link category data in the work item link type "included" array
	linkCat, err := ctx.Application.WorkItemLinkCategories().Load(ctx.Context, single.Data.Relationships.LinkCategory.Data.ID)
	if err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)

	}
	single.Included = append(single.Included, linkCat.Data)

	// Now include the optional link space data in the work item link type "included" array
	space, err := ctx.Application.Spaces().Load(ctx.Context, *single.Data.Relationships.Space.Data.ID)
	if err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
	}
	spaceSingle := &app.SpaceSingle{
		Data: ConvertSpace(ctx.RequestData, space),
	}
	single.Included = append(single.Included, spaceSingle.Data)

	return nil
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

// Create runs the create action.
func (c *WorkItemLinkTypeController) Create(ctx *app.CreateWorkItemLinkTypeContext) error {
	// WorkItemLinkTypeController_Create: start_implement
	// Convert payload from app to model representation
	model := link.WorkItemLinkType{}
	in := app.WorkItemLinkTypeSingle{
		Data: ctx.Payload.Data,
	}
	err := link.ConvertLinkTypeToModel(in, &model)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(err.Error()))
		return ctx.BadRequest(jerrors)
	}
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		linkType, err := appl.WorkItemLinkTypes().Create(ctx.Context, model.Name, model.Description, model.SourceTypeID, model.TargetTypeID, model.ForwardName, model.ReverseName, model.Topology, model.LinkCategoryID, model.SpaceID)
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

// Delete runs the delete action.
func (c *WorkItemLinkTypeController) Delete(ctx *app.DeleteWorkItemLinkTypeContext) error {
	// WorkItemLinkTypeController_Delete: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		err := appl.WorkItemLinkTypes().Delete(ctx.Context, ctx.ID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		return ctx.OK([]byte{})
	})
	// WorkItemLinkTypeController_Delete: end_implement
}

// List runs the list action.
func (c *WorkItemLinkTypeController) List(ctx *app.ListWorkItemLinkTypeContext) error {
	// WorkItemLinkTypeController_List: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.WorkItemLinkTypes().List(ctx.Context)
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
	// WorkItemLinkTypeController_List: end_implement
}

// Show runs the show action.
func (c *WorkItemLinkTypeController) Show(ctx *app.ShowWorkItemLinkTypeContext) error {
	// WorkItemLinkTypeController_Show: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		res, err := appl.WorkItemLinkTypes().Load(ctx.Context, ctx.ID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		// Enrich
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkTypeHref, nil)
		err = enrichLinkTypeSingle(linkCtx, res)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal("Failed to enrich link type: %s", err.Error()))
			return ctx.InternalServerError(jerrors)
		}
		return ctx.OK(res)
	})
	// WorkItemLinkTypeController_Show: end_implement
}

// Update runs the update action.
func (c *WorkItemLinkTypeController) Update(ctx *app.UpdateWorkItemLinkTypeContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	// WorkItemLinkTypeController_Update: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		toSave := app.WorkItemLinkTypeSingle{
			Data: ctx.Payload.Data,
		}
		linkType, err := appl.WorkItemLinkTypes().Save(ctx.Context, toSave)
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
		return ctx.OK(linkType)
	})
	// WorkItemLinkTypeController_Update: end_implement
}
