package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

// SpaceworkitemtypesController implements the spaceworkitemtypes resource.
type SpaceworkitemtypesController struct {
	*goa.Controller
	db     application.DB
	config workItemTypesControllerConfiguration
}

// NewSpaceworkitemtypesController creates a spaceworkitemtypes controller.
func NewSpaceworkitemtypesController(service *goa.Service, db application.DB, config workItemTypesControllerConfiguration) *SpaceworkitemtypesController {
	return &SpaceworkitemtypesController{Controller: service.NewController("SpaceworkitemtypesController"),
		db:     db,
		config: config,
	}
}

// Listforspace runs the listforspace action.
func (c *SpaceworkitemtypesController) Listforspace(ctx *app.ListforspaceSpaceworkitemtypesContext) error {
	log.Debug(ctx, map[string]interface{}{"space_id": ctx.SpaceID}, "Listing work item types per space template")
	result := &app.WorkItemTypeList{}
	err := application.Transactional(c.db, func(appl application.Application) error {
		witModels, err := appl.WorkItemTypes().ListForSpace(ctx.Context, ctx.SpaceID)
		if err != nil {
			return errs.Wrap(err, "Error listing work item types")
		}

		return ctx.ConditionalEntities(witModels, c.config.GetCacheControlWorkItemTypes, func() error {
			// convert from model to app
			// result := &app.WorkItemTypeList{}
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
