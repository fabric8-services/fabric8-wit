package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/workitem/typegroup"
	"github.com/goadesign/goa"
)

// WorkItemTypeGroupController implements the work_item_type_group resource.
type WorkItemTypeGroupController struct {
	*goa.Controller
	db application.DB
}

const APIWorkItemTypeGroups = "workitemtypegroups"

// NewWorkItemTypeGroupController creates a work_item_type_group controller.
func NewWorkItemTypeGroupController(service *goa.Service, db application.DB) *WorkItemTypeGroupController {
	return &WorkItemTypeGroupController{
		Controller: service.NewController("WorkItemTypeGroupController"),
		db:         db,
	}
}

// List runs the list action.
func (c *WorkItemTypeGroupController) List(ctx *app.ListWorkItemTypeGroupContext) error {
	err := application.Transactional(c.db, func(appl application.Application) error {
		return appl.Spaces().CheckExists(ctx, ctx.SpaceTemplateID.String())
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	res := &app.WorkItemTypeGroupSigleSingle{}
	res.Data = &app.WorkItemTypeGroupData{
		Attributes: &app.WorkItemTypeGroupAttributes{
			Hierarchy: []*app.WorkItemTypeGroup{
				ConvertTypeGroup(ctx.RequestData, typegroup.Portfolio0),
				ConvertTypeGroup(ctx.RequestData, typegroup.Portfolio1),
				ConvertTypeGroup(ctx.RequestData, typegroup.Requirements0),
				ConvertTypeGroup(ctx.RequestData, typegroup.Execution0),
			},
		},
		Type: APIWorkItemTypeGroups,
	}
	return ctx.OK(res)
}

// ConvertTypeGroup converts WorkitemTypeGroup model to a response resource
// object for jsonapi.org specification
func ConvertTypeGroup(request *goa.RequestData, tg typegroup.WorkItemTypeGroup) *app.WorkItemTypeGroup {
	return &app.WorkItemTypeGroup{
		Group:         tg.Group,
		Level:         tg.Level,
		Name:          tg.Name,
		WitCollection: tg.WorkItemTypeCollection,
	}
}
