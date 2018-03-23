package controller

import (
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/workitem/event"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// WorkItemEventsController implements the work_item_events resource.
type WorkItemEventsController struct {
	*goa.Controller
	db     application.DB
	config WorkItemEventsControllerConfig
}

// WorkItemEventsControllerConfig the config interface for the WorkitemEventsController
type WorkItemEventsControllerConfig interface {
	//GetCacheControlWorkItemEvents() string
	//GetCacheControlWorkItemEvent() string
}

// NewWorkItemEventsController creates a work_item_events controller.
func NewWorkItemEventsController(service *goa.Service, db application.DB, config WorkItemEventsControllerConfig) *WorkItemEventsController {
	return &WorkItemEventsController{
		Controller: service.NewController("WorkItemEventsController"),
		db:         db,
		config:     config}
}

// List runs the list action.
func (c *WorkItemEventsController) List(ctx *app.ListWorkItemEventsContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		eventList, err := appl.Events().List(ctx, ctx.WiID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		res := &app.EventList{}
		res.Data = ConvertEvents(appl, ctx.Request, eventList, ctx.WiID)
		return ctx.OK(res)
	})
}

// ConvertEvents from internal to external REST representation
func ConvertEvents(appl application.Application, request *http.Request, eventList []event.WorkItemEvent, wiID uuid.UUID) []*app.Event {
	var ls = []*app.Event{}
	for _, i := range eventList {
		ls = append(ls, ConvertEvent(appl, request, i, wiID))
	}
	return ls
}

// ConvertEvent converts from internal to external REST representation
func ConvertEvent(appl application.Application, request *http.Request, wiEvent event.WorkItemEvent, wiID uuid.UUID) *app.Event {
	var eventAttributes *app.EventAttributes
	eventType := event.APIStringTypeEvents
	switch wiEvent.Name {
	case event.Assignees:
		eventAttributes = &app.EventAttributes{
			Name:      wiEvent.Name,
			Modifier:  wiEvent.Modifier,
			Timestamp: wiEvent.Timestamp,
			Old:       &wiEvent.Old,
			New:       &wiEvent.New,
		}
	}
	e := &app.Event{
		Type:       eventType,
		ID:         &wiEvent.ID,
		Attributes: eventAttributes,
	}
	return e
}
