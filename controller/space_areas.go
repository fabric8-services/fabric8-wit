package controller

import (
	"github.com/Sirupsen/logrus"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// SpaceAreasController implements the space-Areas resource.
type SpaceAreasController struct {
	*goa.Controller
	db     application.DB
	config SpaceAreasControllerConfig
}

//SpaceAreasControllerConfig the configuration for the SpaceAreasController
type SpaceAreasControllerConfig interface {
	GetCacheControlAreas() string
}

// NewSpaceAreasController creates a space-Areas controller.
func NewSpaceAreasController(service *goa.Service, db application.DB, config SpaceAreasControllerConfig) *SpaceAreasController {
	return &SpaceAreasController{
		Controller: service.NewController("SpaceAreasController"),
		db:         db,
		config:     config,
	}
}

// List runs the list action.
func (c *SpaceAreasController) List(ctx *app.ListSpaceAreasContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		_, err = appl.Spaces().Load(ctx, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		areas, err := appl.Areas().List(ctx, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		for _, a := range areas {
			logrus.Info("Found space area with ID=", a.ID.String())
		}
		return ctx.ConditionalEntities(areas, c.config.GetCacheControlAreas, func() error {
			res := &app.AreaList{}
			res.Data = ConvertAreas(appl, ctx.RequestData, areas, addResolvedPath)
			return ctx.OK(res)
		})
	})
}
