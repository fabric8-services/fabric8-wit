package controller

import (
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/goadesign/goa"
)

type logoutConfiguration interface {
	GetAuthEndpointLogout(*http.Request) (string, error)
}

// LogoutController implements the logout resource.
type LogoutController struct {
	*goa.Controller
	configuration logoutConfiguration
}

// NewLogoutController creates a logout controller.
func NewLogoutController(service *goa.Service, configuration logoutConfiguration) *LogoutController {
	return &LogoutController{Controller: service.NewController("LogoutController"), configuration: configuration}
}

// Logout runs the logout action.
func (c *LogoutController) Logout(ctx *app.LogoutLogoutContext) error {
	authEndpoint, err := c.configuration.GetAuthEndpointLogout(ctx.RequestData.Request)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}
	locationURL, err := redirectLocation(ctx.Params, authEndpoint)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}
	ctx.ResponseData.Header().Set("Location", locationURL)
	return ctx.TemporaryRedirect()
}
