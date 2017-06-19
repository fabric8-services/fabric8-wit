package controller

import (
	"context"

	"github.com/almighty/almighty-core/app"
	"github.com/goadesign/goa"
)

// UserServiceController implements the UserService resource.
type UserServiceController struct {
	*goa.Controller
	UpdateTenant         func(context.Context) error
	CreateTenant         func(context.Context) error
	CurrentTenantVersion func(context.Context) error // currently deployed version
	LatestTenantVersion  func(context.Context) error // latest available version
}

// NewUserServiceController creates a UserService controller.
func NewUserServiceController(service *goa.Service) *UserServiceController {
	return &UserServiceController{Controller: service.NewController("UserServiceController")}
}

// Update runs the update action.
func (c *UserServiceController) Update(ctx *app.UpdateUserServiceContext) error {
	c.UpdateTenant(ctx)
	return ctx.OK([]byte{})
}

// Create initializes the tenant services for the user
func (c *UserServiceController) Create(ctx *app.CreateUserServiceContext) error {
	c.CreateTenant(ctx)
	return ctx.OK([]byte{})
}

// ShowCurrent gets the current version of tenant services deployed for the user.
func (c *UserServiceController) Current(ctx *app.CurrentUserServiceContext) error {
	// fill.
	// pipelines UI is currently calling OSO directly to get this info
	return ctx.OK([]byte{})
}

// ShowLatest gets the latest version available for the user to deploy.
func (c *UserServiceController) Latest(ctx *app.LatestUserServiceContext) error {
	// fill
	return ctx.OK([]byte{})
}
