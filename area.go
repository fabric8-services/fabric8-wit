package main

import (
	"fmt"
	"strings"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/rest"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// AreaController implements the area resource.
type AreaController struct {
	*goa.Controller
	db application.DB
}

const pathSep = "."

// NewAreaController creates a area controller.
func NewAreaController(service *goa.Service, db application.DB) *AreaController {
	return &AreaController{Controller: service.NewController("AreaController"), db: db}
}

// ShowChild runs the show-child action
func (c *AreaController) ShowChild(ctx *app.ShowChildAreaContext) error {
	id, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		parentArea, err := appl.Areas().Load(ctx, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		c, err := appl.Areas().ListChildren(ctx, parentArea)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.AreaList{}
		res.Data = ConvertAreas(
			ctx.RequestData,
			c)

		return ctx.OK(res)
	})
}

// CreateChild runs the create-child action.
func (c *AreaController) CreateChild(ctx *app.CreateChildAreaContext) error {

	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	parentID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		parent, err := appl.Areas().Load(ctx, parentID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}

		reqArea := ctx.Payload.Data
		if reqArea.Attributes.Name == nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil"))
		}

		childPath := ConvertToLtreeFormat(parentID.String())
		if parent.Path != "" {
			childPath = parent.Path + pathSep + childPath
		}
		newArea := area.Area{
			SpaceID: parent.SpaceID,
			Path:    childPath,
			Name:    *reqArea.Attributes.Name,
		}

		err = appl.Areas().Create(ctx, &newArea)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.AreaSingle{
			Data: ConvertArea(ctx.RequestData, &newArea),
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

		// resolve path names.

		parentUuidStrings := strings.Split(ConvertFromLtreeFormat(c.Path), ".")
		parentUuids := convertToUuid(parentUuidStrings)
		parentAreas, err := appl.Areas().LoadMultiple(ctx, parentUuids)
		path_resolved := ""
		for _, a := range parentAreas {
			if path_resolved == "" {
				path_resolved = a.Name
				continue
			}
			path_resolved = path_resolved + "." + a.Name
		}
		res.Data.Attributes.PathResolved = &path_resolved
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
func ConvertArea(request *goa.RequestData, ar *area.Area, additional ...AreaConvertFunc) *app.Area {
	areaType := area.APIStringTypeAreas
	spaceType := "spaces"

	spaceID := ar.SpaceID.String()

	selfURL := rest.AbsoluteURL(request, app.AreaHref(ar.ID))
	childURL := rest.AbsoluteURL(request, app.AreaHref(ar.ID)+"/children")
	spaceSelfURL := rest.AbsoluteURL(request, app.SpaceHref(spaceID))
	pathToTopMostParent := ConvertFromLtreeFormat(ar.Path) // uuid1.uuid2.uuid3s

	i := &app.Area{
		Type: areaType,
		ID:   &ar.ID,
		Attributes: &app.AreaAttributes{
			Name:      &ar.Name,
			CreatedAt: &ar.CreatedAt,
			Version:   &ar.Version,
			Path:      &pathToTopMostParent,
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
			Children: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Self: &childURL,
				},
			},
		},
		Links: &app.GenericLinks{
			Self: &selfURL,
		},
	}

	// Now check the path, if the path is empty, then this is the topmost area
	// in a specific space.
	if ar.Path != "" {

		allParents := strings.Split(ConvertFromLtreeFormat(ar.Path), ".")
		parentID := allParents[len(allParents)-1]

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
		add(request, ar, i)
	}
	return i
}

// ConvertAreaSimple converts a simple area ID into a Generic Reletionship
func ConvertAreaSimple(request *goa.RequestData, id interface{}) *app.GenericData {
	t := area.APIStringTypeAreas
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

// ConvertToLtreeFormat converts data in UUID format to ltree format.
func ConvertToLtreeFormat(uuid string) string {
	//Ltree allows only "_" as a special character.
	return strings.Replace(uuid, "-", "_", -1)
}

// ConvertFromLtreeFormat converts data to UUID format from ltree format.
func ConvertFromLtreeFormat(uuid string) string {
	// Ltree allows only "_" as a special character.
	return strings.Replace(uuid, "_", "-", -1)
}

func convertToUuid(uuidStrings []string) []uuid.UUID {
	var uUIDs []uuid.UUID

	for i := 0; i < len(uuidStrings); i++ {
		uuidString, _ := uuid.FromString(uuidStrings[i])
		uUIDs = append(uUIDs, uuidString)
	}
	return uUIDs
}
