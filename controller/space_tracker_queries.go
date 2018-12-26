package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/goadesign/goa"
)

// SpaceTrackerQueriesController implements the space_tracker_queries resource.
type SpaceTrackerQueriesController struct {
	*goa.Controller
	db     application.DB
	config SpaceTrackerQueriesControllerConfig
}

//SpaceTrackerQueriesControllerConfig the configuration for the SpaceTrackerQueriesController
type SpaceTrackerQueriesControllerConfig interface {
	GetCacheControlTrackerQueries() string
}

// NewSpaceTrackerQueriesController creates a space_tracker_queries controller.
func NewSpaceTrackerQueriesController(service *goa.Service, db application.DB, config SpaceTrackerQueriesControllerConfig) *SpaceTrackerQueriesController {
	return &SpaceTrackerQueriesController{
		Controller: service.NewController("SpaceTrackerQueriesController"),
		db:         db,
		config:     config,
	}
}

// List runs the list action.
func (c *SpaceTrackerQueriesController) List(ctx *app.ListSpaceTrackerQueriesContext) error {
	var trackerQueries []remoteworkitem.TrackerQuery
	err := application.Transactional(c.db, func(appl application.Application) error {
		err := appl.Spaces().CheckExists(ctx, ctx.SpaceID)
		if err != nil {
			return err

		}
		trackerQueries, err = appl.TrackerQueries().List(ctx, ctx.SpaceID)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.ConditionalEntities(trackerQueries, c.config.GetCacheControlTrackerQueries, func() error {
		res := &app.TrackerQueryList{}
		res.Data = ConvertTrackerQueriesToApp(c.db, ctx.Request, trackerQueries)
		return ctx.OK(res)
	})

}
