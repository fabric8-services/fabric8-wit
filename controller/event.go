package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/workitem/event"

	"github.com/goadesign/goa"
)

// EventController implements the event resource.
type EventController struct {
	*goa.Controller
	db application.DB
}

// NewEventController creates a event controller.
func NewEventController(service *goa.Service, db application.DB) *EventController {
	return &EventController{
		Controller: service.NewController("EventController"),
		db:         db,
	}
}

// Show runs the show action.
func (e *EventController) Show(ctx *app.ShowEventContext) error {
	var events []event.Event
	err := application.Transactional(e.db, func(appl application.Application) (err error) {
		events, err = appl.Events().List(ctx, ctx.EventID)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	// Put your logic here

	// EventController_Show: end_implement
	res := &app.EventList{}
	return ctx.OK(res)
}
