package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/goadesign/goa"
	"fmt"
)

// WorkitemController implements the workitem resource.
type WorkitemController struct {
	*goa.Controller
}

// NewWorkitemController creates a workitem controller.
func NewWorkitemController(service *goa.Service) *WorkitemController {
	return &WorkitemController{Controller: service.NewController("WorkitemController")}
}

// Show runs the show action.
func (c *WorkitemController) Show(ctx *app.ShowWorkitemContext) error {
	// TBD: implement
	res := &app.WorkItem{
		ID: "13", 
		Name: "This is a work item 2",
		Type:"/api/workitemtype/1",
		Version: 1}
	res.Fields= map[string]interface{}{
	"Owner": "tmaeder2"}
	fmt.Print("in here");
	return ctx.OK(res)
}
