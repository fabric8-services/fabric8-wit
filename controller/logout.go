package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/goadesign/goa"
)

// LogoutController implements the logout resource.
type LogoutController struct {
	*goa.Controller
	configuration auth.ServiceConfiguration
}

// NewLogoutController creates a logout controller.
func NewLogoutController(service *goa.Service, configuration auth.ServiceConfiguration) *LogoutController {
	return &LogoutController{Controller: service.NewController("LogoutController"), configuration: configuration}
}

// Logout runs the logout action.
func (c *LogoutController) Logout(ctx *app.LogoutLogoutContext) error {
	return redirectWithParams(ctx, c.configuration, ctx.ResponseData.Header(), ctx.Params, authservice.LogoutLogoutPath())
}
