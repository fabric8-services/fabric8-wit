package main

import (
	"net/http"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/models"
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
	model := models.WorkItemLinkType{}
	in := app.WorkItemLinkType{
		Data: ctx.Payload.Data,
	}
	err := models.ConvertLinkTypeToModel(&in, &model)
	if err != nil {
		return ctx.ResponseData.Service.Send(ctx.Context, http.StatusBadRequest, goa.ErrBadRequest(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		cat, err := appl.WorkItemLinkTypes().Create(ctx.Context, &model)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ConvertErrorFromModelToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		ctx.ResponseData.Header().Set("Location", app.WorkItemLinkTypeHref(cat.Data.ID))
		return ctx.Created(cat)
	})
	// WorkItemLinkTypeController_Create: end_implement
}

// Delete runs the delete action.
func (c *WorkItemLinkTypeController) Delete(ctx *app.DeleteWorkItemLinkTypeContext) error {
	// WorkItemLinkTypeController_Delete: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		err := appl.WorkItemLinkTypes().Delete(ctx.Context, ctx.ID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ConvertErrorFromModelToJSONAPIErrors(err)
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
			jerrors, httpStatusCode := jsonapi.ConvertErrorFromModelToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
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
			jerrors, httpStatusCode := jsonapi.ConvertErrorFromModelToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		return ctx.OK(res)
	})
	// WorkItemLinkTypeController_Show: end_implement
}

// Update runs the update action.
func (c *WorkItemLinkTypeController) Update(ctx *app.UpdateWorkItemLinkTypeContext) error {
	// WorkItemLinkTypeController_Update: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		toSave := app.WorkItemLinkType{
			Data: ctx.Payload.Data,
		}
		linkType, err := appl.WorkItemLinkTypes().Save(ctx.Context, toSave)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ConvertErrorFromModelToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		return ctx.OK(linkType)
	})
	// WorkItemLinkTypeController_Update: end_implement
}
