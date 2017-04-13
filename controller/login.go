package controller

import (
	"context"
	"time"

	"golang.org/x/oauth2"

	"net/http"
	"net/url"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/auth"
	"github.com/almighty/almighty-core/rest"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
)

type loginConfiguration interface {
	GetKeycloakEndpointAuth(*goa.RequestData) (string, error)
	GetKeycloakEndpointToken(*goa.RequestData) (string, error)
	GetKeycloakAccountEndpoint(req *goa.RequestData) (string, error)
	GetKeycloakEndpointBroker(*goa.RequestData) (string, error)
	GetKeycloakEndpointEntitlement(*goa.RequestData) (string, error)
	GetKeycloakClientID() string
	GetKeycloakSecret() string
	IsPostgresDeveloperModeEnabled() bool
	GetKeycloakTestUserName() string
	GetKeycloakTestUserSecret() string
	GetKeycloakTestUser2Name() string
	GetKeycloakTestUser2Secret() string
	GetValidRedirectURLs(*goa.RequestData) (string, error)
}

// LoginController implements the login resource.
type LoginController struct {
	*goa.Controller
	auth          login.KeycloakOAuthService
	tokenManager  token.Manager
	configuration loginConfiguration
}

// NewLoginController creates a login controller.
func NewLoginController(service *goa.Service, auth *login.KeycloakOAuthProvider, tokenManager token.Manager, configuration loginConfiguration) *LoginController {
	return &LoginController{Controller: service.NewController("login"), auth: auth, tokenManager: tokenManager, configuration: configuration}
}

// Authorize runs the authorize action.
func (c *LoginController) Authorize(ctx *app.AuthorizeLoginContext) error {
	authEndpoint, err := c.configuration.GetKeycloakEndpointAuth(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "Unable to get Keycloak auth endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to get Keycloak auth endpoint URL. "+err.Error()))
	}

	tokenEndpoint, err := c.configuration.GetKeycloakEndpointToken(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "Unable to get Keycloak token endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to get Keycloak token endpoint URL. "+err.Error()))
	}

	entitlementEndpoint, err := c.configuration.GetKeycloakEndpointEntitlement(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "Unable to get Keycloak entitlement endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to get Keycloak entitlement endpoint URL. "+err.Error()))
	}

	brokerEndpoint, err := c.configuration.GetKeycloakEndpointBroker(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "Unable to get Keycloak broker endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to get Keycloak broker endpoint URL. "+err.Error()))
	}
	profileEndpoint, err := c.configuration.GetKeycloakAccountEndpoint(ctx.RequestData)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(err.Error()))
	}
	whitelist, err := c.configuration.GetValidRedirectURLs(ctx.RequestData)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(err.Error()))
	}

	oauth := &oauth2.Config{
		ClientID:     c.configuration.GetKeycloakClientID(),
		ClientSecret: c.configuration.GetKeycloakSecret(),
		Scopes:       []string{"user:email"},
		Endpoint:     oauth2.Endpoint{AuthURL: authEndpoint, TokenURL: tokenEndpoint},
		RedirectURL:  rest.AbsoluteURL(ctx.RequestData, "/api/login/authorize"),
	}

	return c.auth.Perform(ctx, oauth, brokerEndpoint, entitlementEndpoint, profileEndpoint, whitelist)
}

// Refresh obtain a new access token using the refresh token.
func (c *LoginController) Refresh(ctx *app.RefreshLoginContext) error {
	refreshToken := ctx.Payload.RefreshToken
	if refreshToken == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("refresh_token", nil).Expected("not nil"))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	endpoint, err := c.configuration.GetKeycloakEndpointToken(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "Unable to get Keycloak token endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to get Keycloak token endpoint URL "+err.Error()))
	}
	res, err := client.PostForm(endpoint, url.Values{
		"client_id":     {c.configuration.GetKeycloakClientID()},
		"client_secret": {c.configuration.GetKeycloakSecret()},
		"refresh_token": {*refreshToken},
		"grant_type":    {"refresh_token"},
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("Error when obtaining token "+err.Error()))
	}
	switch res.StatusCode {
	case 200:
		// OK
	case 401:
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(res.Status+" "+rest.ReadBody(res.Body)))
	case 400:
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError(rest.ReadBody(res.Body), nil))
	default:
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(res.Status+" "+rest.ReadBody(res.Body)))
	}

	token, err := auth.ReadToken(res)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	entitlementEndpoint, err := c.configuration.GetKeycloakEndpointEntitlement(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "Unable to get Keycloak token endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to get Keycloak token endpoint URL "+err.Error()))
	}

	rpt, err := auth.GetEntitlement(ctx, entitlementEndpoint, nil, *token.AccessToken)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "failed to obtain entitlement during login")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	if rpt != nil {
		// Swap access token and rpt which contains all resources available to the user
		token.AccessToken = rpt
	}

	return ctx.OK(convertToken(*token))
}

func convertToken(token auth.Token) *app.AuthToken {
	return &app.AuthToken{Token: &app.TokenData{
		AccessToken:      token.AccessToken,
		ExpiresIn:        token.ExpiresIn,
		NotBeforePolicy:  token.NotBeforePolicy,
		RefreshExpiresIn: token.RefreshExpiresIn,
		RefreshToken:     token.RefreshToken,
		TokenType:        token.TokenType,
	}}
}

// Link links identity provider(s) to the user's account
func (c *LoginController) Link(ctx *app.LinkLoginContext) error {
	brokerEndpoint, err := c.configuration.GetKeycloakEndpointBroker(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "Unable to get Keycloak broker endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to get Keycloak broker endpoint URL "+err.Error()))
	}
	clientID := c.configuration.GetKeycloakClientID()
	whitelist, err := c.configuration.GetValidRedirectURLs(ctx.RequestData)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(err.Error()))
	}

	return c.auth.Link(ctx, brokerEndpoint, clientID, whitelist)
}

// Linksession links identity provider(s) to the user's account
func (c *LoginController) Linksession(ctx *app.LinksessionLoginContext) error {
	brokerEndpoint, err := c.configuration.GetKeycloakEndpointBroker(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "Unable to get Keycloak broker endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to get Keycloak broker endpoint URL "+err.Error()))
	}
	clientID := c.configuration.GetKeycloakClientID()
	whitelist, err := c.configuration.GetValidRedirectURLs(ctx.RequestData)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(err.Error()))
	}

	return c.auth.LinkSession(ctx, brokerEndpoint, clientID, whitelist)
}

// Linkcallback redirects to original referel when Identity Provider account are linked to the user account
func (c *LoginController) Linkcallback(ctx *app.LinkcallbackLoginContext) error {
	brokerEndpoint, err := c.configuration.GetKeycloakEndpointBroker(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "Unable to get Keycloak broker endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to get Keycloak broker endpoint URL "+err.Error()))
	}
	clientID := c.configuration.GetKeycloakClientID()

	return c.auth.LinkCallback(ctx, brokerEndpoint, clientID)
}

// Generate obtain the access token from Keycloak for the test user
func (c *LoginController) Generate(ctx *app.GenerateLoginContext) error {
	var tokens app.AuthTokenCollection

	tokenEndpoint, err := c.configuration.GetKeycloakEndpointToken(ctx.RequestData)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to get Keycloak token endpoint URL "+err.Error()))
	}

	testuser, err := GenerateUserToken(ctx, tokenEndpoint, c.configuration, c.configuration.GetKeycloakTestUserName(), c.configuration.GetKeycloakTestUserSecret())
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to generate test token "+err.Error()))
	}
	// Creates the testuser user and identity if they don't yet exist
	profileEndpoint, err := c.configuration.GetKeycloakAccountEndpoint(ctx.RequestData)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(err.Error()))
	}
	c.auth.CreateOrUpdateKeycloakUser(*testuser.Token.AccessToken, ctx, profileEndpoint)
	tokens = append(tokens, testuser)

	testuser, err = GenerateUserToken(ctx, tokenEndpoint, c.configuration, c.configuration.GetKeycloakTestUser2Name(), c.configuration.GetKeycloakTestUser2Secret())
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to generate test token "+err.Error()))
	}
	// Creates the testuser2 user and identity if they don't yet exist
	c.auth.CreateOrUpdateKeycloakUser(*testuser.Token.AccessToken, ctx, profileEndpoint)
	tokens = append(tokens, testuser)

	// jsonapi.JSONErrorResponse(ctx, errors.NewInternalError("unable to get Keycloak token endpoint URL "+err.Error()))
	return ctx.OK(tokens)
}

// GenerateUserToken obtains the access token from Keycloak for the user
func GenerateUserToken(ctx context.Context, tokenEndpoint string, configuration loginConfiguration, username string, userSecret string) (*app.AuthToken, error) {
	if !configuration.IsPostgresDeveloperModeEnabled() {
		log.Error(ctx, map[string]interface{}{
			"method": "Generate",
		}, "Postgres developer mode not enabled")
		return nil, errors.NewInternalError("Postgres developer mode is not enabled")
	}

	var scopes []account.Identity
	scopes = append(scopes, test.TestIdentity)
	scopes = append(scopes, test.TestObserverIdentity)

	client := &http.Client{Timeout: 10 * time.Second}

	res, err := client.PostForm(tokenEndpoint, url.Values{
		"client_id":     {configuration.GetKeycloakClientID()},
		"client_secret": {configuration.GetKeycloakSecret()},
		"username":      {username},
		"password":      {userSecret},
		"grant_type":    {"password"},
	})
	if err != nil {
		return nil, errors.NewInternalError("error when obtaining token " + err.Error())
	}

	token, err := auth.ReadToken(res)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"token_endpoint": res,
			"err":            err,
		}, "Error when unmarshal json with access token")
		return nil, errors.NewInternalError("error when unmarshal json with access token " + err.Error())
	}

	return convertToken(*token), nil
}
