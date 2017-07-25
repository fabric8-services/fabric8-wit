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
	// SpaceTemplateController_Show: start_implement

	// Put your logic here

	// SpaceTemplateController_Show: end_implement
	return nil
}
