package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/workitem/link"
	"github.com/goadesign/goa"
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
	return application.Transactional(c.db, func(appl application.Application) error {
		linkType, err := appl.WorkItemLinkTypes().Create(ctx.Context, model.Name, model.Description, model.SourceTypeName, model.TargetTypeName, model.ForwardName, model.ReverseName, model.Topology, model.LinkCategoryID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		// Now include the optional link category data in the work item link type "included" array
		linkCat, err := appl.WorkItemLinkCategories().Load(ctx.Context, linkType.Data.Relationships.LinkCategory.Data.ID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		linkType.Included = append(linkType.Included, linkCat.Data)

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
		// Build our "set" of distinct category IDs already converted as strings
		categoryIDMap := map[string]bool{}
		for _, typeData := range result.Data {
			categoryIDMap[typeData.Relationships.LinkCategory.Data.ID] = true
		}
		// Now include the optional link category data in the work item link type "included" array
		for categoryID := range categoryIDMap {
			linkCat, err := appl.WorkItemLinkCategories().Load(ctx.Context, categoryID)
			if err != nil {
				jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
				return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
			}
			result.Included = append(result.Included, linkCat.Data)
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

		// Now include the optional link category data in the work item link type "included" array
		linkCat, err := appl.WorkItemLinkCategories().Load(ctx.Context, res.Data.Relationships.LinkCategory.Data.ID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		res.Included = append(res.Included, linkCat.Data)
		res.Links = &app.WorkItemLinkTypeLinks{
			Self: AbsoluteURL(ctx.RequestData, app.WorkItemLinkTypeHref(ctx.ID)),
		}
		return ctx.OK(res)
	})
	// WorkItemLinkTypeController_Show: end_implement
}

// Update runs the update action.
func (c *WorkItemLinkTypeController) Update(ctx *app.UpdateWorkItemLinkTypeContext) error {
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
		// Now include the optional link category data in the work item link type "included" array
		linkCat, err := appl.WorkItemLinkCategories().Load(ctx.Context, linkType.Data.Relationships.LinkCategory.Data.ID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		linkType.Included = append(linkType.Included, linkCat.Data)
		return ctx.OK(linkType)
	})
	// WorkItemLinkTypeController_Update: end_implement
}
