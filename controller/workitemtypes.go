package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

// WorkitemtypesController implements the workitemtype resource.
type WorkitemtypesController struct {
	*goa.Controller
	db     application.DB
	config workItemTypesControllerConfiguration
}

type workItemTypesControllerConfiguration interface {
	GetCacheControlWorkItemTypes() string
	GetCacheControlWorkItemType() string
}

// NewWorkitemtypesController creates a workitemtype controller.
func NewWorkitemtypesController(service *goa.Service, db application.DB, config workItemTypesControllerConfiguration) *WorkitemtypesController {
	return &WorkitemtypesController{
		Controller: service.NewController("WorkitemtypesController"),
		db:         db,
		config:     config,
	}
}

// List runs the list action
func (c *WorkitemtypesController) List(ctx *app.ListWorkitemtypesContext) error {
	log.Debug(ctx, map[string]interface{}{"space_template_id": ctx.SpaceTemplateID}, "Listing work item types per space template")
	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Could not parse paging"))
	}
	result := &app.WorkItemTypeList{}
	err = application.Transactional(c.db, func(appl application.Application) error {
		witModels, err := appl.WorkItemTypes().List(ctx.Context, ctx.SpaceTemplateID, start, &limit)
		if err != nil {
			return errs.Wrap(err, "Error listing work item types")
		}
		return ctx.ConditionalEntities(witModels, c.config.GetCacheControlWorkItemTypes, func() error {
			// convert from model to app
			result.Data = make([]*app.WorkItemTypeData, len(witModels))
			for index, value := range witModels {
				wit := ConvertWorkItemTypeFromModel(ctx.Request, &value)
				result.Data[index] = &wit
			}
			return nil
		})
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.OK(result)
}
