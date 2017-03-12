package controller

import (
	"context"
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/space"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	satoriuuid "github.com/satori/go.uuid"
)

// following constants define keys to be used in response
const (
	DefaultIterationKey = "defaultIteration"
	BacklogURLKey       = "backlog"
)

// SpaceController implements the space resource.
type SpaceController struct {
	*goa.Controller
	db application.DB
}

// NewSpaceController creates a space controller.
func NewSpaceController(service *goa.Service, db application.DB) *SpaceController {
	return &SpaceController{Controller: service.NewController("SpaceController"), db: db}
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

		space, err := appl.Spaces().Create(ctx, &newSpace)
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
			ID:      satoriuuid.NewV4(),
			SpaceID: space.ID,
			Name:    space.Name,
		}
		err = appl.Areas().Create(ctx, &newArea)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "failed to create area: %s", space.Name))
		}

		// Similar to above, we create a default iteration for this new space
		newIteration := iteration.Iteration{
			ID:      satoriuuid.NewV4(),
			SpaceID: space.ID,
			Name:    space.Name,
		}
		err = appl.Iterations().Create(ctx, &newIteration)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "failed to create iteration for space: %s", space.Name))
		}
		itrRepo := appl.Iterations()
		addBacklogLink := updateSpaceWithLinkToBacklogWI(ctx, itrRepo)
		res := &app.SpaceSingle{
			Data: ConvertSpace(ctx.RequestData, space, addBacklogLink),
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
	id, err := satoriuuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
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
		itrRepo := appl.Iterations()
		addBacklogLink := updateSpaceWithLinkToBacklogWI(ctx, itrRepo)
		response := app.SpaceList{
			Links: &app.PagingLinks{},
			Meta:  &app.SpaceListMeta{TotalCount: count},
			Data:  ConvertSpaces(ctx.RequestData, spaces, addBacklogLink),
		}
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(spaces), offset, limit, count)

		return ctx.OK(&response)
	})

}

// Show runs the show action.
func (c *SpaceController) Show(ctx *app.ShowSpaceContext) error {
	id, err := satoriuuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		s, err := appl.Spaces().Load(ctx.Context, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		itrRepo := appl.Iterations()
		addBacklogLink := updateSpaceWithLinkToBacklogWI(ctx, itrRepo)
		resp := app.SpaceSingle{
			Data: ConvertSpace(ctx.RequestData, s, addBacklogLink),
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
	id, err := satoriuuid.FromString(ctx.ID)
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

		if !satoriuuid.Equal(*currentUser, s.OwnerId) {
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
func ConvertSpace(request *goa.RequestData, sp *space.Space, additional ...SpaceConvertFunc) *app.Space {
	selfURL := rest.AbsoluteURL(request, app.SpaceHref(sp.ID))
	relatedIterationList := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/iterations", sp.ID.String()))
	relatedAreaList := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/areas", sp.ID.String()))
	s := &app.Space{
		ID:   &sp.ID,
		Type: "spaces",
		Attributes: &app.SpaceAttributes{
			Name:        &sp.Name,
			Description: &sp.Description,
			CreatedAt:   &sp.CreatedAt,
			UpdatedAt:   &sp.UpdatedAt,
			Version:     &sp.Version,
		},
		Links: &app.GenericLinks{
			Self: &selfURL,
		},
		Relationships: &app.SpaceRelationships{
			OwnedBy: &app.SpaceOwnedBy{
				Data: &app.IdentityRelationData{
					Type: "identities",
					ID:   &sp.OwnerId,
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
	for _, add := range additional {
		add(request, sp, s)
	}
	return s
}

func updateSpaceWithLinkToBacklogWI(ctx context.Context, itrRepo iteration.Repository) SpaceConvertFunc {
	return func(request *goa.RequestData, sp *space.Space, appSpace *app.Space) {
		defaultItr, err := itrRepo.LoadDefault(ctx, *sp)
		if err == nil {
			// add default iteration ID to app.Space instace
			if appSpace.Relationships == nil {
				appSpace.Relationships = &app.SpaceRelationships{}
			}
			if appSpace.Relationships.Iterations == nil {
				appSpace.Relationships.Iterations = &app.RelationGeneric{}
			}
			if appSpace.Relationships.Iterations.Meta == nil {
				appSpace.Relationships.Iterations.Meta = map[string]interface{}{}
			}
			appSpace.Relationships.Iterations.Meta[DefaultIterationKey] = defaultItr.ID.String()

			// add BacklogURL to app.Space instace
			backlogWorkItemsLink := rest.AbsoluteURL(request, app.WorkitemHref("?filter[iteration]="+defaultItr.ID.String()))

			if appSpace.Relationships.Workitems == nil {
				appSpace.Relationships.Workitems = &app.RelationGeneric{}
			}
			if appSpace.Relationships.Workitems.Meta == nil {
				appSpace.Relationships.Workitems.Meta = map[string]interface{}{}
			}
			appSpace.Relationships.Workitems.Meta[BacklogURLKey] = backlogWorkItemsLink
		}
	}
}
