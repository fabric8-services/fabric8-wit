package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"net/http"
	"net/url"

	"fmt"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	"github.com/pkg/errors"
)

// LoginController implements the login resource.
type LoginController struct {
	*goa.Controller
	auth         login.KeycloakOAuthService
	tokenManager token.Manager
}

type tokenJSON struct {
	AccessToken string `json:"access_token"`
}

// NewLoginController creates a login controller.
func NewLoginController(service *goa.Service, auth *login.KeycloakOAuthProvider, tokenManager token.Manager) *LoginController {
	return &LoginController{Controller: service.NewController("login"), auth: auth, tokenManager: tokenManager}
}

// Authorize runs the authorize action.
func (c *LoginController) Authorize(ctx *app.AuthorizeLoginContext) error {
	return c.auth.Perform(ctx)
}

// Generate runs the authorize action.
func (c *LoginController) Generate(ctx *app.GenerateLoginContext) error {
	if !configuration.IsPostgresDeveloperModeEnabled() {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized("Postgres developer mode not enabled"))
		return ctx.Unauthorized(jerrors)
	}

	var scopes []account.Identity
	scopes = append(scopes, test.TestIdentity)
	scopes = append(scopes, test.TestObserverIdentity)

	client := &http.Client{Timeout: 10 * time.Second}

	username := configuration.GetKeycloakTestUserName()
	res, err := client.PostForm(configuration.GetKeycloakEndpointToken(), url.Values{
		"client_id":     {configuration.GetKeycloakClientID()},
		"client_secret": {configuration.GetKeycloakSecret()},
		"username":      {username},
		"password":      {configuration.GetKeycloakTestUserSecret()},
		"grant_type":    {"password"},
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.Wrap(err, "Error when obtaining token"))
	}

	// Read the json out of the response body
	buf := new(bytes.Buffer)
	io.Copy(buf, res.Body)
	res.Body.Close()
	jsonString := strings.TrimSpace(buf.String())

	var token tokenJSON
	err = json.Unmarshal([]byte(jsonString), &token)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.Wrap(err, fmt.Sprintf("Error when unmarshal json with access token %s", jsonString)))
	}
	if token.AccessToken == "" {
		return jsonapi.JSONErrorResponse(ctx, errors.Wrap(err, fmt.Sprintf("Can't obtain access token from %s", jsonString)))
	}
	var tokens app.AuthTokenCollection
	tokens = append(tokens, &app.AuthToken{Token: token.AccessToken})

	c.auth.CreateKeycloakUser(token.AccessToken, ctx)

	return ctx.OK(tokens)
}
