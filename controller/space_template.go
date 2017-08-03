package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/goadesign/goa"
)

// SpaceTemplateController implements the space_template resource.
type SpaceTemplateController struct {
	*goa.Controller
}

// NewSpaceTemplateController creates a space_template controller.
func NewSpaceTemplateController(service *goa.Service) *SpaceTemplateController {
	return &SpaceTemplateController{Controller: service.NewController("SpaceTemplateController")}
}

// Show runs the show action.
func (c *SpaceTemplateController) Show(ctx *app.ShowSpaceTemplateContext) error {
	// till we have space templates in place, let this controller redirect
	// user to the typegroups endpoint that returns list of type-groups.
	typeGroupURL := app.SpaceTemplateHref(ctx.SpaceTemplateID) + "/workitemtypegroups/"
	ctx.ResponseData.Header().Set("Location", typeGroupURL)
	return ctx.TemporaryRedirect()
}
