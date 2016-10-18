package main

import (
	"time"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/configuration"
	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
)

// LoginController implements the login resource.
type LoginController struct {
	*goa.Controller
}

// NewLoginController creates a login controller.
func NewLoginController(service *goa.Service) *LoginController {
	return &LoginController{Controller: service.NewController("login")}
}

// Authorize runs the authorize action.
func (c *LoginController) Authorize(ctx *app.AuthorizeLoginContext) error {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims.(jwt.MapClaims)["exp"] = time.Now().Add(time.Hour * 72).Unix()
	token.Claims.(jwt.MapClaims)["scopes"] = []string{"system"}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(configuration.GetTokenPrivateKey())
	if err != nil {
		panic(err)
	}

	tokenStr, err := token.SignedString(key)
	if err != nil {
		panic(err)
	}
	authToken := app.AuthToken{Token: tokenStr}
	return ctx.OK(&authToken)
}

// Generate runs the authorize action.
func (c *LoginController) Generate(ctx *app.GenerateLoginContext) error {
	if !configuration.IsPostgresDeveloperModeEnabled() {
		return ctx.Unauthorized()
	}

	type User struct {
		Name   string
		Scopes []string
	}

	var scopes []User
	scopes = append(scopes, User{
		Name:   "Test Developer",
		Scopes: Permissions.CRUDWorkItem(),
	})
	scopes = append(scopes, User{
		Name:   "Test Observer",
		Scopes: []string{Permissions.ReadWorkItem},
	})

	var tokens app.AuthTokenCollection
	for _, user := range scopes {
		token := jwt.New(jwt.SigningMethodRS256)

		token.Claims.(jwt.MapClaims)["exp"] = time.Now().Add(time.Hour * 72).Unix()
		token.Claims.(jwt.MapClaims)["scopes"] = user.Scopes
		token.Claims.(jwt.MapClaims)["name"] = user.Name

		key, err := jwt.ParseRSAPrivateKeyFromPEM(configuration.GetTokenPrivateKey())
		if err != nil {
			panic(err)
		}

		tokenStr, err := token.SignedString(key)
		if err != nil {
			panic(err)
		}
		tokens = append(tokens, &app.AuthToken{Token: tokenStr})

	}
	return ctx.OK(tokens)
}
