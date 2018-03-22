package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/goadesign/goa"
	//uuid "github.com/satori/go.uuid"
)

// WorkItemLinkCategoryController implements the work-item-link-category resource.
type WorkItemLinkCategoryController struct {
	*goa.Controller
	db application.DB
}

// NewWorkItemLinkCategoryController creates a WorkItemLinkCategoryController.
func NewWorkItemLinkCategoryController(service *goa.Service, db application.DB) *WorkItemLinkCategoryController {
	return &WorkItemLinkCategoryController{
		Controller: service.NewController("WorkItemLinkCategoryController"),
		db:         db,
	}
}

// enrichLinkCategorySingle includes related resources in the single's "included" array
func enrichLinkCategorySingle(ctx *workItemLinkContext, single app.WorkItemLinkCategorySingle) error {
	// Add "links" element
	relatedURL := rest.AbsoluteURL(ctx.Request, ctx.LinkFunc(single.Data.ID))
	single.Data.Links = &app.GenericLinks{
		Self:    &relatedURL,
		Related: &relatedURL,
	}
	return nil
}

// enrichLinkCategoryList includes related resources in the list's "included" array
func enrichLinkCategoryList(ctx *workItemLinkContext, list *app.WorkItemLinkCategoryList) error {
	// Add "links" element
	for _, data := range list.Data {
		relatedURL := rest.AbsoluteURL(ctx.Request, ctx.LinkFunc(*data.ID))
		data.Links = &app.GenericLinks{
			Self:    &relatedURL,
			Related: &relatedURL,
		}
	}
	return nil
}

// Create runs the create action.
func (c *WorkItemLinkCategoryController) Create(ctx *app.CreateWorkItemLinkCategoryContext) error {
	// Currently not used. Disabled as part of https://github.com/fabric8-services/fabric8-wit/issues/1299
	if true {
		return ctx.MethodNotAllowed()
	}
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	modelCategory := convertLinkCategoryToModel(app.WorkItemLinkCategorySingle{Data: ctx.Payload.Data})
	var appCategory app.WorkItemLinkCategorySingle
	err = application.Transactional(c.db, func(appl application.Application) error {
		_, err := appl.WorkItemLinkCategories().Create(ctx.Context, &modelCategory)
		if err != nil {
			return err
		}
		appCategory = ConvertLinkCategoryFromModel(modelCategory)
		linkCtx := newWorkItemLinkContext(ctx.Context, ctx.Service, appl, c.db, ctx.Request, ctx.ResponseWriter, app.WorkItemLinkCategoryHref, currentUserIdentityID)
		return enrichLinkCategorySingle(linkCtx, appCategory)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	ctx.ResponseData.Header().Set("Location", app.WorkItemLinkCategoryHref(appCategory.Data.ID))
	return ctx.Created(&appCategory)
}

// Show runs the show action.
func (c *WorkItemLinkCategoryController) Show(ctx *app.ShowWorkItemLinkCategoryContext) error {
	err := application.Transactional(c.db, func(appl application.Application) error {
		modelCategory, err := appl.WorkItemLinkCategories().Load(ctx.Context, ctx.ID)
		if err != nil {
			return err
		}
		appCategory := ConvertLinkCategoryFromModel(*modelCategory)
		linkCtx := newWorkItemLinkContext(ctx.Context, ctx.Service, appl, c.db, ctx.Request, ctx.ResponseWriter, app.WorkItemLinkCategoryHref, nil)
		err = enrichLinkCategorySingle(linkCtx, appCategory)
		if err != nil {
			return goa.ErrInternal("Failed to enrich link category: %s", err.Error())
		}
		return ctx.OK(&appCategory)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return nil
}

// List runs the list action.
func (c *WorkItemLinkCategoryController) List(ctx *app.ListWorkItemLinkCategoryContext) error {
	err := application.Transactional(c.db, func(appl application.Application) error {
		modelCategories, err := appl.WorkItemLinkCategories().List(ctx.Context)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(ctx, err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		// convert
		appCategories := app.WorkItemLinkCategoryList{}
		appCategories.Data = make([]*app.WorkItemLinkCategoryData, len(modelCategories))
		for index, value := range modelCategories {
			cat := ConvertLinkCategoryFromModel(value)
			appCategories.Data[index] = cat.Data
		}
		// TODO: When adding pagination, this must not be len(rows) but
		// the overall total number of elements from all pages.
		appCategories.Meta = &app.WorkItemLinkCategoryListMeta{
			TotalCount: len(modelCategories),
		}
		// Enrich
		linkCtx := newWorkItemLinkContext(ctx.Context, ctx.Service, appl, c.db, ctx.Request, ctx.ResponseWriter, app.WorkItemLinkCategoryHref, nil)
		err = enrichLinkCategoryList(linkCtx, &appCategories)
		if err != nil {
			return goa.ErrInternal("Failed to enrich link categories: %s", err.Error())
		}
		return ctx.OK(&appCategories)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return nil
}

// Delete runs the delete action.
func (c *WorkItemLinkCategoryController) Delete(ctx *app.DeleteWorkItemLinkCategoryContext) error {
	// Currently not used. Disabled as part of https://github.com/fabric8-services/fabric8-wit/issues/1299
	if true {
		return ctx.MethodNotAllowed()
	}
	err := application.Transactional(c.db, func(appl application.Application) error {
		err := appl.WorkItemLinkCategories().Delete(ctx.Context, ctx.ID)
		if err != nil {
			return err
		}
		return ctx.OK([]byte{})
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return nil
}

// Update runs the update action.
func (c *WorkItemLinkCategoryController) Update(ctx *app.UpdateWorkItemLinkCategoryContext) error {
	// Currently not used. Disabled as part of https://github.com/fabric8-services/fabric8-wit/issues/1299
	if true {
		return ctx.MethodNotAllowed()
	}
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	appCategory := app.WorkItemLinkCategorySingle{
		Data: ctx.Payload.Data,
	}
	if appCategory.Data.ID == nil {
		return errors.NewBadParameterError("data.id", appCategory.Data.ID)
	}
	if appCategory.Data.Attributes.Name == nil || *appCategory.Data.Attributes.Name == "" {
		return errors.NewBadParameterError("data.attributes.name", "nil or empty")
	}
	modelCategory := convertLinkCategoryToModel(appCategory)
	err = application.Transactional(c.db, func(appl application.Application) error {
		savedModelCategory, err := appl.WorkItemLinkCategories().Save(ctx.Context, modelCategory)
		if err != nil {
			return err
		}
		// convert to app representation
		savedAppCategory := ConvertLinkCategoryFromModel(*savedModelCategory)
		// Enrich
		linkCtx := newWorkItemLinkContext(ctx.Context, ctx.Service, appl, c.db, ctx.Request, ctx.ResponseWriter, app.WorkItemLinkCategoryHref, currentUserIdentityID)
		err = enrichLinkCategorySingle(linkCtx, savedAppCategory)
		if err != nil {
			return goa.ErrInternal("Failed to enrich link category: %s", err.Error())
		}
		return ctx.OK(&savedAppCategory)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return nil
}

// convertLinkCategoryFromModel converts work item link category from model to app representation
func ConvertLinkCategoryFromModel(t link.WorkItemLinkCategory) app.WorkItemLinkCategorySingle {
	var converted = app.WorkItemLinkCategorySingle{
		Data: &app.WorkItemLinkCategoryData{
			Type: link.EndpointWorkItemLinkCategories,
			ID:   &t.ID,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &t.Name,
				Description: t.Description,
				Version:     &t.Version,
			},
		},
	}
	return converted
}

// convertLinkCategoryToModel converts work item link category from app to app representation
func convertLinkCategoryToModel(t app.WorkItemLinkCategorySingle) link.WorkItemLinkCategory {
	var converted = link.WorkItemLinkCategory{}
	if t.Data.ID != nil {
		converted.ID = *t.Data.ID
	}
	if t.Data.Attributes.Version != nil {
		converted.Version = *t.Data.Attributes.Version
	}
	if t.Data.Attributes.Name != nil {
		converted.Name = *t.Data.Attributes.Name
	}
	if t.Data.Attributes.Description != nil {
		converted.Description = t.Data.Attributes.Description
	}
	return converted
}
