package controller

import (
	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/workitem/link"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
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

// Instead of using app.WorkItemLinkHref directly, we store a function object
// inside the WorkItemLinkContext in order to generate proper links nomatter
// from which controller the context is being used. For example one could use
// app.WorkItemRelationshipsLinksHref.
type hrefLinkFunc func(obj interface{}) string

// workItemLinkContext bundles objects that are needed by most of the functions.
// It can easily be extended.
type workItemLinkContext struct {
	RequestData  *goa.RequestData
	ResponseData *goa.ResponseData
	Application  application.Application
	Context      context.Context
	DB           application.DB
	LinkFunc     hrefLinkFunc
}

// newWorkItemLinkContext returns a new workItemLinkContext
func newWorkItemLinkContext(ctx context.Context, appl application.Application, db application.DB, requestData *goa.RequestData, responseData *goa.ResponseData, linkFunc hrefLinkFunc) *workItemLinkContext {
	return &workItemLinkContext{
		RequestData:  requestData,
		ResponseData: responseData,
		Application:  appl,
		Context:      ctx,
		DB:           db,
		LinkFunc:     linkFunc,
	}
}

// getTypesOfLinks returns an array of distinct work item link types for the
// given work item links
func getTypesOfLinks(ctx *workItemLinkContext, linksDataArr []*app.WorkItemLinkData) ([]*app.WorkItemLinkTypeData, error) {
	// Build our "set" of distinct type IDs already converted as strings
	typeIDMap := map[string]bool{}
	for _, linkData := range linksDataArr {
		typeIDMap[linkData.Relationships.LinkType.Data.ID] = true
	}
	// Now include the optional link type data in the work item link "included" array
	typeDataArr := []*app.WorkItemLinkTypeData{}
	for typeID := range typeIDMap {
		linkType, err := ctx.Application.WorkItemLinkTypes().Load(ctx.Context, typeID)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		typeDataArr = append(typeDataArr, linkType.Data)
	}
	return typeDataArr, nil
}

// getWorkItemsOfLinks returns an array of distinct work items as they appear as
// source or target in the given work item links.
func getWorkItemsOfLinks(ctx *workItemLinkContext, linksDataArr []*app.WorkItemLinkData) ([]*app.WorkItem2, error) {
	// Build our "set" of distinct work item IDs already converted as strings
	workItemIDMap := map[string]bool{}
	for _, linkData := range linksDataArr {
		workItemIDMap[linkData.Relationships.Source.Data.ID] = true
		workItemIDMap[linkData.Relationships.Target.Data.ID] = true
	}
	// Now include the optional work item data in the work item link "included" array
	workItemArr := []*app.WorkItem2{}
	for workItemID := range workItemIDMap {
		wi, err := ctx.Application.WorkItems().Load(ctx.Context, workItemID)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		workItemArr = append(workItemArr, ConvertWorkItem(ctx.RequestData, wi))
	}
	return workItemArr, nil
}

// getCategoriesOfLinkTypes returns an array of distinct work item link
// categories for the given work item link types
func getCategoriesOfLinkTypes(ctx *workItemLinkContext, linkTypeDataArr []*app.WorkItemLinkTypeData) ([]*app.WorkItemLinkCategoryData, error) {
	// Build our "set" of distinct category IDs already converted as strings
	catIDMap := map[string]bool{}
	for _, linkTypeData := range linkTypeDataArr {
		catIDMap[linkTypeData.Relationships.LinkCategory.Data.ID] = true
	}
	// Now include the optional link category data in the work item link "included" array
	catDataArr := []*app.WorkItemLinkCategoryData{}
	for catID := range catIDMap {
		linkType, err := ctx.Application.WorkItemLinkCategories().Load(ctx.Context, catID)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		catDataArr = append(catDataArr, linkType.Data)
	}
	return catDataArr, nil
}

// enrichLinkSingle includes related resources in the link's "included" array
func enrichLinkSingle(ctx *workItemLinkContext, link *app.WorkItemLinkSingle) error {

	// include link type
	linkType, err := ctx.Application.WorkItemLinkTypes().Load(ctx.Context, link.Data.Relationships.LinkType.Data.ID)
	if err != nil {
		return errs.WithStack(err)
	}
	link.Included = append(link.Included, linkType.Data)

	// include link category
	linkCat, err := ctx.Application.WorkItemLinkCategories().Load(ctx.Context, linkType.Data.Relationships.LinkCategory.Data.ID)
	if err != nil {
		return errs.WithStack(err)
	}
	link.Included = append(link.Included, linkCat.Data)

	// TODO(kwk): include source work item type (once #559 is merged)
	// sourceWit, err := appl.WorkItemTypes().Load(ctx, linkType.Data.Relationships.SourceType.Data.ID)
	// if err != nil {
	// 	return errs.WithStack(err)
	// }
	// link.Included = append(link.Included, sourceWit.Data)

	// TODO(kwk): include target work item type (once #559 is merged)
	// targetWit, err := appl.WorkItemTypes().Load(ctx, linkType.Data.Relationships.TargetType.Data.ID)
	// if err != nil {
	// 	return errs.WithStack(err)
	// }
	// link.Included = append(link.Included, targetWit.Data)

	// TODO(kwk): include source work item
	sourceWi, err := ctx.Application.WorkItems().Load(ctx.Context, link.Data.Relationships.Source.Data.ID)
	if err != nil {
		return errs.WithStack(err)
	}
	link.Included = append(link.Included, ConvertWorkItem(ctx.RequestData, sourceWi))

	// TODO(kwk): include target work item
	targetWi, err := ctx.Application.WorkItems().Load(ctx.Context, link.Data.Relationships.Target.Data.ID)
	if err != nil {
		return errs.WithStack(err)
	}
	link.Included = append(link.Included, ConvertWorkItem(ctx.RequestData, targetWi))

	// Add links to individual link data element
	selfURL := rest.AbsoluteURL(ctx.RequestData, ctx.LinkFunc(*link.Data.ID))
	link.Data.Links = &app.GenericLinks{
		Self: &selfURL,
	}

	return nil
}

// enrichLinkList includes related resources in the linkArr's "included" element
func enrichLinkList(ctx *workItemLinkContext, linkArr *app.WorkItemLinkList) error {

	// include link types
	typeDataArr, err := getTypesOfLinks(ctx, linkArr.Data)
	if err != nil {
		return errs.WithStack(err)
	}
	// Convert slice of objects to slice of interface (see https://golang.org/doc/faq#convert_slice_of_interface)
	interfaceArr := make([]interface{}, len(typeDataArr))
	for i, v := range typeDataArr {
		interfaceArr[i] = v
	}
	linkArr.Included = append(linkArr.Included, interfaceArr...)

	// include link categories
	catDataArr, err := getCategoriesOfLinkTypes(ctx, typeDataArr)
	if err != nil {
		return errs.WithStack(err)
	}
	// Convert slice of objects to slice of interface (see https://golang.org/doc/faq#convert_slice_of_interface)
	interfaceArr = make([]interface{}, len(catDataArr))
	for i, v := range catDataArr {
		interfaceArr[i] = v
	}
	linkArr.Included = append(linkArr.Included, interfaceArr...)

	// TODO(kwk): Include WIs from source and target
	workItemDataArr, err := getWorkItemsOfLinks(ctx, linkArr.Data)
	if err != nil {
		return errs.WithStack(err)
	}
	// Convert slice of objects to slice of interface (see https://golang.org/doc/faq#convert_slice_of_interface)
	interfaceArr = make([]interface{}, len(workItemDataArr))
	for i, v := range workItemDataArr {
		interfaceArr[i] = v
	}
	linkArr.Included = append(linkArr.Included, interfaceArr...)

	// TODO(kwk): Include WITs (once #559 is merged)

	// Add links to individual link data element
	for _, link := range linkArr.Data {
		selfURL := rest.AbsoluteURL(ctx.RequestData, ctx.LinkFunc(*link.ID))
		link.Links = &app.GenericLinks{
			Self: &selfURL,
		}
	}

	return nil
}

type createWorkItemLinkFuncs interface {
	BadRequest(r *app.JSONAPIErrors) error
	Created(r *app.WorkItemLinkSingle) error
}

func createWorkItemLink(ctx *workItemLinkContext, funcs createWorkItemLinkFuncs, payload *app.CreateWorkItemLinkPayload) error {
	// Convert payload from app to model representation
	model := link.WorkItemLink{}
	in := app.WorkItemLinkSingle{
		Data: payload.Data,
	}
	err := link.ConvertLinkToModel(in, &model)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(err)
		return funcs.BadRequest(jerrors)
	}
	link, err := ctx.Application.WorkItemLinks().Create(ctx.Context, model.SourceID, model.TargetID, model.LinkTypeID)
	if err != nil {
		cause := errs.Cause(err)
		switch cause.(type) {
		case errors.NotFoundError:
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(err.Error()))
			return funcs.BadRequest(jerrors)
		case errors.BadParameterError:
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(err.Error()))
			return funcs.BadRequest(jerrors)
		default:
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
	}
	if err := enrichLinkSingle(ctx, link); err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
	}

	ctx.ResponseData.Header().Set("Location", app.WorkItemLinkHref(link.Data.ID))
	return funcs.Created(link)
}

// Create runs the create action.
func (c *WorkItemLinkController) Create(ctx *app.CreateWorkItemLinkContext) error {
	return createWorkItemLink(newWorkItemLinkContext(ctx.Context, c.db, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkHref), ctx, ctx.Payload)
}

type deleteWorkItemLinkFuncs interface {
	OK(resp []byte) error
}

func deleteWorkItemLink(ctx *workItemLinkContext, funcs deleteWorkItemLinkFuncs, linkID string) error {
	err := ctx.Application.WorkItemLinks().Delete(ctx.Context, linkID)
	if err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
	}
	return funcs.OK([]byte{})
}

// Delete runs the delete action
func (c *WorkItemLinkController) Delete(ctx *app.DeleteWorkItemLinkContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		return deleteWorkItemLink(newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkHref), ctx, ctx.LinkID)
	})
}

type listWorkItemLinkFuncs interface {
	OK(r *app.WorkItemLinkList) error
}

func listWorkItemLink(ctx *workItemLinkContext, funcs listWorkItemLinkFuncs, wiIDStr *string) error {
	var linkArr *app.WorkItemLinkList
	var err error
	if wiIDStr != nil {
		linkArr, err = ctx.Application.WorkItemLinks().ListByWorkItemID(ctx.Context, *wiIDStr)
	} else {
		linkArr, err = ctx.Application.WorkItemLinks().List(ctx.Context)
	}

	if err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
	}
	if err := enrichLinkList(ctx, linkArr); err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
	}
	return funcs.OK(linkArr)
}

// List runs the list action.
func (c *WorkItemLinkController) List(ctx *app.ListWorkItemLinkContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		return listWorkItemLink(newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkHref), ctx, nil)
	})
}

type showWorkItemLinkFuncs interface {
	OK(r *app.WorkItemLinkSingle) error
}

func showWorkItemLink(ctx *workItemLinkContext, funcs showWorkItemLinkFuncs, linkID string) error {
	link, err := ctx.Application.WorkItemLinks().Load(ctx.Context, linkID)
	if err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
	}
	if err := enrichLinkSingle(ctx, link); err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
	}
	return funcs.OK(link)
}

// Show runs the show action.
func (c *WorkItemLinkController) Show(ctx *app.ShowWorkItemLinkContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		return showWorkItemLink(newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkHref), ctx, ctx.LinkID)
	})
}

type updateWorkItemLinkFuncs interface {
	OK(r *app.WorkItemLinkSingle) error
}

func updateWorkItemLink(ctx *workItemLinkContext, funcs updateWorkItemLinkFuncs, payload *app.UpdateWorkItemLinkPayload) error {
	toSave := app.WorkItemLinkSingle{
		Data: payload.Data,
	}
	link, err := ctx.Application.WorkItemLinks().Save(ctx.Context, toSave)
	if err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
	}
	if err := enrichLinkSingle(ctx, link); err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
		return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
	}
	return funcs.OK(link)
}

// Update runs the update action.
func (c *WorkItemLinkController) Update(ctx *app.UpdateWorkItemLinkContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		return updateWorkItemLink(newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkHref), ctx, ctx.Payload)
	})
}
