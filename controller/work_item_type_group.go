package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/workitem/typegroup"
	"github.com/goadesign/goa"
)

// WorkItemTypeGroupController implements the work_item_type_group resource.
type WorkItemTypeGroupController struct {
	*goa.Controller
}

// NewWorkItemTypeGroupController creates a work_item_type_group controller.
func NewWorkItemTypeGroupController(service *goa.Service) *WorkItemTypeGroupController {
	return &WorkItemTypeGroupController{Controller: service.NewController("WorkItemTypeGroupController")}
}

// List runs the list action.
func (c *WorkItemTypeGroupController) List(ctx *app.ListWorkItemTypeGroupContext) error {
	// WorkItemTypeGroupController_List: start_implement

	// Put your logic here

	// WorkItemTypeGroupController_List: end_implement
	res := &app.WorkItemTypeGroupSigleSingle{}
	res.Data = &app.WorkItemTypeGroupData{
		Attributes: &app.WorkItemTypeGroupAttributes{
			Hierarchy: []*app.WorkItemTypeGroup{
				ConvertTypeGroup(ctx.RequestData, typegroup.Portfolio0),
				ConvertTypeGroup(ctx.RequestData, typegroup.Portfolio1),
				ConvertTypeGroup(ctx.RequestData, typegroup.Requirements0),
			},
		},
	}
	return ctx.OK(res)
}

func ConvertTypeGroup(request *goa.RequestData, tg typegroup.WorkItemTypeGroup) *app.WorkItemTypeGroup {
	return &app.WorkItemTypeGroup{
		Group:         tg.Group,
		Level:         tg.Level,
		Name:          tg.Name,
		Sublevel:      tg.Sublevel,
		WitCollection: tg.WorkItemTypeCollection,
	}
}
