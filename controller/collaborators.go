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
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
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
}

type collaboratorContext interface {
	context.Context
	jsonapi.InternalServerError
}

// NewCollaboratorsController creates a collaborators controller.
func NewCollaboratorsController(service *goa.Service, db application.DB, config CollaboratorsConfiguration, policyManager auth.AuthzPolicyManager) *CollaboratorsController {
	return &CollaboratorsController{Controller: service.NewController("CollaboratorsController"), db: db, config: config, policyManager: policyManager}
}

// List collaborators for the given space ID.
func (c *CollaboratorsController) List(ctx *app.ListCollaboratorsContext) error {
	var userIDs string
	if !c.config.IsAuthorizationEnabled() {
		// Return the space owner if authZ is disabled (by default in Dev Mode)
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
	} else {
		policy, _, err := c.getPolicy(ctx, ctx.Request, ctx.SpaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		userIDs = policy.Config.UserIDs
	}

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
				return errors.New("Identity listed in the space policy not found")
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

// Add user's identity to the list of space collaborators.
func (c *CollaboratorsController) Add(ctx *app.AddCollaboratorsContext) error {
	if !c.config.IsAuthorizationEnabled() {
		// Ignore if authZ is disabled (by default in Dev Mode)
		log.Warn(ctx, map[string]interface{}{
			"space_id": ctx.SpaceID,
		}, "Authorization is disabled. No space collaborators added")
		return ctx.OK([]byte{})
	}
	identityIDs := []*app.UpdateUserID{{ID: ctx.IdentityID}}
	err := c.updatePolicy(ctx, ctx.Request, ctx.SpaceID, identityIDs, c.policyManager.AddUserToPolicy)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.OK([]byte{})
}

// AddMany adds user's identities to the list of space collaborators.
func (c *CollaboratorsController) AddMany(ctx *app.AddManyCollaboratorsContext) error {
	if !c.config.IsAuthorizationEnabled() {
		// Ignore if authZ is disabled (by default in Dev Mode)
		log.Warn(ctx, map[string]interface{}{
			"space_id": ctx.SpaceID,
		}, "Authorization is disabled. No space collaborators added")
		return ctx.OK([]byte{})
	}
	if ctx.Payload != nil && ctx.Payload.Data != nil {
		err := c.updatePolicy(ctx, ctx.Request, ctx.SpaceID, ctx.Payload.Data, c.policyManager.AddUserToPolicy)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
	}
	return ctx.OK([]byte{})
}

// Remove user from the list of space collaborators.
func (c *CollaboratorsController) Remove(ctx *app.RemoveCollaboratorsContext) error {
	if !c.config.IsAuthorizationEnabled() {
		// Ignore if authZ is disabled (by default in Dev Mode)
		log.Warn(ctx, map[string]interface{}{
			"space_id": ctx.SpaceID,
		}, "Authorization is disabled. No space collaborators removed")
		return ctx.OK([]byte{})
	}
	// Don't remove the space owner
	err := c.checkSpaceOwner(ctx, ctx.SpaceID, ctx.IdentityID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	identityIDs := []*app.UpdateUserID{{ID: ctx.IdentityID}}
	err = c.updatePolicy(ctx, ctx.Request, ctx.SpaceID, identityIDs, c.policyManager.RemoveUserFromPolicy)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.OK([]byte{})
}

// RemoveMany removes users from the list of space collaborators.
func (c *CollaboratorsController) RemoveMany(ctx *app.RemoveManyCollaboratorsContext) error {
	if !c.config.IsAuthorizationEnabled() {
		// Ignore if authZ is disabled (by default in Dev Mode)
		log.Warn(ctx, map[string]interface{}{
			"space_id": ctx.SpaceID,
		}, "Authorization is disabled. No space collaborators removed")
		return ctx.OK([]byte{})
	}
	if ctx.Payload != nil && ctx.Payload.Data != nil {
		// Don't remove the space owner
		for _, idn := range ctx.Payload.Data {
			if idn != nil {
				err := c.checkSpaceOwner(ctx, ctx.SpaceID, idn.ID)
				if err != nil {
					return jsonapi.JSONErrorResponse(ctx, err)
				}
			}
		}
		err := c.updatePolicy(ctx, ctx.Request, ctx.SpaceID, ctx.Payload.Data, c.policyManager.RemoveUserFromPolicy)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
	}

	return ctx.OK([]byte{})
}

func (c *CollaboratorsController) checkSpaceOwner(ctx context.Context, spaceID uuid.UUID, identityID string) error {
	var ownerID string
	err := application.Transactional(c.db, func(appl application.Application) error {
		space, err := appl.Spaces().Load(ctx, spaceID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"space_id": spaceID.String(),
				"err":      err,
			}, "unable to find the space")
			return err
		}
		ownerID = space.OwnerId.String()
		return nil
	})
	if err != nil {
		return goa.ErrNotFound(err.Error())
	}
	if identityID == ownerID {
		return goa.ErrBadRequest("Space owner can't be removed from the list of the space collaborators")
	}
	return nil
}

func (c *CollaboratorsController) updatePolicy(ctx collaboratorContext, req *http.Request, spaceID uuid.UUID, identityIDs []*app.UpdateUserID, update func(policy *auth.KeycloakPolicy, identityID string) bool) error {
	// Authorize current user
	authorized, err := authz.Authorize(ctx, spaceID.String())
	if err != nil {
		return goa.ErrUnauthorized(err.Error())
	}
	if !authorized {
		return goa.ErrUnauthorized("User not among space collaborators")
	}

	// Update policy
	policy, pat, err := c.getPolicy(ctx, req, spaceID)
	if err != nil {
		return err
	}
	updated := false
	for _, identityIDData := range identityIDs {
		if identityIDData != nil {
			identityID := identityIDData.ID
			identityUUID, err := uuid.FromString(identityID)
			if err != nil {
				log.Error(ctx, map[string]interface{}{
					"identity_id": identityID,
				}, "unable to convert the identity ID to uuid v4")
				return goa.ErrBadRequest(err.Error())
			}
			err = application.Transactional(c.db, func(appl application.Application) error {
				identities, err := appl.Identities().Query(account.IdentityFilterByID(identityUUID), account.IdentityWithUser())
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"identity_id": identityID,
						"err":         err,
					}, "unable to find the identity")
					return err
				}
				if len(identities) == 0 {
					log.Error(ctx, map[string]interface{}{
						"identity_id": identityID,
					}, "unable to find the identity")
					return errors.New("Identity not found")
				}
				return nil
			})
			if err != nil {
				return goa.ErrNotFound(err.Error())
			}
			updated = update(policy, identityID) || updated
		}
	}
	if !updated {
		// Nothing changed. No need to update
		return nil
	}

	err = c.policyManager.UpdatePolicy(ctx, req, *policy, *pat)
	if err != nil {
		return goa.ErrInternal(err.Error())
	}

	// We need to update the resource to triger RPT token refreshing when users try to access this space
	err = application.Transactional(c.db, func(appl application.Application) error {
		resource, err := appl.SpaceResources().LoadBySpace(ctx, &spaceID)
		_, err = appl.SpaceResources().Save(ctx, resource)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"resource":   resource,
				"space_uuid": spaceID.String(),
				"err":        err,
			}, "unable to update the space resource")
			return err
		}
		return nil
	})
	if err != nil {
		return goa.ErrInternal(err.Error())
	}

	return nil
}

func (c *CollaboratorsController) getPolicy(ctx collaboratorContext, req *http.Request, spaceID uuid.UUID) (*auth.KeycloakPolicy, *string, error) {
	var policyID string
	err := application.Transactional(c.db, func(appl application.Application) error {
		// Load associated space resource
		resource, err := appl.SpaceResources().LoadBySpace(ctx, &spaceID)
		if err != nil {
			return err
		}
		policyID = resource.PolicyID
		return nil
	})

	if err != nil {
		return nil, nil, goa.ErrNotFound(err.Error())
	}
	policy, pat, err := c.policyManager.GetPolicy(ctx, req, policyID)
	if err != nil {
		return nil, nil, goa.ErrInternal(err.Error())
	}
	return policy, pat, nil
}
