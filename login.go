package main

import (
	"fmt"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// LoginController implements the login resource.
type LoginController struct {
	*goa.Controller
	auth         login.Service
	tokenManager token.Manager
}

// NewLoginController creates a login controller.
func NewLoginController(service *goa.Service, auth login.Service, tokenManager token.Manager) *LoginController {
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
	scopes = append(scopes, account.Identity{
		ID:       uuid.NewV4(),
		Username: "TestDeveloper",
	})
	scopes = append(scopes, account.Identity{
		ID:       uuid.NewV4(),
		Username: "TestObserver",
	})

	var tokens app.AuthTokenCollection
	for _, user := range scopes {
		tokenStr, err := c.tokenManager.Generate(user)
		if err != nil {
			fmt.Println("Failed to generate token", err)
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(fmt.Sprintf("Failed to generate token: %s", err.Error())))
			return ctx.Unauthorized(jerrors)
		}
		tokens = append(tokens, &app.AuthToken{Token: tokenStr})
	}
	return ctx.OK(tokens)
}
