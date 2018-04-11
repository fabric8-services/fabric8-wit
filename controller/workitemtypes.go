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

// WorkitemtypesController implements the workitemtype resource.
type WorkitemtypesController struct {
	*goa.Controller
	db     application.DB
	config WorkItemControllerConfiguration
}

// NewWorkitemtypesController creates a workitemtype controller.
func NewWorkitemtypesController(service *goa.Service, db application.DB, config WorkItemControllerConfiguration) *WorkitemtypesController {
	return &WorkitemtypesController{
		Controller: service.NewController("WorkitemtypesController"),
		db:         db,
		config:     config,
	}
}

// List runs the list action
func (c *WorkitemtypesController) List(ctx *app.ListWorkitemtypesContext) error {
	log.Debug(ctx, map[string]interface{}{"space_id": ctx.SpaceID}, "Listing work item types per space")
	witModels := []workitem.WorkItemType{}
	err := application.Transactional(c.db, func(appl application.Application) error {
		witModelsOrig, err := appl.WorkItemTypes().List(ctx.Context, ctx.SpaceID)
		if err != nil {
			return errs.Wrap(err, "Error listing work item types")
		}
		// Remove "planneritem" from the list of WITs
		for _, wit := range witModelsOrig {
			if wit.ID != workitem.SystemPlannerItem {
				witModels = append(witModels, wit)
			}
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.ConditionalEntities(witModels, c.config.GetCacheControlWorkItemTypes, func() error {
		// TEMP!!!!! Until Space Template can setup a Space, redirect to SystemSpace WITs if non are found
		// for the space.
		err = application.Transactional(c.db, func(appl application.Application) error {
			if len(witModels) == 0 {
				witModels, err = appl.WorkItemTypes().List(ctx.Context, space.SystemSpace)
				if err != nil {
					return errs.Wrap(err, "Error listing work item types")
				}
			}
			return nil
		})
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
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
}
