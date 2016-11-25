package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/models"
	"github.com/goadesign/goa"
)

// WorkItemLinkController implements the work-item-link resource.
type WorkItemLinkController struct {
	*goa.Controller
	db application.DB
}

// NewWorkItemLinkController creates a work-item-link controller.
func NewWorkItemLinkController(service *goa.Service, db application.DB) *WorkItemLinkController {
	if db == nil {
		panic("db must not be nil")
	}
	return &WorkItemLinkController{
		Controller: service.NewController("WorkItemLinkController"),
		db:         db,
	}
}

// Create runs the create action.
func (c *WorkItemLinkController) Create(ctx *app.CreateWorkItemLinkContext) error {
	// WorkItemLinkController_Create: start_implement
	// Convert payload from app to model representation
	model := models.WorkItemLink{}
	in := app.WorkItemLink{
		Data: ctx.Payload.Data,
	}
	err := models.ConvertLinkToModel(&in, &model)
	if err != nil {
		jerrors, httpStatusCode := jsonapi.ConvertErrorFromModelToJSONAPIErrors(err)
		return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		cat, err := appl.WorkItemLinks().Create(ctx.Context, model.SourceID, model.TargetID, model.LinkTypeID)
		if err != nil {
			switch err.(type) {
			case models.NotFoundError:
				jerrors, _ := jsonapi.ConvertErrorFromModelToJSONAPIErrors(goa.ErrBadRequest(err.Error()))
				return ctx.BadRequest(jerrors)
			default:
				jerrors, httpStatusCode := jsonapi.ConvertErrorFromModelToJSONAPIErrors(err)
				return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
			}
		}
		ctx.ResponseData.Header().Set("Location", app.WorkItemLinkHref(cat.Data.ID))
		return ctx.Created(cat)
	})
	// WorkItemLinkController_Create: end_implement
}

// Delete runs the delete action.
func (c *WorkItemLinkController) Delete(ctx *app.DeleteWorkItemLinkContext) error {
	// WorkItemLinkController_Delete: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		err := appl.WorkItemLinks().Delete(ctx.Context, ctx.ID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ConvertErrorFromModelToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		return ctx.OK([]byte{})
	})
	// WorkItemLinkController_Delete: end_implement
}

// List runs the list action.
func (c *WorkItemLinkController) List(ctx *app.ListWorkItemLinkContext) error {
	// WorkItemLinkController_List: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.WorkItemLinks().List(ctx.Context)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ConvertErrorFromModelToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		return ctx.OK(result)
	})
	// WorkItemLinkController_List: end_implement
}

// Show runs the show action.
func (c *WorkItemLinkController) Show(ctx *app.ShowWorkItemLinkContext) error {
	// WorkItemLinkController_Show: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		res, err := appl.WorkItemLinks().Load(ctx.Context, ctx.ID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ConvertErrorFromModelToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		return ctx.OK(res)
	})
	// WorkItemLinkController_Show: end_implement
}

// Update runs the update action.
func (c *WorkItemLinkController) Update(ctx *app.UpdateWorkItemLinkContext) error {
	// WorkItemLinkController_Update: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		toSave := app.WorkItemLink{
			Data: ctx.Payload.Data,
		}
		linkType, err := appl.WorkItemLinks().Save(ctx.Context, toSave)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ConvertErrorFromModelToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		return ctx.OK(linkType)
	})
	// WorkItemLinkController_Update: end_implement
}
