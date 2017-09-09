package controller

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"

	"net/http"
	"net/url"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/rest"
	errs "github.com/pkg/errors"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/token"
	"github.com/goadesign/goa"
)

type loginConfiguration interface {
	GetKeycloakEndpointAuth(*http.Request) (string, error)
	GetKeycloakEndpointToken(*http.Request) (string, error)
	GetKeycloakAccountEndpoint(req *http.Request) (string, error)
	GetKeycloakEndpointBroker(*http.Request) (string, error)
	GetKeycloakEndpointEntitlement(*http.Request) (string, error)
	GetKeycloakClientID() string
	GetKeycloakSecret() string
	IsPostgresDeveloperModeEnabled() bool
	GetKeycloakTestUserName() string
	GetKeycloakTestUserSecret() string
	GetKeycloakTestUser2Name() string
	GetKeycloakTestUser2Secret() string
	GetValidRedirectURLs(*http.Request) (string, error)
	GetHeaderMaxLength() int64
	GetAuthNotApprovedRedirect() string
}

const maxRecentSpacesForRPT = 10

// LoginController implements the login resource.
type LoginController struct {
	*goa.Controller
	auth               login.KeycloakOAuthService
	tokenManager       token.Manager
	configuration      loginConfiguration
	identityRepository account.IdentityRepository
}

// NewLoginController creates a login controller.
func NewLoginController(service *goa.Service, auth *login.KeycloakOAuthProvider, tokenManager token.Manager, configuration loginConfiguration, identityRepository account.IdentityRepository) *LoginController {
	return &LoginController{Controller: service.NewController("login"), auth: auth, tokenManager: tokenManager, configuration: configuration, identityRepository: identityRepository}
}

// Authorize runs the authorize action.
func (c *LoginController) Authorize(ctx *app.AuthorizeLoginContext) error {
	authEndpoint, err := c.configuration.GetKeycloakEndpointAuth(ctx.Request)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak auth endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak auth endpoint URL")))
	}

	tokenEndpoint, err := c.configuration.GetKeycloakEndpointToken(ctx.Request)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak token endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak token endpoint URL")))
	}

	entitlementEndpoint, err := c.configuration.GetKeycloakEndpointEntitlement(ctx.Request)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak entitlement endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak entitlement endpoint URL")))
	}

	brokerEndpoint, err := c.configuration.GetKeycloakEndpointBroker(ctx.Request)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak broker endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak broker endpoint URL")))
	}
	profileEndpoint, err := c.configuration.GetKeycloakAccountEndpoint(ctx.Request)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak account endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}
	whitelist, err := c.configuration.GetValidRedirectURLs(ctx.Request)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}

	if ctx.Scope != nil {
		authEndpoint = fmt.Sprintf("%s?scope=%s", authEndpoint, *ctx.Scope) // Offline token
	}
	oauth := &oauth2.Config{
		ClientID:     c.configuration.GetKeycloakClientID(),
		ClientSecret: c.configuration.GetKeycloakSecret(),
		Scopes:       []string{"user:email"},
		Endpoint:     oauth2.Endpoint{AuthURL: authEndpoint, TokenURL: tokenEndpoint},
		RedirectURL:  rest.AbsoluteURL(ctx.Request, "/api/login/authorize"),
	}

	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
	return c.auth.Perform(ctx, oauth, brokerEndpoint, entitlementEndpoint, profileEndpoint, whitelist, c.configuration.GetAuthNotApprovedRedirect())
}

// Refresh obtain a new access token using the refresh token.
func (c *LoginController) Refresh(ctx *app.RefreshLoginContext) error {
	refreshToken := ctx.Payload.RefreshToken
	if refreshToken == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("refresh_token", nil).Expected("not nil"))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	endpoint, err := c.configuration.GetKeycloakEndpointToken(ctx.Request)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak token endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak token endpoint URL")))
	}
	res, err := client.PostForm(endpoint, url.Values{
		"client_id":     {c.configuration.GetKeycloakClientID()},
		"client_secret": {c.configuration.GetKeycloakSecret()},
		"refresh_token": {*refreshToken},
		"grant_type":    {"refresh_token"},
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "error when obtaining token")))
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case 200:
		// OK
	case 401:
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(res.Status+" "+rest.ReadBody(res.Body)))
	case 400:
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(res.Status+" "+rest.ReadBody(res.Body)))
	default:
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.New(res.Status+" "+rest.ReadBody(res.Body))))
	}

	token, err := auth.ReadToken(ctx, res)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
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
	brokerEndpoint, err := c.configuration.GetKeycloakEndpointBroker(ctx.Request)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak broker endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak broker endpoint URL")))
	}
	clientID := c.configuration.GetKeycloakClientID()
	whitelist, err := c.configuration.GetValidRedirectURLs(ctx.Request)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}

	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
	return c.auth.Link(ctx, brokerEndpoint, clientID, whitelist)
}

// Linksession links identity provider(s) to the user's account
func (c *LoginController) Linksession(ctx *app.LinksessionLoginContext) error {
	brokerEndpoint, err := c.configuration.GetKeycloakEndpointBroker(ctx.Request)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak broker endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak broker endpoint URL")))
	}
	clientID := c.configuration.GetKeycloakClientID()
	whitelist, err := c.configuration.GetValidRedirectURLs(ctx.Request)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}

	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
	return c.auth.LinkSession(ctx, brokerEndpoint, clientID, whitelist)
}

// Linkcallback redirects to original referel when Identity Provider account are linked to the user account
func (c *LoginController) Linkcallback(ctx *app.LinkcallbackLoginContext) error {
	brokerEndpoint, err := c.configuration.GetKeycloakEndpointBroker(ctx.Request)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak broker endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak broker endpoint URL ")))
	}
	clientID := c.configuration.GetKeycloakClientID()

	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
	return c.auth.LinkCallback(ctx, brokerEndpoint, clientID)
}

// Generate obtain the access token from Keycloak for the test user
func (c *LoginController) Generate(ctx *app.GenerateLoginContext) error {
	var tokens app.AuthTokenCollection

	tokenEndpoint, err := c.configuration.GetKeycloakEndpointToken(ctx.Request)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak token endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak token endpoint URL")))
	}

	testuser, err := GenerateUserToken(ctx, tokenEndpoint, c.configuration, c.configuration.GetKeycloakTestUserName(), c.configuration.GetKeycloakTestUserSecret())
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":      err,
			"username": c.configuration.GetKeycloakTestUserName(),
		}, "unable to get Generate User token")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to generate test token ")))
	}
	// Creates the testuser user and identity if they don't yet exist
	profileEndpoint, err := c.configuration.GetKeycloakAccountEndpoint(ctx.Request)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak account endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}
	c.auth.CreateOrUpdateKeycloakUser(*testuser.Token.AccessToken, ctx, profileEndpoint)
	tokens = append(tokens, testuser)

	testuser, err = GenerateUserToken(ctx, tokenEndpoint, c.configuration, c.configuration.GetKeycloakTestUser2Name(), c.configuration.GetKeycloakTestUser2Secret())
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":      err,
			"username": c.configuration.GetKeycloakTestUser2Name(),
		}, "unable to generate test token")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to generate test token")))
	}
	// Creates the testuser2 user and identity if they don't yet exist
	c.auth.CreateOrUpdateKeycloakUser(*testuser.Token.AccessToken, ctx, profileEndpoint)
	tokens = append(tokens, testuser)

	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
	return ctx.OK(tokens)
}

// GenerateUserToken obtains the access token from Keycloak for the user
func GenerateUserToken(ctx context.Context, tokenEndpoint string, configuration loginConfiguration, username string, userSecret string) (*app.AuthToken, error) {
	if !configuration.IsPostgresDeveloperModeEnabled() {
		log.Error(ctx, map[string]interface{}{
			"method": "Generate",
		}, "Postgres developer mode not enabled")
		return nil, errors.NewInternalError(ctx, errs.New("postgres developer mode is not enabled"))
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
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "error when obtaining token"))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Error(ctx, map[string]interface{}{
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
			"username":        username,
		}, "unable to obtain token")
		return nil, errors.NewInternalError(ctx, errs.Errorf("unable to obtain token. Response status: %s. Responce body: %s", res.Status, rest.ReadBody(res.Body)))
	}
	token, err := auth.ReadToken(ctx, res)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"token_endpoint": res,
			"err":            err,
			"username":       username,
		}, "error when unmarshal json with access token")
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "error when unmarshal json with access token"))
	}

	return convertToken(*token), nil
}
