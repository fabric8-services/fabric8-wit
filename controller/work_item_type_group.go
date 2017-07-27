package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem/typegroup"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

// WorkItemTypeGroupController implements the work_item_type_group resource.
type WorkItemTypeGroupController struct {
	*goa.Controller
	db application.DB
}

// NewWorkItemTypeGroupController creates a work_item_type_group controller.
func NewWorkItemTypeGroupController(service *goa.Service, db application.DB) *WorkItemTypeGroupController {
	return &WorkItemTypeGroupController{
		Controller: service.NewController("WorkItemTypeGroupController"),
		db:         db,
	}
}

// List runs the list action.
func (c *WorkItemTypeGroupController) List(ctx *app.ListWorkItemTypeGroupContext) error {
	res := &app.WorkItemTypeGroupSigleSingle{}
	res.Data = &app.WorkItemTypeGroupData{
		Attributes: &app.WorkItemTypeGroupAttributes{
			Hierarchy: []*app.WorkItemTypeGroup{
				ConvertTypeGroup(ctx.RequestData, typegroup.Portfolio0),
				ConvertTypeGroup(ctx.RequestData, typegroup.Portfolio1),
				ConvertTypeGroup(ctx.RequestData, typegroup.Requirements0),
			},
		},
		Included: []*app.WorkItemTypeData{},
	}
	err := includeWorkItemType(c, ctx, res.Data)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Failed to load work item type groups"))
	}
	return ctx.OK(res)
}

// include workitemtype entries into JSON API response
func includeWorkItemType(c *WorkItemTypeGroupController, ctx *app.ListWorkItemTypeGroupContext, res *app.WorkItemTypeGroupData) error {
	err := application.Transactional(c.db, func(appl application.Application) error {
		witTypes, err := appl.WorkItemTypes().List(ctx, space.SystemSpace, nil, nil)
		if err != nil {
			return err
		}
		for _, witType := range witTypes {
			converted := ConvertWorkItemTypeFromModel(ctx.RequestData, &witType)
			res.Included = append(res.Included, &converted)
		}
		return nil
	})
	if err != nil {
		log.Error(ctx, map[string]interface{}{"space_id": space.SystemSpace}, "Unable to retrieve workitem types from system space")
	}
	return err
}

// ConvertTypeGroup converts WorkitemTypeGroup model to a response resource
// object for jsonapi.org specification
func ConvertTypeGroup(request *goa.RequestData, tg typegroup.WorkItemTypeGroup) *app.WorkItemTypeGroup {
	return &app.WorkItemTypeGroup{
		Group:         tg.Group,
		Level:         tg.Level,
		Name:          tg.Name,
		Sublevel:      tg.Sublevel,
		WitCollection: tg.WorkItemTypeCollection,
	}
}
