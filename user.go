package main

import (
	"errors"
	"fmt"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	jwttoken "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware/security/jwt"
	uuid "github.com/satori/go.uuid"
)

// UserController implements the user resource.
type UserController struct {
	*goa.Controller
	identityRepository account.IdentityRepository
}

// NewUserController creates a user controller.
func NewUserController(service *goa.Service, identityRepository account.IdentityRepository) *UserController {
	return &UserController{Controller: service.NewController("UserController"), identityRepository: identityRepository}
}

// Show returns the authorized user based on the provided Token
func (c *UserController) Show(ctx *app.ShowUserContext) error {
	token := jwt.ContextJWT(ctx)
	id := token.Claims.(jwttoken.MapClaims)["uuid"]
	if id == nil {
		return ctx.BadRequest(errors.New("invalid token"))
	}
	idTyped, err := uuid.FromString(id.(string))
	if err != nil {
		return ctx.BadRequest(errors.New("invalid token"))
	}
	res := &app.User{}

	ident, err := c.identityRepository.Load(ctx, idTyped)
	if err != nil {
		fmt.Printf("Auth token contains id %s of unknown Identity\n", id)
		return ctx.Unauthorized()
	}

	res.FullName = &ident.FullName
	res.ImageURL = &ident.ImageURL

	return ctx.OK(res)
}
