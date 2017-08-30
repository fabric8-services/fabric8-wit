package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/goadesign/goa"
)

type logoutConfiguration interface {
	GetAuthEndpointLogout(req *goa.RequestData) (string, error)
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
	logoutEndpoint, err := c.configuration.GetAuthEndpointLogout(ctx.RequestData)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}
	redirect := ctx.Redirect
	referrer := ctx.RequestData.Header.Get("Referer")
	if redirect == nil {
		if referrer == "" {
			log.Error(ctx, nil, "Failed to logout. Referer Header and redirect param are both empty.")
			return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest("referer Header and redirect param are both empty (at least one should be specified)"))
		}
		redirect = &referrer
	}
	log.Info(ctx, map[string]interface{}{
		"referrer": referrer,
		"redirect": redirect,
	}, "Got Request to logout!")

	ctx.ResponseData.Header().Set("Location", logoutEndpoint+"?redirect="+*redirect)
	return ctx.TemporaryRedirect()
}
