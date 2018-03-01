package controller

import (
	"fmt"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"

	"github.com/goadesign/goa"
)

// WorkItemLinkTypesController implements the work-item-link-types resource.
type WorkItemLinkTypesController struct {
	*goa.Controller
	db     application.DB
	config WorkItemLinkTypesControllerConfiguration
}

// WorkItemLinkTypesControllerConfiguration the configuration for the WorkItemLinkTypesController
type WorkItemLinkTypesControllerConfiguration interface {
	GetCacheControlWorkItemLinkTypes() string
	GetCacheControlWorkItemLinkType() string
}

// NewWorkItemLinkTypesController creates a work-item-link-types controller.
func NewWorkItemLinkTypesController(service *goa.Service, db application.DB, config WorkItemLinkTypesControllerConfiguration) *WorkItemLinkTypesController {
	return &WorkItemLinkTypesController{
		Controller: service.NewController("WorkItemLinkTypesController"),
		db:         db,
		config:     config,
	}
}

// List runs the list action.
func (c *WorkItemLinkTypesController) List(ctx *app.ListWorkItemLinkTypesContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		modelLinkTypes, err := appl.WorkItemLinkTypes().List(ctx.Context, ctx.SpaceTemplateID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalEntities(modelLinkTypes, c.config.GetCacheControlWorkItemLinkTypes, func() error {
			// convert to rest representation
			appLinkTypes := app.WorkItemLinkTypeList{}
			appLinkTypes.Data = make([]*app.WorkItemLinkTypeData, len(modelLinkTypes))
			for index, modelLinkType := range modelLinkTypes {
				appLinkType := ConvertWorkItemLinkTypeFromModel(ctx.Request, modelLinkType)
				appLinkTypes.Data[index] = appLinkType.Data
			}
			// TODO: When adding pagination, this must not be len(rows) but
			// the overall total number of elements from all pages.
			appLinkTypes.Meta = &app.WorkItemLinkTypeListMeta{
				TotalCount: len(modelLinkTypes),
			}
			// Enrich
			hrefFunc := func(obj interface{}) string {
				return fmt.Sprintf(app.WorkItemLinkTypeHref("%v"), obj)
			}
			linkCtx := newWorkItemLinkContext(ctx.Context, ctx.Service, appl, c.db, ctx.Request, ctx.ResponseWriter, hrefFunc, nil)
			err = enrichLinkTypeList(linkCtx, &appLinkTypes)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal("Failed to enrich link types: %s", err.Error()))
			}
			return ctx.OK(&appLinkTypes)
		})
	})
}
