package main

import (
	"fmt"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
)

// UserController implements the user resource.
type UserController struct {
	*goa.Controller
	identityRepository account.IdentityRepository
	tokenManager       token.Manager
}

// NewUserController creates a user controller.
func NewUserController(service *goa.Service, identityRepository account.IdentityRepository, tokenManager token.Manager) *UserController {
	return &UserController{Controller: service.NewController("UserController"), identityRepository: identityRepository, tokenManager: tokenManager}
}

// Show returns the authorized user based on the provided Token
func (c *UserController) Show(ctx *app.ShowUserContext) error {
	identID, err := c.tokenManager.Locate(ctx)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(err.Error()))
		return ctx.BadRequest(jerrors)
	}
	ident, err := c.identityRepository.Load(ctx, identID)
	if err != nil {
		fmt.Printf("Auth token contains id %s of unknown Identity\n", identID)
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(fmt.Sprintf("Auth token contains id %s of unknown Identity\n", identID)))
		return ctx.Unauthorized(jerrors)
	}

	return ctx.OK(ident.ConvertIdentityFromModel())
}
