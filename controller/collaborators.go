package controller

import (
	"context"
	"errors"
	"strings"

	"fmt"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/auth"
	errs "github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/space"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
)

// CollaboratorsController implements the collaborators resource.
type CollaboratorsController struct {
	*goa.Controller
	db            application.DB
	config        collaboratorsConfiguration
	policyManager auth.AuthzPolicyManager
}

type collaboratorsConfiguration interface {
	GetKeycloakEndpointEntitlement(*goa.RequestData) (string, error)
}

type collaboratorContext interface {
	context.Context
	jsonapi.InternalServerError
}

// NewCollaboratorsController creates a collaborators controller.
func NewCollaboratorsController(service *goa.Service, db application.DB, config collaboratorsConfiguration, policyManager auth.AuthzPolicyManager) *CollaboratorsController {
	return &CollaboratorsController{Controller: service.NewController("CollaboratorsController"), db: db, config: config, policyManager: policyManager}
}

// List collaborators for the given space ID.
func (c *CollaboratorsController) List(ctx *app.ListCollaboratorsContext) error {
	policy, _, err := c.getPolicy(ctx, ctx.RequestData, ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	userIDs := policy.Config.UserIDs
	//UsersIDs format : "[\"<ID>\",\"<ID>\"]"
	s := strings.Split(userIDs, ",")
	count := len(s)

	offset, limit := computePagingLimts(ctx.PageOffset, ctx.PageLimit)
	if offset > len(s) {
		offset = len(s)
	}
	if offset+limit > len(s) {
		limit = len(s)
	}
	page := s[offset : offset+limit]

	data := make([]*app.IdentityData, len(s))
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
			appIdentity := ConvertUser(ctx.RequestData, identities[0], &identities[0].User)
			data[i] = appIdentity.Data
			return nil
		})
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}
	}

	response := app.UserList{
		Links: &app.PagingLinks{},
		Meta:  &app.UserListMeta{TotalCount: count},
		Data:  data,
	}
	setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(page), offset, limit, count)
	return ctx.OK(&response)
}

// Add user's identity to the list of space collaborators.
func (c *CollaboratorsController) Add(ctx *app.AddCollaboratorsContext) error {
	identityIDs := []*app.UpdateUserID{{ID: ctx.IdentityID}}
	err := c.updatePolicy(ctx, ctx.RequestData, ctx.ID, identityIDs, c.policyManager.AddUserToPolicy)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.OK([]byte{})
}

// AddMany adds user's identities to the list of space collaborators.
func (c *CollaboratorsController) AddMany(ctx *app.AddManyCollaboratorsContext) error {
	if ctx.Payload != nil && ctx.Payload.Data != nil {
		err := c.updatePolicy(ctx, ctx.RequestData, ctx.ID, ctx.Payload.Data, c.policyManager.AddUserToPolicy)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
	}
	return ctx.OK([]byte{})
}

// Remove user from the list of space collaborators.
func (c *CollaboratorsController) Remove(ctx *app.RemoveCollaboratorsContext) error {
	// Don't remove the space owner
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": ctx.ID,
		}, "unable to convert the space ID to uuid v4")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest(err.Error()))
	}
	err = c.checkSpaceOwner(ctx, spaceID, ctx.IdentityID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	identityIDs := []*app.UpdateUserID{{ID: ctx.IdentityID}}
	err = c.updatePolicy(ctx, ctx.RequestData, ctx.ID, identityIDs, c.policyManager.RemoveUserFromPolicy)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.OK([]byte{})
}

// RemoveMany removes users from the list of space collaborators.
func (c *CollaboratorsController) RemoveMany(ctx *app.RemoveManyCollaboratorsContext) error {
	if ctx.Payload != nil && ctx.Payload.Data != nil {
		// Don't remove the space owner
		spaceID, err := uuid.FromString(ctx.ID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"space_id": ctx.ID,
			}, "unable to convert the space ID to uuid v4")
			return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest(err.Error()))
		}
		for _, idn := range ctx.Payload.Data {
			if idn != nil {
				err := c.checkSpaceOwner(ctx, spaceID, idn.ID)
				if err != nil {
					return jsonapi.JSONErrorResponse(ctx, err)
				}
			}
		}
		err = c.updatePolicy(ctx, ctx.RequestData, ctx.ID, ctx.Payload.Data, c.policyManager.RemoveUserFromPolicy)
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

func (c *CollaboratorsController) updatePolicy(ctx collaboratorContext, req *goa.RequestData, spaceID string, identityIDs []*app.UpdateUserID, update func(policy *auth.KeycloakPolicy, identityID string) bool) error {
	// Authorize current user
	authorized, err := c.policyManager.VerifyUser(ctx, req, spaceID)
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
			var identity *account.Identity
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
				identity = identities[0]
				return nil
			})
			if err != nil {
				return goa.ErrNotFound(err.Error())
			}
			updated = updated || update(policy, identityID)
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

	// TODO	We will need to update the resource when implementing http cache
	// _, err = appl.SpaceResources().Save(ctx, resource)

	return nil
}

func (c *CollaboratorsController) getPolicy(ctx collaboratorContext, req *goa.RequestData, spaceID string) (*auth.KeycloakPolicy, *string, error) {
	spaceUUID, err := uuid.FromString(spaceID)
	if err != nil {
		return nil, nil, goa.ErrBadRequest(err.Error())
	}
	var policyID string
	var spaceForMissingResource *space.Space
	var policy *auth.KeycloakPolicy
	var pat *string
	err = application.Transactional(c.db, func(appl application.Application) error {
		// Load associated space resource
		resource, err := appl.SpaceResources().LoadBySpace(ctx, &spaceUUID)
		if err != nil {
			if _, ok := err.(errs.NotFoundError); !ok {
				return err
			}
			// No space resource found. Check if the space exists.
			space, err := appl.Spaces().Load(ctx, spaceUUID)
			if err != nil {
				return err
			}
			// Space found but there is no space resource assosiated with this Space. Can happen for old spaces.
			spaceForMissingResource = space
			return nil
		}
		policyID = resource.PolicyID
		return nil
	})

	if err != nil {
		return nil, nil, goa.ErrNotFound(err.Error())
	}
	if spaceForMissingResource != nil {
		policy, pat, err = c.createPolicy(ctx, req, *spaceForMissingResource)
	} else {
		policy, pat, err = c.policyManager.GetPolicy(ctx, req, policyID)
	}
	if err != nil {
		return nil, nil, goa.ErrInternal(err.Error())
	}
	return policy, pat, nil
}

// Creates Policy if missing. Can happen for old spaces.
func (c *CollaboratorsController) createPolicy(ctx collaboratorContext, req *goa.RequestData, spc space.Space) (*auth.KeycloakPolicy, *string, error) {
	resource, err := c.policyManager.CreateResource(ctx, req, spc.ID.String(), spaceResourceType, &spc.Name, &scopes, spc.OwnerId.String(), fmt.Sprintf("%s-%s", spc.Name, uuid.NewV4().String()))
	if err != nil {
		return nil, nil, err
	}
	spaceResource := &space.Resource{
		ResourceID:   resource.ResourceID,
		PolicyID:     resource.PolicyID,
		PermissionID: resource.PermissionID,
		SpaceID:      spc.ID,
	}
	err = application.Transactional(c.db, func(appl application.Application) error {
		// Create space resource which will represent the keyclok resource associated with this space
		_, err = appl.SpaceResources().Create(ctx, spaceResource)
		return err
	})
	if err != nil {
		// Clean up KC resource if transaction failed
		resErr := c.policyManager.DeleteResource(ctx, req, *resource)
		if resErr != nil {
			return nil, nil, fmt.Errorf("%s %s", err.Error(), resErr.Error())
		}
		return nil, nil, err
	}
	return c.policyManager.GetPolicy(ctx, req, resource.PolicyID)
}
