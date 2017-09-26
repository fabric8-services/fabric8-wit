package controller

import (
	"context"
	"encoding/json"
	er "errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/client"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/test/token"

	"github.com/goadesign/goa"
	goaclient "github.com/goadesign/goa/client"
	errs "github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

type loginConfiguration interface {
	auth.AuthServiceConfiguration
	IsPostgresDeveloperModeEnabled() bool
	GetKeycloakTestUserName() string
	GetKeycloakTestUser2Name() string
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
	if !c.configuration.IsAuthorizationEnabled() {
		// Login as test user
		redirects := ctx.Params["redirect"]
		var redirect string
		referrer := ctx.Request.Header.Get("Referer")
		if redirects == nil {
			if referrer == "" {
				return jsonapi.JSONErrorResponse(ctx, er.New("referrer header and redirect param are both empty; at least one should be specified"))
			}
			redirect = referrer
		} else if len(redirects) > 0 {
			redirect = redirects[0]
		}

		cln := client.New(goaclient.HTTPClientDoer(http.DefaultClient))
		cln.Host = ctx.Request.Host
		cln.Scheme = ctx.URL.Scheme
		res, err := cln.GenerateLogin(goasupport.ForwardContextRequestID(ctx), client.GenerateLoginPath())
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		tokens, err := cln.DecodeAuthTokenCollection(res)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		tokenData := &app.TokenData{
			AccessToken:      tokens[0].Token.AccessToken,
			ExpiresIn:        tokens[0].Token.ExpiresIn,
			RefreshToken:     tokens[0].Token.RefreshToken,
			RefreshExpiresIn: tokens[0].Token.RefreshExpiresIn,
			NotBeforePolicy:  tokens[0].Token.NotBeforePolicy,
			TokenType:        tokens[0].Token.TokenType,
		}
		b, err := json.Marshal(tokenData)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		location, err := url.Parse(redirect)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		parameters := location.Query()
		parameters.Add("token_json", string(b))
		location.RawQuery = parameters.Encode()
		ctx.ResponseData.Header().Set("Location", location.String())
		return ctx.TemporaryRedirect()
	}
	return redirectWithParams(ctx, c.configuration, ctx.ResponseData.Header(), ctx.Params, authservice.LoginLoginPath())
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

func redirectWithParams(ctx redirectContext, config auth.AuthServiceConfiguration, header http.Header, params url.Values, path string) error {
	locationURL := fmt.Sprintf("%s%s", config.GetAuthServiceURL(), path)
	locationURLWithParams, err := redirectLocation(params, locationURL)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}
	header.Set("Location", locationURLWithParams)
	return ctx.TemporaryRedirect()
}

// Refresh obtain a new access token using the refresh token.
func (c *LoginController) Refresh(ctx *app.RefreshLoginContext) error {
	if !c.configuration.IsAuthorizationEnabled() {
		return jsonapi.JSONErrorResponse(ctx, er.New("token refresh endpoint disabled"))
	}
	ctx.ResponseData.Header().Set("Location", fmt.Sprintf("%s%s", c.configuration.GetAuthServiceURL(), authservice.RefreshTokenPath()))
	return ctx.TemporaryRedirect()
}

// Link links identity provider(s) to the user's account
func (c *LoginController) Link(ctx *app.LinkLoginContext) error {
	return redirectWithParams(ctx, c.configuration, ctx.ResponseData.Header(), ctx.Params, authservice.LinkLinkPath())
}

// Linksession links identity provider(s) to the user's account
func (c *LoginController) Linksession(ctx *app.LinksessionLoginContext) error {
	return redirectWithParams(ctx, c.configuration, ctx.ResponseData.Header(), ctx.Params, authservice.SessionLinkPath())
}

// Generate generates access tokens in Dev Mode
func (c *LoginController) Generate(ctx *app.GenerateLoginContext) error {
	var tokens app.AuthTokenCollection

	testuser, err := generateUserToken(ctx, c.configuration, c.configuration.GetKeycloakTestUserName())
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":      err,
			"username": c.configuration.GetKeycloakTestUserName(),
		}, "unable to get Generate User token")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to generate test token ")))
	}
	// Creates the testuser user and identity if they don't yet exist
	c.auth.CreateOrUpdateKeycloakUser(*testuser.Token.AccessToken, ctx)
	tokens = append(tokens, testuser)

	testuser, err = generateUserToken(ctx, c.configuration, c.configuration.GetKeycloakTestUser2Name())
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":      err,
			"username": c.configuration.GetKeycloakTestUser2Name(),
		}, "unable to generate test token")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to generate test token")))
	}
	// Creates the testuser2 user and identity if they don't yet exist
	c.auth.CreateOrUpdateKeycloakUser(*testuser.Token.AccessToken, ctx)
	tokens = append(tokens, testuser)

	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
	return ctx.OK(tokens)
}

func generateUserToken(ctx context.Context, configuration loginConfiguration, username string) (*app.AuthToken, error) {
	if !configuration.IsPostgresDeveloperModeEnabled() {
		log.Error(ctx, map[string]interface{}{
			"method": "Generate",
		}, "Developer mode not enabled")
		return nil, errors.NewInternalError(ctx, errs.New("postgres developer mode is not enabled"))
	}
	t, err := token.GenerateToken(uuid.NewV4().String(), username, token.PrivateKey())
	if err != nil {
		return nil, err
	}
	bearer := "Bearer"
	return &app.AuthToken{Token: &app.TokenData{
		AccessToken:      &t,
		ExpiresIn:        60 * 60 * 24 * 30,
		NotBeforePolicy:  0,
		RefreshExpiresIn: 60 * 60 * 24 * 30,
		RefreshToken:     &t,
		TokenType:        &bearer,
	}}, nil
}
