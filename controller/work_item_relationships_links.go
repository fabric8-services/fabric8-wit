package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/goadesign/goa"
)

// WorkItemRelationshipsLinksController implements the work-item-relationships-links resource.
type WorkItemRelationshipsLinksController struct {
	*goa.Controller
	db     application.DB
	config WorkItemRelationshipsLinksControllerConfig
}

// WorkItemRelationshipsLinksControllerConfig the config interface for the WorkItemRelationshipsLinksController
type WorkItemRelationshipsLinksControllerConfig interface {
	GetCacheControlWorkItemLinks() string
}

// NewWorkItemRelationshipsLinksController creates a work-item-relationships-links controller.
func NewWorkItemRelationshipsLinksController(service *goa.Service, db application.DB, config WorkItemRelationshipsLinksControllerConfig) *WorkItemRelationshipsLinksController {
	return &WorkItemRelationshipsLinksController{
		Controller: service.NewController("WorkItemRelationshipsLinksController"),
		db:         db,
		config:     config,
	}
}

// List runs the list action.
func (c *WorkItemRelationshipsLinksController) List(ctx *app.ListWorkItemRelationshipsLinksContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		modelLinks, err := appl.WorkItemLinks().ListByWorkItem(ctx.Context, ctx.WiID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalEntities(modelLinks, c.config.GetCacheControlWorkItemLinks, func() error {
			appLinks := app.WorkItemLinkList{}
			appLinks.Data = make([]*app.WorkItemLinkData, len(modelLinks))
			for index, modelLink := range modelLinks {
				appLink := ConvertLinkFromModel(ctx.Request, modelLink)
				appLinks.Data[index] = appLink.Data
			}
			// TODO: When adding pagination, this must not be len(rows) but
			// the overall total number of elements from all pages.
			appLinks.Meta = &app.WorkItemLinkListMeta{
				TotalCount: len(modelLinks),
			}
			if err := enrichLinkList(ctx.Context, appl, ctx.Request, &appLinks); err != nil {
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			return ctx.OK(&appLinks)
		})
	})
}
