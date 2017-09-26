package controller

import (
	"context"
	"fmt"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"

	"github.com/goadesign/goa"
)

// UserController implements the user resource.
type UserController struct {
	*goa.Controller
	config     UserControllerConfiguration
	InitTenant func(context.Context) error
}

// UserControllerConfiguration the configuration for the UserController
type UserControllerConfiguration interface {
	auth.AuthServiceConfiguration
	GetCacheControlUser() string
}

// NewUserController creates a user controller.
func NewUserController(service *goa.Service, config UserControllerConfiguration) *UserController {
	return &UserController{
		Controller: service.NewController("UserController"),
		config:     config,
	}
}

// Show returns the authorized user based on the provided Token
func (c *UserController) Show(ctx *app.ShowUserContext) error {
	if c.InitTenant != nil {
		go func(ctx context.Context) {
			c.InitTenant(ctx)
		}(ctx)
	}
	ctx.ResponseData.Header().Set("Location", fmt.Sprintf("%s%s", c.config.GetAuthServiceURL(), authservice.ShowUserPath()))
	return ctx.TemporaryRedirect()
}
