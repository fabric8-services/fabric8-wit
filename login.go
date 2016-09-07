package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/login"
	"github.com/goadesign/goa"
)

// LoginController implements the login resource.
type LoginController struct {
	*goa.Controller
	auth login.Service
}

// NewLoginController creates a login controller.
func NewLoginController(service *goa.Service, auth login.Service) *LoginController {
	return &LoginController{Controller: service.NewController("login"), auth: auth}
}

// Authorize runs the authorize action.
func (c *LoginController) Authorize(ctx *app.AuthorizeLoginContext) error {
	/*
		token := jwt.New(jwt.SigningMethodRS256)
		token.Claims.(jwt.MapClaims)["exp"] = time.Now().Add(time.Hour * 72).Unix()
		token.Claims.(jwt.MapClaims)["scopes"] = []string{"system"}

		key, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSAPrivateKey)))
		if err != nil {
			panic(err)
		}

		tokenStr, err := token.SignedString(key)
		if err != nil {
			panic(err)
		}
		authToken := app.AuthToken{Token: tokenStr}
		return ctx.OK(&authToken)
	*/
	return c.auth.Perform(ctx)
}

// Generate runs the authorize action.
func (c *LoginController) Generate(ctx *app.GenerateLoginContext) error {
	if !Development {
		return ctx.Unauthorized()
	}

	type User struct {
		Name   string
		Scopes []string
	}
	/*
		var scopes []User
		scopes = append(scopes, User{
			Name:   "Test Developer",
			Scopes: Permissions.CRUDWotkItem(),
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

			key, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSAPrivateKey)))
			if err != nil {
				panic(err)
			}

			tokenStr, err := token.SignedString(key)
			if err != nil {
				panic(err)
			}
			tokens = append(tokens, &app.AuthToken{Token: tokenStr})

		}
	*/
	return ctx.OK(app.AuthTokenCollection{})
}
