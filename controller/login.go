package controller

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/token"

	"github.com/goadesign/goa"
	goaclient "github.com/goadesign/goa/client"
	errs "github.com/pkg/errors"
)

type loginConfiguration interface {
	GetKeycloakEndpointToken(*http.Request) (string, error)
	GetKeycloakAccountEndpoint(req *http.Request) (string, error)
	GetKeycloakClientID() string
	GetKeycloakSecret() string
	IsPostgresDeveloperModeEnabled() bool
	GetKeycloakTestUserName() string
	GetKeycloakTestUserSecret() string
	GetKeycloakTestUser2Name() string
	GetKeycloakTestUser2Secret() string
	GetAuthEndpointLogin(*http.Request) (string, error)
	GetAuthEndpointLink(req *http.Request) (string, error)
	GetAuthEndpointLinksession(req *http.Request) (string, error)
	GetAuthEndpointTokenRefresh(req *http.Request) (string, error)
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
	authEndpoint, err := c.configuration.GetAuthEndpointLogin(ctx.RequestData.Request)
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

func (c *LoginController) createRefreshClient(ctx *app.RefreshLoginContext) (*authservice.Client, error) {
	authEndpoint, err := c.configuration.GetAuthEndpointTokenRefresh(ctx.Request)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(authEndpoint)
	if err != nil {
		return nil, err
	}
	client := authservice.New(goaclient.HTTPClientDoer(http.DefaultClient))
	client.Host = u.Host
	client.Scheme = u.Scheme
	return client, nil
}

// Refresh obtain a new access token using the refresh token.
func (c *LoginController) Refresh(ctx *app.RefreshLoginContext) error {
	refreshToken := ctx.Payload.RefreshToken
	if refreshToken == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("refresh_token", nil).Expected("not nil"))
	}
	refreshTokenPayload := &authservice.RefreshToken{
		RefreshToken: refreshToken,
	}
	client, err := c.createRefreshClient(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	res, err := client.RefreshToken(goasupport.ForwardContextRequestID(ctx), authservice.RefreshTokenPath(), refreshTokenPayload)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to refresh token via auth service")
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "unable to refresh token via auth service"))
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case 200:
		// OK
	case 401:
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(res.Status+" "+rest.ReadBody(res.Body)))
	default:
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.New(res.Status+" "+rest.ReadBody(res.Body))))
	}

	token, err := client.DecodeAuthToken(res)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":           err,
			"response_body": rest.ReadBody(res.Body),
		}, "unable to decode refresh token request result")

		return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "unable to decode refresh token request result %s ", rest.ReadBody(res.Body)))
	}

	resultToken := &app.AuthToken{
		Token: &app.TokenData{
			AccessToken:      token.Token.AccessToken,
			ExpiresIn:        token.Token.ExpiresIn,
			RefreshToken:     token.Token.RefreshToken,
			RefreshExpiresIn: token.Token.RefreshExpiresIn,
			NotBeforePolicy:  token.Token.NotBeforePolicy,
			TokenType:        token.Token.TokenType,
		},
	}
	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
	return ctx.OK(resultToken)
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
	authEndpoint, err := c.configuration.GetAuthEndpointLink(ctx.RequestData.Request)
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

// Linksession links identity provider(s) to the user's account
func (c *LoginController) Linksession(ctx *app.LinksessionLoginContext) error {
	authEndpoint, err := c.configuration.GetAuthEndpointLinksession(ctx.RequestData.Request)
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
