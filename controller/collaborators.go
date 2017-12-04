package controller

import (
	"context"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/rest/proxy"
	"github.com/goadesign/goa"
)

// CollaboratorsController implements the collaborators resource.
type CollaboratorsController struct {
	*goa.Controller
	config CollaboratorsConfiguration
}

type CollaboratorsConfiguration interface {
	auth.ServiceConfiguration
	GetKeycloakEndpointEntitlement(*http.Request) (string, error)
	GetCacheControlCollaborators() string
}

type collaboratorContext interface {
	context.Context
	jsonapi.InternalServerError
}

// NewCollaboratorsController creates a collaborators controller.
func NewCollaboratorsController(service *goa.Service, config CollaboratorsConfiguration) *CollaboratorsController {
	return &CollaboratorsController{Controller: service.NewController("CollaboratorsController"), config: config}
}

// List collaborators for the given space ID.
func (c *CollaboratorsController) List(ctx *app.ListCollaboratorsContext) error {
	return proxy.RouteHTTP(ctx, c.config.GetAuthShortServiceHostName())
}

// Add user's identity to the list of space collaborators.
func (c *CollaboratorsController) Add(ctx *app.AddCollaboratorsContext) error {
	return proxy.RouteHTTP(ctx, c.config.GetAuthShortServiceHostName())
}

// AddMany adds user's identities to the list of space collaborators.
func (c *CollaboratorsController) AddMany(ctx *app.AddManyCollaboratorsContext) error {
	return proxy.RouteHTTP(ctx, c.config.GetAuthShortServiceHostName())
}

// Remove user from the list of space collaborators.
func (c *CollaboratorsController) Remove(ctx *app.RemoveCollaboratorsContext) error {
	return proxy.RouteHTTP(ctx, c.config.GetAuthShortServiceHostName())
}

// RemoveMany removes users from the list of space collaborators.
func (c *CollaboratorsController) RemoveMany(ctx *app.RemoveManyCollaboratorsContext) error {
	return proxy.RouteHTTP(ctx, c.config.GetAuthShortServiceHostName())
}
