package main

import (
	"github.com/ALMighty/almighty-core/app"
	"github.com/goadesign/goa"
)

// VersionController implements the version resource.
type VersionController struct {
	*goa.Controller
}

// NewVersionController creates a version controller.
func NewVersionController(service *goa.Service) *VersionController {
	return &VersionController{Controller: service.NewController("version")}
}

// Show runs the show action.
func (c *VersionController) Show(ctx *app.ShowVersionContext) error {
	res := &app.Version{}
	res.Commit = Commit
	res.BuildTime = BuildTime
	return ctx.OK(res)
}
