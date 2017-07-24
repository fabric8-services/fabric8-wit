package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/goadesign/goa"
)

// SpaceTemplateController implements the space_template resource.
type SpaceTemplateController struct {
	*goa.Controller
	db     application.DB
	config SpaceIterationsControllerConfiguration
}

// NewSpaceTemplateController creates a space_template controller.
func NewSpaceTemplateController(service *goa.Service, db application.DB, config SpaceIterationsControllerConfiguration) *SpaceTemplateController {
	return &SpaceTemplateController{Controller: service.NewController("SpaceTemplateController"), db: db, config: config}
}

// Show runs the show action.
func (c *SpaceTemplateController) Show(ctx *app.ShowSpaceTemplateContext) error {
	// SpaceTemplateController_Show: start_implement

	// Put your logic here

	// SpaceTemplateController_Show: end_implement
	res := &app.SpaceTemplateSingle{}
	return ctx.OK(res)
}
