package main

import (
	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/workitem/link"
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

// getTypesOfLinks returns an array of distinct work item link types for the
// given work item links
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

// getWorkItemsOfLinks returns an array of distinct work items as they appear as
// source or target in the given work item links.
func getWorkItemsOfLinks(appl application.Application, ctx context.Context, linksDataArr []*app.WorkItemLinkData) ([]*app.WorkItem2, error) {
	// Build our "set" of distinct work item IDs already converted as strings
	workItemIDMap := map[string]bool{}
	for _, linkData := range linksDataArr {
		workItemIDMap[linkData.Relationships.Source.Data.ID] = true
		workItemIDMap[linkData.Relationships.Target.Data.ID] = true
	}
	// Now include the optional work item data in the work item link "included" array
	workItemArr := []*app.WorkItem2{}
	for workItemID := range workItemIDMap {
		wi, err := appl.WorkItems().Load(ctx, workItemID)
		if err != nil {
			return nil, err
		}
		workItemArr = append(workItemArr, ConvertWorkItemToJSONAPI(wi))
	}
	return workItemArr, nil
}

// getCategoriesOfLinkTypes returns an array of distinct work item link
// categories for the given work item link types
func getCategoriesOfLinkTypes(appl application.Application, ctx context.Context, linkTypeDataArr []*app.WorkItemLinkTypeData) ([]*app.WorkItemLinkCategoryData, error) {
	// Build our "set" of distinct category IDs already converted as strings
	catIDMap := map[string]bool{}
	for _, linkTypeData := range linkTypeDataArr {
		catIDMap[linkTypeData.Relationships.LinkCategory.Data.ID] = true
	}
	// Now include the optional link category data in the work item link "included" array
	catDataArr := []*app.WorkItemLinkCategoryData{}
	for catID := range catIDMap {
		linkType, err := appl.WorkItemLinkCategories().Load(ctx, catID)
		if err != nil {
			return nil, err
		}
		catDataArr = append(catDataArr, linkType.Data)
	}
	return catDataArr, nil
}

// enrichLink includes related resources in the link's "included" array
func enrichLink(appl application.Application, ctx context.Context, link *app.WorkItemLink) error {

	// include link type
	linkType, err := appl.WorkItemLinkTypes().Load(ctx, link.Data.Relationships.LinkType.Data.ID)
	if err != nil {
		return err
	}
	link.Included = append(link.Included, linkType.Data)

	// include link category
	linkCat, err := appl.WorkItemLinkCategories().Load(ctx, linkType.Data.Relationships.LinkCategory.Data.ID)
	if err != nil {
		return err
	}
	link.Included = append(link.Included, linkCat.Data)

	// TODO(kwk): include source work item type (once #559 is merged)
	// sourceWit, err := appl.WorkItemTypes().Load(ctx, linkType.Data.Relationships.SourceType.Data.ID)
	// if err != nil {
	// 	return err
	// }
	// link.Included = append(link.Included, sourceWit.Data)

	// TODO(kwk): include target work item type (once #559 is merged)
	// targetWit, err := appl.WorkItemTypes().Load(ctx, linkType.Data.Relationships.TargetType.Data.ID)
	// if err != nil {
	// 	return err
	// }
	// link.Included = append(link.Included, targetWit.Data)

	// TODO(kwk): include source work item
	sourceWi, err := appl.WorkItems().Load(ctx, link.Data.Relationships.Source.Data.ID)
	if err != nil {
		return err
	}
	link.Included = append(link.Included, ConvertWorkItemToJSONAPI(sourceWi))

	// TODO(kwk): include target work item
	targetWi, err := appl.WorkItems().Load(ctx, link.Data.Relationships.Target.Data.ID)
	if err != nil {
		return err
	}
	link.Included = append(link.Included, ConvertWorkItemToJSONAPI(targetWi))

	return nil
}

// enrichLinkArray includes related resources in the linkArr's "included" element
func enrichLinkArray(appl application.Application, ctx context.Context, linkArr *app.WorkItemLinkArray) error {

	// include link types
	typeDataArr, err := getTypesOfLinks(appl, ctx, linkArr.Data)
	if err != nil {
		return err
	}
	// Convert slice of objects to slice of interface (see https://golang.org/doc/faq#convert_slice_of_interface)
	interfaceArr := make([]interface{}, len(typeDataArr))
	for i, v := range typeDataArr {
		interfaceArr[i] = v
	}
	linkArr.Included = append(linkArr.Included, interfaceArr...)

	// include link categories
	catDataArr, err := getCategoriesOfLinkTypes(appl, ctx, typeDataArr)
	if err != nil {
		return err
	}
	// Convert slice of objects to slice of interface (see https://golang.org/doc/faq#convert_slice_of_interface)
	interfaceArr = make([]interface{}, len(catDataArr))
	for i, v := range catDataArr {
		interfaceArr[i] = v
	}
	linkArr.Included = append(linkArr.Included, interfaceArr...)

	// TODO(kwk): Include WIs from source and target
	workItemDataArr, err := getWorkItemsOfLinks(appl, ctx, linkArr.Data)
	if err != nil {
		return err
	}
	// Convert slice of objects to slice of interface (see https://golang.org/doc/faq#convert_slice_of_interface)
	interfaceArr = make([]interface{}, len(workItemDataArr))
	for i, v := range workItemDataArr {
		interfaceArr[i] = v
	}
	linkArr.Included = append(linkArr.Included, interfaceArr...)

	// TODO(kwk): Include WITs (once #559 is merged)

	return nil
}

type createWorkItemLinkFuncs interface {
	BadRequest(r *app.JSONAPIErrors) error
	Created(r *app.WorkItemLink) error
}

func createWorkItemLink(appl application.Application, ctx context.Context, db application.DB, responseData *goa.ResponseData, funcs createWorkItemLinkFuncs, payload *app.CreateWorkItemLinkPayload) error {
	// Convert payload from app to model representation
	model := link.WorkItemLink{}
	in := app.WorkItemLink{
		Data: payload.Data,
	}
	err := link.ConvertLinkToModel(in, &model)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(err)
		return funcs.BadRequest(jerrors)
	}
	link, err := appl.WorkItemLinks().Create(ctx, model.SourceID, model.TargetID, model.LinkTypeID)
	if err != nil {
		switch err.(type) {
		case errors.NotFoundError:
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(err.Error()))
			return funcs.BadRequest(jerrors)
		default:
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return responseData.Service.Send(ctx, httpStatusCode, jerrors)
		}
	}
	if err := enrichLink(appl, ctx, link); err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return responseData.Service.Send(ctx, httpStatusCode, jerrors)
	}
	responseData.Header().Set("Location", app.WorkItemLinkHref(link.Data.ID))
	return funcs.Created(link)
}

// Create runs the create action.
func (c *WorkItemLinkController) Create(ctx *app.CreateWorkItemLinkContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		return createWorkItemLink(appl, ctx.Context, c.db, ctx.ResponseData, ctx, ctx.Payload)
	})
}

type deleteWorkItemLinkFuncs interface {
	OK(resp []byte) error
}

func deleteWorkItemLink(appl application.Application, ctx context.Context, db application.DB, responseData *goa.ResponseData, funcs deleteWorkItemLinkFuncs, linkID string) error {
	err := appl.WorkItemLinks().Delete(ctx, linkID)
	if err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return responseData.Service.Send(ctx, httpStatusCode, jerrors)
	}
	return funcs.OK([]byte{})
}

// Delete runs the delete action
func (c *WorkItemLinkController) Delete(ctx *app.DeleteWorkItemLinkContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		return deleteWorkItemLink(appl, ctx.Context, c.db, ctx.ResponseData, ctx, ctx.LinkID)
	})
}

type listWorkItemLinkFuncs interface {
	OK(r *app.WorkItemLinkArray) error
}

func listWorkItemLink(appl application.Application, ctx context.Context, db application.DB, responseData *goa.ResponseData, funcs listWorkItemLinkFuncs, wiIDStr *string) error {
	linkArr, err := appl.WorkItemLinks().List(ctx, wiIDStr)
	if err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return responseData.Service.Send(ctx, httpStatusCode, jerrors)
	}
	if err := enrichLinkArray(appl, ctx, linkArr); err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return responseData.Service.Send(ctx, httpStatusCode, jerrors)
	}
	return funcs.OK(linkArr)
}

// List runs the list action.
func (c *WorkItemLinkController) List(ctx *app.ListWorkItemLinkContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		return listWorkItemLink(appl, ctx.Context, c.db, ctx.ResponseData, ctx, nil)
	})
}

type showWorkItemLinkFuncs interface {
	OK(r *app.WorkItemLink) error
}

func showWorkItemLink(appl application.Application, ctx context.Context, db application.DB, responseData *goa.ResponseData, funcs showWorkItemLinkFuncs, linkID string) error {
	link, err := appl.WorkItemLinks().Load(ctx, linkID)
	if err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return responseData.Service.Send(ctx, httpStatusCode, jerrors)
	}
	if err := enrichLink(appl, ctx, link); err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return responseData.Service.Send(ctx, httpStatusCode, jerrors)
	}
	return funcs.OK(link)
}

// Show runs the show action.
func (c *WorkItemLinkController) Show(ctx *app.ShowWorkItemLinkContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		return showWorkItemLink(appl, ctx.Context, c.db, ctx.ResponseData, ctx, ctx.LinkID)
	})
}

type updateWorkItemLinkFuncs interface {
	OK(r *app.WorkItemLink) error
}

func updateWorkItemLink(appl application.Application, ctx context.Context, db application.DB, responseData *goa.ResponseData, funcs updateWorkItemLinkFuncs, payload *app.UpdateWorkItemLinkPayload) error {
	toSave := app.WorkItemLink{
		Data: payload.Data,
	}
	link, err := appl.WorkItemLinks().Save(ctx, toSave)
	if err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return responseData.Service.Send(ctx, httpStatusCode, jerrors)
	}
	if err := enrichLink(appl, ctx, link); err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return responseData.Service.Send(ctx, httpStatusCode, jerrors)
	}
	return funcs.OK(link)
}

// Update runs the update action.
func (c *WorkItemLinkController) Update(ctx *app.UpdateWorkItemLinkContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		return updateWorkItemLink(appl, ctx.Context, c.db, ctx.ResponseData, ctx, ctx.Payload)
	})
}
