package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
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
	res := &app.WorkItemTypeGroupsList{}
	return ctx.OK(res)
}
