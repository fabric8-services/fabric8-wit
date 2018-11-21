package controller

import (
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/ptr"

	"context"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/path"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// AreaController implements the area resource.
type AreaController struct {
	*goa.Controller
	db     application.DB
	config AreaControllerConfiguration
}

// AreaControllerConfiguration the configuration for the AreaController
type AreaControllerConfiguration interface {
	GetCacheControlAreas() string
	GetCacheControlArea() string
}

// NewAreaController creates a area controller.
func NewAreaController(service *goa.Service, db application.DB, config AreaControllerConfiguration) *AreaController {
	return &AreaController{
		Controller: service.NewController("AreaController"),
		db:         db,
		config:     config}
}

// ShowChildren runs the show-children action
func (c *AreaController) ShowChildren(ctx *app.ShowChildrenAreaContext) error {
	id, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}
	var children []area.Area
	err = application.Transactional(c.db, func(appl application.Application) error {
		parentArea, err := appl.Areas().Load(ctx, id)
		if err != nil {
			return err
		}
		children, err = appl.Areas().ListChildren(ctx, parentArea)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	return ctx.ConditionalEntities(children, c.config.GetCacheControlAreas, func() error {
		res := &app.AreaList{}
		res.Data = ConvertAreas(c.db, ctx.Request, children, addResolvedPath)
		return ctx.OK(res)
	})
}

// CreateChild runs the create-child action.
func (c *AreaController) CreateChild(ctx *app.CreateChildAreaContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	parentID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}
	var a *area.Area
	err = application.Transactional(c.db, func(appl application.Application) error {
		parent, err := appl.Areas().Load(ctx, parentID)
		if err != nil {
			return err
		}
		s, err := appl.Spaces().Load(ctx, parent.SpaceID)
		if err != nil {
			return err
		}
		if !uuid.Equal(*currentUser, s.OwnerID) {
			log.Warn(ctx, map[string]interface{}{
				"space_id":     s.ID,
				"space_owner":  s.OwnerID,
				"current_user": *currentUser,
			}, "user is not the space owner")
			return errors.NewForbiddenError("user is not the space owner")
		}

		reqArea := ctx.Payload.Data
		if reqArea.Attributes.Name == nil {
			return errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil")
		}

		a = &area.Area{
			SpaceID: parent.SpaceID,
			Name:    *reqArea.Attributes.Name,
		}
		a.MakeChildOf(*parent)
		return appl.Areas().Create(ctx, a)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	result := &app.AreaSingle{
		Data: ConvertArea(c.db, ctx.Request, *a, addResolvedPath),
	}
	ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.Request, app.AreaHref(result.Data.ID)))
	return ctx.Created(result)
}

// Show runs the show action.
func (c *AreaController) Show(ctx *app.ShowAreaContext) error {
	id, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}
	var a *area.Area
	err = application.Transactional(c.db, func(appl application.Application) error {
		a, err = appl.Areas().Load(ctx, id)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.ConditionalRequest(*a, c.config.GetCacheControlArea, func() error {
		res := &app.AreaSingle{}
		res.Data = ConvertArea(c.db, ctx.Request, *a, addResolvedPath)
		return ctx.OK(res)
	})
}

// addResolvedPath resolves the path in the form of /area1/area2/area3
func addResolvedPath(db application.DB, req *http.Request, mArea *area.Area, sArea *app.Area) error {
	pathResolved, error := getResolvePath(db, mArea)
	sArea.Attributes.ParentPathResolved = pathResolved
	return error
}

func getResolvePath(db application.DB, a *area.Area) (*string, error) {
	parentUuids := a.Path.ParentPath()
	var parentAreas []area.Area
	err := application.Transactional(db, func(appl application.Application) error {
		var err error
		parentAreas, err = appl.Areas().LoadMultiple(context.Background(), parentUuids)
		return err
	})
	if err != nil {
		return nil, err
	}
	pathResolved := ""
	for _, a := range parentUuids {
		area := getAreaByID(a, parentAreas)
		if area == nil {
			continue
		}
		pathResolved = pathResolved + path.SepInService + area.Name
	}

	// Add the leading "/" in the "area1/area2/area3" styled path
	if pathResolved == "" {
		pathResolved = "/"
	}
	return &pathResolved, nil
}

func getAreaByID(id uuid.UUID, areas []area.Area) *area.Area {
	for _, a := range areas {
		if a.ID == id {
			return &a
		}
	}
	return nil
}

// AreaConvertFunc is a open ended function to add additional links/data/relations to a area during
// convertion from internal to API
type AreaConvertFunc func(application.DB, *http.Request, *area.Area, *app.Area) error

// ConvertAreas converts between internal and external REST representation
func ConvertAreas(db application.DB, request *http.Request, areas []area.Area, additional ...AreaConvertFunc) []*app.Area {
	var is = []*app.Area{}
	for _, i := range areas {
		is = append(is, ConvertArea(db, request, i, additional...))
	}
	return is
}

// ConvertArea converts between internal and external REST representation
func ConvertArea(db application.DB, request *http.Request, ar area.Area, options ...AreaConvertFunc) *app.Area {
	relatedURL := rest.AbsoluteURL(request, app.AreaHref(ar.ID))
	childURL := rest.AbsoluteURL(request, app.AreaHref(ar.ID)+"/children")
	spaceRelatedURL := rest.AbsoluteURL(request, app.SpaceHref(ar.SpaceID))
	i := &app.Area{
		Type: area.APIStringTypeAreas,
		ID:   &ar.ID,
		Attributes: &app.AreaAttributes{
			Name:       &ar.Name,
			CreatedAt:  &ar.CreatedAt,
			UpdatedAt:  &ar.UpdatedAt,
			Version:    &ar.Version,
			ParentPath: ptr.String(ar.Path.ParentPath().String()),
			Number:     &ar.Number,
		},
		Relationships: &app.AreaRelations{
			Space: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: &space.SpaceType,
					ID:   ptr.String(ar.SpaceID.String()),
				},
				Links: &app.GenericLinks{
					Self:    &spaceRelatedURL,
					Related: &spaceRelatedURL,
				},
			},
			Children: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Self:    &childURL,
					Related: &childURL,
				},
			},
		},
		Links: &app.GenericLinks{
			Self:    &relatedURL,
			Related: &relatedURL,
		},
	}

	// Now check the path, if the path is empty, then this is the topmost area
	// in a specific space.
	if !ar.Path.ParentPath().IsEmpty() {
		parent := ar.Path.ParentID().String()
		i.Relationships.Parent = &app.RelationGeneric{
			Data: &app.GenericData{
				Type: ptr.String(area.APIStringTypeAreas),
				ID:   &parent,
			},
			Links: &app.GenericLinks{
				// Only the immediate parent's URL.
				Self: ptr.String(rest.AbsoluteURL(request, app.AreaHref(parent))),
			},
		}
	}
	for _, opt := range options {
		opt(db, request, &ar, i)
	}
	return i
}

// ConvertAreaSimple converts a simple area ID into a Generic Relationship
// data+links element
func ConvertAreaSimple(request *http.Request, id interface{}) (*app.GenericData, *app.GenericLinks) {
	t := area.APIStringTypeAreas
	i := fmt.Sprint(id)
	data := &app.GenericData{
		Type: &t,
		ID:   &i,
	}
	relatedURL := rest.AbsoluteURL(request, app.AreaHref(i))
	links := &app.GenericLinks{
		Self:    &relatedURL,
		Related: &relatedURL,
	}
	return data, links
}
