package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/goadesign/goa"
)

// TrackersController implements the trackers resource.
type TrackersController struct {
	*goa.Controller
}

// NewTrackersController creates a trackers controller.
func NewTrackersController(service *goa.Service) *TrackersController {
	return &TrackersController{Controller: service.NewController("TrackersController")}
}

// Show runs the show action.
func (c *TrackersController) Show(ctx *app.ShowTrackersContext) error {
	// TrackersController_Show: start_implement

	// Put your logic here

	// TrackersController_Show: end_implement
	res := &app.TrackerItem{}
	return ctx.OK(res)
}
