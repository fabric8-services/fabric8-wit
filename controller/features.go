package controller

import (
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-wit/app"

	"github.com/goadesign/goa"
)

// FeaturesController implements the features resource/controller.
type FeaturesController struct {
	*goa.Controller
	config FeaturesControllerConfiguration
}

// FeaturesControllerConfiguration the configuration for the features resource/controller.
type FeaturesControllerConfiguration interface {
	GetTogglesServiceURL() string
}

// NewFeaturesController creates a features controller.
func NewFeaturesController(service *goa.Service, config FeaturesControllerConfiguration) *FeaturesController {
	return &FeaturesController{
		Controller: service.NewController("FeaturesController"),
		config:     config,
	}
}

// List runs the list action.
func (c *FeaturesController) List(ctx *app.ListFeaturesContext) error {
	return httpsupport.RouteHTTP(ctx, c.config.GetTogglesServiceURL())
}

// Show runs the show action.
func (c *FeaturesController) Show(ctx *app.ShowFeaturesContext) error {
	return httpsupport.RouteHTTP(ctx, c.config.GetTogglesServiceURL())
}
