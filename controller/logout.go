package controller

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/login"
	"github.com/goadesign/goa"
)

type logoutConfiguration interface {
	GetKeycloakEndpointLogout(*goa.RequestData) (string, error)
	GetValidRedirectURLs(*goa.RequestData) (string, error)
}

// LogoutController implements the logout resource.
type LogoutController struct {
	*goa.Controller
	logoutService login.LogoutService
	configuration logoutConfiguration
}

// NewLogoutController creates a logout controller.
func NewLogoutController(service *goa.Service, logoutService *login.KeycloakLogoutService, configuration logoutConfiguration) *LogoutController {
	return &LogoutController{Controller: service.NewController("LogoutController"), logoutService: logoutService, configuration: configuration}
}

// Logout runs the logout action.
func (c *LogoutController) Logout(ctx *app.LogoutLogoutContext) error {
	logoutEndpoint, err := c.configuration.GetKeycloakEndpointLogout(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "Unable to get Keycloak logout endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to get Keycloak logout endpoint URL. "+err.Error()))
	}
	whitelist, err := c.configuration.GetValidRedirectURLs(ctx.RequestData)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(err.Error()))
	}
	return c.logoutService.Logout(ctx, logoutEndpoint, whitelist)
}
