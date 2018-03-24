package controller

import (
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/workitem/event"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// EventsController implements the work_item_events resource.
type EventsController struct {
	*goa.Controller
	db     application.DB
	config EventsControllerConfig
}

// EventsControllerConfig the config interface for the WorkitemEventsController
type EventsControllerConfig interface {
	GetCacheControlEvents() string
	//GetCacheControlEvent() string
}

// NewEventsController creates a work_item_events controller.
func NewEventsController(service *goa.Service, db application.DB, config EventsControllerConfig) *EventsController {
	return &EventsController{
		Controller: service.NewController("EventsController"),
		db:         db,
		config:     config}
}

// List runs the list action.
func (c *EventsController) List(ctx *app.ListWorkItemEventsContext) error {
	var eventList []event.Event
	err := application.Transactional(c.db, func(appl application.Application) error {
		var err error
		eventList, err = appl.Events().List(ctx, ctx.WiID)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.ConditionalEntities(eventList, c.config.GetCacheControlEvents, func() error {
		res := &app.EventList{}
		res.Data = ConvertEvents(c.db, ctx.Request, eventList, ctx.WiID)
		return ctx.OK(res)
	})
}

// ConvertEvents from internal to external REST representation
func ConvertEvents(appl application.Application, request *http.Request, eventList []event.Event, wiID uuid.UUID) []*app.Event {
	var ls = []*app.Event{}
	for _, i := range eventList {
		ls = append(ls, ConvertEvent(appl, request, i, wiID))
	}
	return ls
}

// ConvertEvent converts from internal to external REST representation
func ConvertEvent(appl application.Application, request *http.Request, wiEvent event.Event, wiID uuid.UUID) *app.Event {
	var eventAttributes *app.EventAttributes
	eventType := event.APIStringTypeEvents
	eventAttributes = &app.EventAttributes{
		Name:      wiEvent.Name,
		Timestamp: wiEvent.Timestamp,
		OldValue:  &wiEvent.Old,
		NewValue:  &wiEvent.New,
	}
	relatedCreatorLink := rest.AbsoluteURL(request, fmt.Sprintf("%s/%s", usersEndpoint, wiEvent.Modifier.String()))
	e := &app.Event{
		Type:       eventType,
		ID:         &wiEvent.ID,
		Attributes: eventAttributes,
		Relationships: &app.EventRelations{
			Modifier: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: ptr.String(APIStringTypeUser),
					ID:   ptr.String(wiEvent.Modifier.String()),
					Links: &app.GenericLinks{
						Related: &relatedCreatorLink,
					},
				},
			},
		},
	}
	return e
}
