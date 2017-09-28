package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/goadesign/goa"
	goauuid "github.com/goadesign/goa/uuid"
	"github.com/satori/go.uuid"
)

// CollaboratorsController implements the collaborators resource.
type CollaboratorsController struct {
	*goa.Controller
	db     application.DB
	config CollaboratorsConfiguration
}

type CollaboratorsConfiguration interface {
	auth.AuthServiceConfiguration
	GetKeycloakEndpointEntitlement(*http.Request) (string, error)
	GetCacheControlCollaborators() string
}

type collaboratorContext interface {
	context.Context
	jsonapi.InternalServerError
}

// NewCollaboratorsController creates a collaborators controller.
func NewCollaboratorsController(service *goa.Service, db application.DB, config CollaboratorsConfiguration) *CollaboratorsController {
	return &CollaboratorsController{Controller: service.NewController("CollaboratorsController"), db: db, config: config}
}

type redirectContext interface {
	context.Context
	TemporaryRedirect() error
}

// List collaborators for the given space ID.
func (c *CollaboratorsController) List(ctx *app.ListCollaboratorsContext) error {
	return c.redirect(ctx, ctx.ResponseData.Header(), ctx.Request, ctx.SpaceID, "")
}

func (c *CollaboratorsController) redirect(ctx redirectContext, header http.Header, request *http.Request, spaceID uuid.UUID, identityID string) error {
	sID, err := goauuid.FromString(spaceID.String())
	if err != nil {
		return err
	}
	var locationURL string
	if identityID != "" {
		locationURL = fmt.Sprintf("%s%s", c.config.GetAuthServiceURL(), authservice.AddCollaboratorsPath(sID, identityID))
	} else {
		locationURL = fmt.Sprintf("%s%s", c.config.GetAuthServiceURL(), authservice.AddManyCollaboratorsPath(sID))
	}
	header.Set("Location", locationURL)
	return ctx.TemporaryRedirect()
}

// Add user's identity to the list of space collaborators.
func (c *CollaboratorsController) Add(ctx *app.AddCollaboratorsContext) error {
	return c.redirect(ctx, ctx.ResponseData.Header(), ctx.Request, ctx.SpaceID, ctx.IdentityID)
}

// AddMany adds user's identities to the list of space collaborators.
func (c *CollaboratorsController) AddMany(ctx *app.AddManyCollaboratorsContext) error {
	return c.redirect(ctx, ctx.ResponseData.Header(), ctx.Request, ctx.SpaceID, "")
}

// Remove user from the list of space collaborators.
func (c *CollaboratorsController) Remove(ctx *app.RemoveCollaboratorsContext) error {
	return c.redirect(ctx, ctx.ResponseData.Header(), ctx.Request, ctx.SpaceID, ctx.IdentityID)
}

// RemoveMany removes users from the list of space collaborators.
func (c *CollaboratorsController) RemoveMany(ctx *app.RemoveManyCollaboratorsContext) error {
	return c.redirect(ctx, ctx.ResponseData.Header(), ctx.Request, ctx.SpaceID, "")
}
