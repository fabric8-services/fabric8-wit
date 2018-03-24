package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
)

// Show runs the show action.
func (c *EventsController) Show(ctx *app.ShowEventsContext) error {
	// EventsController_Show: start_implement

	// Put your logic here

	// EventsController_Show: end_implement
	res := &app.EventSingle{}
	return ctx.OK(res)
}
