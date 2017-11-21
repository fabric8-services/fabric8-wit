package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"

	"github.com/fabric8-services/fabric8-wit/rest/proxy"
	"github.com/goadesign/goa"
)

type loginConfiguration interface {
	auth.ServiceConfiguration
	IsPostgresDeveloperModeEnabled() bool
	GetKeycloakTestUserName() string
	GetKeycloakTestUser2Name() string
}

type redirectContext interface {
	context.Context
	TemporaryRedirect() error
}

// LoginController implements the login resource.
type LoginController struct {
	*goa.Controller
	auth               login.KeycloakOAuthService
	configuration      loginConfiguration
	identityRepository account.IdentityRepository
}

// NewLoginController creates a login controller.
func NewLoginController(service *goa.Service, auth *login.KeycloakOAuthProvider, configuration loginConfiguration, identityRepository account.IdentityRepository) *LoginController {
	return &LoginController{Controller: service.NewController("login"), auth: auth, configuration: configuration, identityRepository: identityRepository}
}

// Authorize runs the authorize action.
func (c *LoginController) Authorize(ctx *app.AuthorizeLoginContext) error {
	return redirectWithParams(ctx, c.configuration, ctx.ResponseData.Header(), ctx.Params, authservice.LoginLoginPath())
}

// Refresh obtain a new access token using the refresh token.
func (c *LoginController) Refresh(ctx *app.RefreshLoginContext) error {
	return proxy.RouteHTTPToPath(ctx, c.configuration.GetAuthShortServiceHostName(), authservice.RefreshTokenPath())
}

// Link links identity provider(s) to the user's account
func (c *LoginController) Link(ctx *app.LinkLoginContext) error {
	return redirectWithParams(ctx, c.configuration, ctx.ResponseData.Header(), ctx.Params, authservice.LinkLinkPath())
}

// Linksession links identity provider(s) to the user's account
func (c *LoginController) Linksession(ctx *app.LinksessionLoginContext) error {
	return redirectWithParams(ctx, c.configuration, ctx.ResponseData.Header(), ctx.Params, authservice.SessionLinkPath())
}

func redirectLocation(params url.Values, location string) (string, error) {
	locationURL, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	parameters := locationURL.Query()
	for name := range params {
		parameters.Add(name, params.Get(name))
	}
	locationURL.RawQuery = parameters.Encode()
	return locationURL.String(), nil
}

func redirectWithParams(ctx redirectContext, config auth.ServiceConfiguration, header http.Header, params url.Values, path string) error {
	locationURL := fmt.Sprintf("%s%s", config.GetAuthServiceURL(), path)
	locationURLWithParams, err := redirectLocation(params, locationURL)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}
	header.Set("Location", locationURLWithParams)
	return ctx.TemporaryRedirect()
}

// Generate generates access tokens in Dev Mode
func (c *LoginController) Generate(ctx *app.GenerateLoginContext) error {
	return proxy.RouteHTTPToPath(ctx, c.configuration.GetAuthServiceURL(), authservice.GenerateTokenPath())
}
