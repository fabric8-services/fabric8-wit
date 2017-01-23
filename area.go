package main

import (
	"fmt"
	"strings"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/rest"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// AreaController implements the area resource.
type AreaController struct {
	*goa.Controller
	db application.DB
}

// NewAreaController creates a area controller.
func NewAreaController(service *goa.Service, db application.DB) *AreaController {
	return &AreaController{Controller: service.NewController("AreaController"), db: db}
}

// CreateChild runs the create-child action.
func (c *AreaController) CreateChild(ctx *app.CreateChildAreaContext) error {
	/*
		_, err := login.ContextIdentity(ctx)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
		}*/
	parentID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		parent, err := appl.Areas().Load(ctx, parentID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}

		reqIter := ctx.Payload.Data
		if reqIter.Attributes.Name == nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil"))
		}

		newItr := area.Area{
			SpaceID: parent.SpaceID,

			// the ltree data type doesn't support the "-" character.
			// hence everything is being saved as "_".
			// TODO: Move the replacement to a different method?
			// TODO: Get all parents of the parent if present and create a "." delimited path
			Path: strings.Replace(parentID.String(), "-", "_", -1),

			Name: *reqIter.Attributes.Name,
		}

		err = appl.Areas().Create(ctx, &newItr)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.AreaSingle{
			Data: ConvertArea(ctx.RequestData, &newItr),
		}
		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.RequestData, app.AreaHref(res.Data.ID)))
		return ctx.Created(res)
	})
}

// Show runs the show action.
func (c *AreaController) Show(ctx *app.ShowAreaContext) error {
	id, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		c, err := appl.Areas().Load(ctx, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.AreaSingle{}
		res.Data = ConvertArea(
			ctx.RequestData,
			c)

		return ctx.OK(res)
	})
}

// AreaConvertFunc is a open ended function to add additional links/data/relations to a area during
// convertion from internal to API
type AreaConvertFunc func(*goa.RequestData, *area.Area, *app.Area)

// ConvertAreas converts between internal and external REST representation
func ConvertAreas(request *goa.RequestData, areas []*area.Area, additional ...AreaConvertFunc) []*app.Area {
	var is = []*app.Area{}
	for _, i := range areas {
		is = append(is, ConvertArea(request, i, additional...))
	}
	return is
}

// ConvertArea converts between internal and external REST representation
func ConvertArea(request *goa.RequestData, area *area.Area, additional ...AreaConvertFunc) *app.Area {
	areaType := "areas"
	spaceType := "spaces"

	spaceID := area.SpaceID.String()

	selfURL := rest.AbsoluteURL(request, app.AreaHref(area.ID))
	spaceSelfURL := rest.AbsoluteURL(request, "/api/spaces/"+spaceID)

	i := &app.Area{
		Type: areaType,
		ID:   &area.ID,
		Attributes: &app.AreaAttributes{
			Name: &area.Name,
		},
		Relationships: &app.AreaRelations{
			Space: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: &spaceType,
					ID:   &spaceID,
				},
				Links: &app.GenericLinks{
					Self: &spaceSelfURL,
				},
			},
		},
		Links: &app.GenericLinks{
			Self: &selfURL,
		},
	}

	// Now check the path, if the path is empty, then this is the topmost area
	// in a specific space.
	if area.Path != "" {

		// Parent ID of the immediate parent.
		// the ltree data type doesn't support the "-" character.
		// hence everything is being saved as "_". After retrieving the data
		// convert this back to "-".
		// TODO: Move the replacement to a different method?

		parentID := strings.Replace(area.Path, "_", "-", -1)

		// Only the immediate parent's URL.
		parentSelfURL := rest.AbsoluteURL(request, app.AreaHref(parentID))

		i.Relationships.Parent = &app.RelationGeneric{
			Data: &app.GenericData{
				Type: &areaType,
				ID:   &parentID,
			},
			Links: &app.GenericLinks{
				Self: &parentSelfURL,
			},
		}
	}
	for _, add := range additional {
		add(request, area, i)
	}
	return i
}

// ConvertAreaSimple converts a simple area ID into a Generic Reletionship
func ConvertAreaSimple(request *goa.RequestData, id interface{}) *app.GenericData {
	t := "areas"
	i := fmt.Sprint(id)
	return &app.GenericData{
		Type:  &t,
		ID:    &i,
		Links: createAreaLinks(request, id),
	}
}

func createAreaLinks(request *goa.RequestData, id interface{}) *app.GenericLinks {
	selfURL := rest.AbsoluteURL(request, app.AreaHref(id))
	return &app.GenericLinks{
		Self: &selfURL,
	}
}
