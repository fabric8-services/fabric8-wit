package controller

import (
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/auth"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/space"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const (
	spaceResourceType = "space"
)

var scopes = []string{"read:space", "admin:space"}

type spaceConfiguration interface {
	GetKeycloakEndpointAuthzResourceset(*goa.RequestData) (string, error)
	GetKeycloakEndpointToken(*goa.RequestData) (string, error)
	GetKeycloakEndpointClients(*goa.RequestData) (string, error)
	GetKeycloakEndpointAdmin(*goa.RequestData) (string, error)
	GetKeycloakClientID() string
	GetKeycloakSecret() string
}

// SpaceController implements the space resource.
type SpaceController struct {
	*goa.Controller
	db              application.DB
	configuration   spaceConfiguration
	resourceManager auth.AuthzResourceManager
}

// NewSpaceController creates a space controller.
func NewSpaceController(service *goa.Service, db application.DB, configuration spaceConfiguration, resourceManager auth.AuthzResourceManager) *SpaceController {
	return &SpaceController{Controller: service.NewController("SpaceController"), db: db, configuration: configuration, resourceManager: resourceManager}
}

// Create runs the create action.
func (c *SpaceController) Create(ctx *app.CreateSpaceContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}

	err = validateCreateSpace(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		reqSpace := ctx.Payload.Data

		newSpace := space.Space{
			Name:    *reqSpace.Attributes.Name,
			OwnerId: *currentUser,
		}
		if reqSpace.Attributes.Description != nil {
			newSpace.Description = *reqSpace.Attributes.Description
		}

		rSpace, err := appl.Spaces().Create(ctx, &newSpace)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		/*
			Should we create the new area
			- over the wire(service) something like app.NewCreateSpaceAreasContext(..), OR
			- as part of a db transaction ?

			The argument 'for' creating it at a transaction level is :
			You absolutely need both space creation + area creation
			to happen in a single transaction as per requirements.
		*/

		newArea := area.Area{
			ID:      uuid.NewV4(),
			SpaceID: rSpace.ID,
			Name:    rSpace.Name,
		}
		err = appl.Areas().Create(ctx, &newArea)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "failed to create area: %s", rSpace.Name))
		}

		res := &app.SpaceSingle{
			Data: ConvertSpace(ctx.RequestData, rSpace),
		}

		// Create keycloak resource for this space
		resource, err := c.resourceManager.CreateResource(ctx, ctx.RequestData, rSpace.ID.String(), spaceResourceType, &rSpace.Name, &scopes, rSpace.OwnerId.String(), rSpace.Name+"-"+uuid.NewV4().String())
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		spaceResource := &space.Resource{
			ResourceID:   resource.ResourceID,
			PolicyID:     resource.PolicyID,
			PermissionID: resource.PermissionID,
			SpaceID:      rSpace.ID,
		}

		// Create space resource which will represent the keyclok resource associated with this space
		_, err = appl.SpaceResources().Create(ctx, spaceResource)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.RequestData, app.SpaceHref(res.Data.ID)))
		return ctx.Created(res)
	})
}

// Delete runs the delete action.
func (c *SpaceController) Delete(ctx *app.DeleteSpaceContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	id, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		// Delete associated space resource
		resource, err := appl.SpaceResources().LoadBySpace(ctx, &id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		c.resourceManager.DeleteResource(ctx, ctx.RequestData, auth.Resource{ResourceID: resource.PermissionID, PermissionID: resource.PermissionID, PolicyID: resource.PolicyID})
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		appl.SpaceResources().Delete(ctx, resource.ID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		err = appl.Spaces().Delete(ctx.Context, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		return ctx.OK([]byte{})
	})
}

// List runs the list action.
func (c *SpaceController) List(ctx *app.ListSpaceContext) error {
	offset, limit := computePagingLimts(ctx.PageOffset, ctx.PageLimit)

	return application.Transactional(c.db, func(appl application.Application) error {
		spaces, c, err := appl.Spaces().List(ctx.Context, &offset, &limit)
		count := int(c)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		response := app.SpaceList{
			Links: &app.PagingLinks{},
			Meta:  &app.SpaceListMeta{TotalCount: count},
			Data:  ConvertSpaces(ctx.RequestData, spaces),
		}
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(spaces), offset, limit, count)

		return ctx.OK(&response)
	})

}

// Show runs the show action.
func (c *SpaceController) Show(ctx *app.ShowSpaceContext) error {
	id, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		s, err := appl.Spaces().Load(ctx.Context, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		resp := app.SpaceSingle{
			Data: ConvertSpace(ctx.RequestData, s),
		}

		return ctx.OK(&resp)
	})
}

// Update runs the update action.
func (c *SpaceController) Update(ctx *app.UpdateSpaceContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	id, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	err = validateUpdateSpace(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		s, err := appl.Spaces().Load(ctx.Context, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		if !uuid.Equal(*currentUser, s.OwnerId) {
			log.Error(ctx, map[string]interface{}{"currentUser": *currentUser, "owner": s.OwnerId}, "Current user is not owner")
			return jsonapi.JSONErrorResponse(ctx, goa.NewErrorClass("forbidden", 403)("User is not the space owner"))
		}

		s.Version = *ctx.Payload.Data.Attributes.Version
		if ctx.Payload.Data.Attributes.Name != nil {
			s.Name = *ctx.Payload.Data.Attributes.Name
		}
		if ctx.Payload.Data.Attributes.Description != nil {
			s.Description = *ctx.Payload.Data.Attributes.Description
		}

		s, err = appl.Spaces().Save(ctx.Context, s)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		response := app.SpaceSingle{
			Data: ConvertSpace(ctx.RequestData, s),
		}

		return ctx.OK(&response)
	})
}

func validateCreateSpace(ctx *app.CreateSpaceContext) error {
	if ctx.Payload.Data == nil {
		return errors.NewBadParameterError("data", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes == nil {
		return errors.NewBadParameterError("data.attributes", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Name == nil {
		return errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil")
	}
	return nil
}

func validateUpdateSpace(ctx *app.UpdateSpaceContext) error {
	if ctx.Payload.Data == nil {
		return errors.NewBadParameterError("data", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes == nil {
		return errors.NewBadParameterError("data.attributes", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Name == nil {
		return errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Version == nil {
		return errors.NewBadParameterError("data.attributes.version", nil).Expected("not nil")
	}
	return nil
}

// SpaceConvertFunc is a open ended function to add additional links/data/relations to a Space during
// conversion from internal to API
type SpaceConvertFunc func(*goa.RequestData, *space.Space, *app.Space)

// ConvertSpaces converts between internal and external REST representation
func ConvertSpaces(request *goa.RequestData, spaces []*space.Space, additional ...SpaceConvertFunc) []*app.Space {
	var ps = []*app.Space{}
	for _, p := range spaces {
		ps = append(ps, ConvertSpace(request, p, additional...))
	}
	return ps
}

// ConvertSpace converts between internal and external REST representation
func ConvertSpace(request *goa.RequestData, p *space.Space, additional ...SpaceConvertFunc) *app.Space {
	selfURL := rest.AbsoluteURL(request, app.SpaceHref(p.ID))
	relatedIterationList := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/iterations", p.ID.String()))
	relatedAreaList := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/areas", p.ID.String()))
	return &app.Space{
		ID:   &p.ID,
		Type: "spaces",
		Attributes: &app.SpaceAttributes{
			Name:        &p.Name,
			Description: &p.Description,
			CreatedAt:   &p.CreatedAt,
			UpdatedAt:   &p.UpdatedAt,
			Version:     &p.Version,
		},
		Links: &app.GenericLinks{
			Self: &selfURL,
		},
		Relationships: &app.SpaceRelationships{
			OwnedBy: &app.SpaceOwnedBy{
				Data: &app.IdentityRelationData{
					Type: "identities",
					ID:   &p.OwnerId,
				},
			},
			Iterations: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedIterationList,
				},
			},
			Areas: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedAreaList,
				},
			},
		},
	}
}
