package controller

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/account/tenant"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// UserServiceController implements the UserService resource.
type UserServiceController struct {
	*goa.Controller
	UpdateTenant func(context.Context) error
	CleanTenant  func(context.Context) error
	ShowTenant   func(context.Context) (*tenant.TenantSingle, error)
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

// Clean runs the clean action.
func (c *UserServiceController) Clean(ctx *app.CleanUserServiceContext) error {
	err := c.CleanTenant(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.OK([]byte{})
}

// Show runs the show action.
func (c *UserServiceController) Show(ctx *app.ShowUserServiceContext) error {
	t, err := c.ShowTenant(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	return ctx.OK(convert(t))
}

func convert(t *tenant.TenantSingle) *app.UserServiceSingle {
	var ns []*app.NamespaceAttributes
	for _, tn := range t.Data.Attributes.Namespaces {
		ns = append(ns, &app.NamespaceAttributes{
			CreatedAt:  tn.CreatedAt,
			UpdatedAt:  tn.UpdatedAt,
			Name:       tn.Name,
			State:      tn.State,
			Version:    tn.Version,
			Type:       tn.Type,
			ClusterURL: tn.ClusterURL,
		})
	}
	id := uuid.UUID(*t.Data.ID)
	u := app.UserServiceSingle{
		Data: &app.UserService{
			Attributes: &app.UserServiceAttributes{
				CreatedAt:  t.Data.Attributes.CreatedAt,
				Namespaces: ns,
			},
			ID:   &id,
			Type: "userservices",
		},
	}
	return &u
}
