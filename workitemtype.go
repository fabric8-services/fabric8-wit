package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/goadesign/goa"
)

var wellKnown = map[string]*app.WorkItemType{
	"1": &app.WorkItemType{
		ID:      "1",
		Name:    "system.workitem",
		Version: 1,
		Fields: []*app.Field{
			{"system.owner", "user"},
			{"system.state", "string"}}}}

// WorkitemtypeController implements the workitemtype resource.
type WorkitemtypeController struct {
	*goa.Controller
}

// NewWorkitemtypeController creates a workitemtype controller.
func NewWorkitemtypeController(service *goa.Service) *WorkitemtypeController {
	return &WorkitemtypeController{Controller: service.NewController("WorkitemtypeController")}
}

// Show runs the show action.
func (c *WorkitemtypeController) Show(ctx *app.ShowWorkitemtypeContext) error {
	res := wellKnown[ctx.ID]
	if res != nil {
		return ctx.OK(res)
	}
	return ctx.NotFound()
}

//enum('foo', 'bar')
