package controllers

import (
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/space"
	"github.com/goadesign/goa"
	satoriuuid "github.com/satori/go.uuid"
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
	_, err := login.ContextIdentity(ctx)
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
			Name: *reqSpace.Attributes.Name,
		}
		if reqSpace.Attributes.Description != nil {
			newSpace.Description = *reqSpace.Attributes.Description
		}

		space, err := appl.Spaces().Create(ctx, &newSpace)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		res := &app.SpaceSingle{
			Data: ConvertSpace(ctx.RequestData, space),
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
	id, err := satoriuuid.FromString(ctx.ID)
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
	_, err := login.ContextIdentity(ctx)
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
