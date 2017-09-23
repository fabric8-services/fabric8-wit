package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/auth"
	errs "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"

	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
)

// CollaboratorsController implements the collaborators resource.
type CollaboratorsController struct {
	*goa.Controller
	db            application.DB
	config        CollaboratorsConfiguration
	policyManager auth.AuthzPolicyManager
}

type CollaboratorsConfiguration interface {
	GetKeycloakEndpointEntitlement(*http.Request) (string, error)
	GetCacheControlCollaborators() string
	IsAuthorizationEnabled() bool
	GetAuthEndpointSpaces(req *http.Request) (string, error)
}

type collaboratorContext interface {
	context.Context
	jsonapi.InternalServerError
}

// NewCollaboratorsController creates a collaborators controller.
func NewCollaboratorsController(service *goa.Service, db application.DB, config CollaboratorsConfiguration, policyManager auth.AuthzPolicyManager) *CollaboratorsController {
	return &CollaboratorsController{Controller: service.NewController("CollaboratorsController"), db: db, config: config, policyManager: policyManager}
}

type redirectContext interface {
	context.Context
	TemporaryRedirect() error
}

// List collaborators for the given space ID.
func (c *CollaboratorsController) List(ctx *app.ListCollaboratorsContext) error {
	if c.config.IsAuthorizationEnabled() {
		authEndpoint, err := c.config.GetAuthEndpointSpaces(ctx.RequestData.Request)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.NewInternalError(ctx, err))
		}
		spaceID := ctx.SpaceID.String()
		locationURL, err := redirectLocation(ctx.Params, fmt.Sprintf("%s/%s/collaborators", authEndpoint, spaceID))
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.NewInternalError(ctx, err))
		}
		ctx.ResponseData.Header().Set("Location", locationURL)
		return ctx.TemporaryRedirect()
	}

	// Return the space owner if authZ is disabled (by default in Dev Mode)
	var userIDs string
	var ownerID string
	err := application.Transactional(c.db, func(appl application.Application) error {
		space, err := appl.Spaces().Load(ctx, ctx.SpaceID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"space_id": ctx.SpaceID,
				"err":      err,
			}, "unable to find the space")
			return err
		}
		ownerID = space.OwnerId.String()
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.NewInternalError(ctx.Context, err))
	}
	userIDs = fmt.Sprintf("[\"%s\"]", ownerID)
	log.Warn(ctx, map[string]interface{}{
		"space_id": ctx.SpaceID,
		"owner_id": ownerID,
	}, "Authorization is disabled. Space owner is the only collaborator")
	//UsersIDs format : "[\"<ID>\",\"<ID>\"]"
	s := strings.Split(userIDs, ",")
	count := len(s)

	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)

	pageOffset := offset
	pageLimit := offset + limit
	if offset > len(s) {
		pageOffset = len(s)
	}
	if offset+limit > len(s) {
		pageLimit = len(s)
	}
	page := s[pageOffset:pageLimit]
	resultIdentities := make([]account.Identity, len(page))
	resultUsers := make([]account.User, len(page))
	for i, id := range page {
		id = strings.Trim(id, "[]\"")
		uID, err := uuid.FromString(id)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"identity_id": id,
				"users-ids":   userIDs,
			}, "unable to convert the identity ID to uuid v4")
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}
		err = application.Transactional(c.db, func(appl application.Application) error {
			identities, err := appl.Identities().Query(account.IdentityFilterByID(uID), account.IdentityWithUser())
			if err != nil {
				log.Error(ctx, map[string]interface{}{
					"identity_id": id,
					"err":         err,
				}, "unable to find the identity listed in the space policy")
				return err
			}
			if len(identities) == 0 {
				log.Error(ctx, map[string]interface{}{
					"identity_id": id,
				}, "unable to find the identity listed in the space policy")
				return errors.New("identity listed in the space policy not found")
			}
			resultIdentities[i] = identities[0]
			resultUsers[i] = identities[0].User
			return nil
		})
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}
	}

	return ctx.ConditionalEntities(resultUsers, c.config.GetCacheControlCollaborators, func() error {
		data := make([]*app.UserData, len(page))
		for i := range resultUsers {
			appUser := ConvertToAppUser(ctx.Request, &resultUsers[i], &resultIdentities[i])
			data[i] = appUser.Data
		}
		response := app.UserList{
			Links: &app.PagingLinks{},
			Meta:  &app.UserListMeta{TotalCount: count},
			Data:  data,
		}
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.Request), len(page), offset, limit, count)
		return ctx.OK(&response)
	})
}

func (c *CollaboratorsController) redirect(ctx redirectContext, header http.Header, request *http.Request, spaceID uuid.UUID, identityID string) error {
	authEndpoint, err := c.config.GetAuthEndpointSpaces(request)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.NewInternalError(ctx, err))
	}
	locationURL := fmt.Sprintf("%s/%s/collaborators", authEndpoint, spaceID.String())
	if identityID != "" {
		locationURL = fmt.Sprintf("%s/%s", locationURL, identityID)
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
