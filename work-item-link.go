package main

import (
	"golang.org/x/net/context"

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

// getTypesOfLinks returns an array of distinct work item link types for the given work item links
func getTypesOfLinks(appl application.Application, ctx context.Context, linksDataArr []*app.WorkItemLinkData) ([]*app.WorkItemLinkTypeData, error) {
	// Build our "set" of distinct type IDs already converted as strings
	typeIDMap := map[string]bool{}
	for _, linkData := range linksDataArr {
		typeIDMap[linkData.Relationships.LinkType.Data.ID] = true
	}
	// Now include the optional link type data in the work item link "included" array
	typeDataArr := []*app.WorkItemLinkTypeData{}
	for typeID := range typeIDMap {
		linkType, err := appl.WorkItemLinkTypes().Load(ctx, typeID)
		if err != nil {
			return nil, err
		}
		typeDataArr = append(typeDataArr, linkType.Data)
	}
	return typeDataArr, nil
}

// enrichLinkWithType includes the optional link type data in the work item link "included" array
func enrichLinkWithType(appl application.Application, ctx context.Context, link *app.WorkItemLink) error {
	linkType, err := appl.WorkItemLinkTypes().Load(ctx, link.Data.Relationships.LinkType.Data.ID)
	if err != nil {
		return err
	}
	link.Included = append(link.Included, linkType.Data)
	return nil
}

// enrichLinkArrayWithTypes includes distinct work item link types in the "included" element
func enrichLinkArrayWithTypes(appl application.Application, ctx context.Context, linkArr *app.WorkItemLinkArray) error {
	typeDataArr, err := getTypesOfLinks(appl, ctx, linkArr.Data)
	if err != nil {
		return err
	}
	linkArr.Included = append(linkArr.Included, typeDataArr...)
	return nil
}

type createWorkItemLinkFuncs interface {
	BadRequest(r *app.JSONAPIErrors) error
	Created(r *app.WorkItemLink) error
}

func createWorkItemLink(ctx context.Context, db application.DB, responseData *goa.ResponseData, funcs createWorkItemLinkFuncs, wiIDStr *string, payload *app.CreateWorkItemLinkPayload) error {
	// Convert payload from app to model representation
	model := models.WorkItemLink{}
	in := app.WorkItemLink{
		Data: payload.Data,
	}
	err := models.ConvertLinkToModel(in, &model)
	if err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return responseData.Service.Send(ctx, httpStatusCode, jerrors)
	}
	return application.Transactional(db, func(appl application.Application) error {
		link, err := appl.WorkItemLinks().Create(ctx, wiIDStr, model.SourceID, model.TargetID, model.LinkTypeID)
		if err != nil {
			switch err.(type) {
			case models.NotFoundError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(err.Error()))
				return funcs.BadRequest(jerrors)
			default:
				jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
				return responseData.Service.Send(ctx, httpStatusCode, jerrors)
			}
		}
		if err := enrichLinkWithType(appl, ctx, link); err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return responseData.Service.Send(ctx, httpStatusCode, jerrors)
		}
		responseData.Header().Set("Location", app.WorkItemLinkHref(link.Data.ID))
		return funcs.Created(link)
	})
}

// Create runs the create action.
func (c *WorkItemLinkController) Create(ctx *app.CreateWorkItemLinkContext) error {
	return createWorkItemLink(ctx.Context, c.db, ctx.ResponseData, ctx, nil, ctx.Payload)
}

type deleteWorkItemLinkFuncs interface {
	OK(resp []byte) error
}

func deleteWorkItemLink(ctx context.Context, db application.DB, responseData *goa.ResponseData, funcs deleteWorkItemLinkFuncs, wiIDStr *string, linkID string) error {
	return application.Transactional(db, func(appl application.Application) error {
		err := appl.WorkItemLinks().Delete(ctx, wiIDStr, linkID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return responseData.Service.Send(ctx, httpStatusCode, jerrors)
		}
		return funcs.OK([]byte{})
	})
}

// Delete runs the delete action
func (c *WorkItemLinkController) Delete(ctx *app.DeleteWorkItemLinkContext) error {
	return deleteWorkItemLink(ctx.Context, c.db, ctx.ResponseData, ctx, nil, ctx.LinkID)
}

type listWorkItemLinkFuncs interface {
	OK(r *app.WorkItemLinkArray) error
}

func listWorkItemLink(ctx context.Context, db application.DB, responseData *goa.ResponseData, funcs listWorkItemLinkFuncs, wiIDStr *string) error {
	return application.Transactional(db, func(appl application.Application) error {
		linkArr, err := appl.WorkItemLinks().List(ctx, wiIDStr)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return responseData.Service.Send(ctx, httpStatusCode, jerrors)
		}
		if err := enrichLinkArrayWithTypes(appl, ctx, linkArr); err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return responseData.Service.Send(ctx, httpStatusCode, jerrors)
		}
		return funcs.OK(linkArr)
	})
}

// List runs the list action.
func (c *WorkItemLinkController) List(ctx *app.ListWorkItemLinkContext) error {
	return listWorkItemLink(ctx.Context, c.db, ctx.ResponseData, ctx, nil)
}

type showWorkItemLinkFuncs interface {
	OK(r *app.WorkItemLink) error
}

func showWorkItemLink(ctx context.Context, db application.DB, responseData *goa.ResponseData, funcs showWorkItemLinkFuncs, wiIDStr *string, linkID string) error {
	return application.Transactional(db, func(appl application.Application) error {
		link, err := appl.WorkItemLinks().Load(ctx, wiIDStr, linkID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return responseData.Service.Send(ctx, httpStatusCode, jerrors)
		}
		if err := enrichLinkWithType(appl, ctx, link); err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return responseData.Service.Send(ctx, httpStatusCode, jerrors)
		}
		return funcs.OK(link)
	})
}

// Show runs the show action.
func (c *WorkItemLinkController) Show(ctx *app.ShowWorkItemLinkContext) error {
	return showWorkItemLink(ctx.Context, c.db, ctx.ResponseData, ctx, nil, ctx.LinkID)
}

type updateWorkItemLinkFuncs interface {
	OK(r *app.WorkItemLink) error
}

func updateWorkItemLink(ctx context.Context, db application.DB, responseData *goa.ResponseData, funcs updateWorkItemLinkFuncs, wiIDStr *string, payload *app.UpdateWorkItemLinkPayload) error {
	return application.Transactional(db, func(appl application.Application) error {
		toSave := app.WorkItemLink{
			Data: payload.Data,
		}
		link, err := appl.WorkItemLinks().Save(ctx, wiIDStr, toSave)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return responseData.Service.Send(ctx, httpStatusCode, jerrors)
		}
		if err := enrichLinkWithType(appl, ctx, link); err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return responseData.Service.Send(ctx, httpStatusCode, jerrors)
		}
		return funcs.OK(link)
	})
}

// Update runs the update action.
func (c *WorkItemLinkController) Update(ctx *app.UpdateWorkItemLinkContext) error {
	return updateWorkItemLink(ctx.Context, c.db, ctx.ResponseData, ctx, nil, ctx.Payload)
}
