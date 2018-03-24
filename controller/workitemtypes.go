package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

// WorkitemtypesController implements the workitemtypes resource.
type WorkitemtypesController struct {
	*goa.Controller
	db     application.DB
	config WorkItemTypeControllerConfiguration
}

// NewWorkitemtypesController creates a workitemtype controller.
func NewWorkitemtypesController(service *goa.Service, db application.DB, config WorkItemTypeControllerConfiguration) *WorkitemtypesController {
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
	return application.Transactional(c.db, func(appl application.Application) error {
		witModelsOrig, err := appl.WorkItemTypes().List(ctx.Context, ctx.SpaceTemplateID, start, &limit)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Error listing work item types"))
		}
		// Remove "planneritem" from the list of WITs
		// TODO(kwk): This workaround can be removed because we have wit.CanConstruct now and the UI can filter on it.
		witModels := []workitem.WorkItemType{}
		for _, wit := range witModelsOrig {
			if wit.ID != workitem.SystemPlannerItem {
				witModels = append(witModels, wit)
			}
		}
		return ctx.ConditionalEntities(witModels, c.config.GetCacheControlWorkItemTypes, func() error {
			// TODO(kwk): Work item types are associated with space template, so use that here to list
			// TEMP!!!!! Until Space Template can setup a Space, redirect to SystemSpace WITs if non are found
			// for the space.
			if len(witModels) == 0 {
				witModels, err = appl.WorkItemTypes().List(ctx.Context, space.SystemSpace, start, &limit)
				if err != nil {
					return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Error listing work item types"))
				}
			}
			// convert from model to app
			result := &app.WorkItemTypeList{}
			result.Data = make([]*app.WorkItemTypeData, len(witModels))
			for index, value := range witModels {
				wit := ConvertWorkItemTypeFromModel(ctx.Request, &value)
				result.Data[index] = &wit
			}
			return ctx.OK(result)
		})
	})
}
