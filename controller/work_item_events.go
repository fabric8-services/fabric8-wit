package controller

import (
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/workitem"
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
	relatedCreatorLink := rest.AbsoluteURL(request, fmt.Sprintf("%s/%s", usersEndpoint, wiEvent.Modifier.String()))
	relatedURL := rest.AbsoluteURL(request, app.WorkitemHref(wiID))
	labelsRelated := relatedURL + "/labels"
	modifier := &app.RelationGeneric{
		Data: &app.GenericData{
			Type: ptr.String(APIStringTypeUser),
			ID:   ptr.String(wiEvent.Modifier.String()),
			Links: &app.GenericLinks{
				Related: &relatedCreatorLink,
			},
		},
	}

	var e *app.Event
	switch wiEvent.Name {
	case workitem.SystemState, workitem.SystemTitle:
		e = &app.Event{
			Type: event.APIStringTypeEvents,
			ID:   &wiEvent.ID,
			Attributes: map[string]interface{}{
				"name":      wiEvent.Name,
				"new":       wiEvent.New,
				"old":       wiEvent.Old,
				"timestamp": wiEvent.Timestamp,
			},

			Relationships: &app.EventRelations{
				Modifier: modifier,
			},
		}
	case workitem.SystemDescription:
		e = &app.Event{
			Type: event.APIStringTypeEvents,
			ID:   &wiEvent.ID,
			Attributes: map[string]interface{}{
				"name":      wiEvent.Name,
				"new":       nil,
				"old":       nil,
				"timestamp": wiEvent.Timestamp,
			},

			Relationships: &app.EventRelations{
				Modifier: modifier,
			},
		}
	case workitem.SystemArea:
		e = &app.Event{
			Type: event.APIStringTypeEvents,
			ID:   &wiEvent.ID,
			Attributes: map[string]interface{}{
				"name":      wiEvent.Name,
				"new":       nil,
				"old":       nil,
				"timestamp": wiEvent.Timestamp,
			},

			Relationships: &app.EventRelations{
				Modifier: modifier,
				OldArea: &app.RelationGeneric{
					Data: ConvertAreaSimple(request, wiEvent.Old),
				},
				NewArea: &app.RelationGeneric{
					Data: ConvertAreaSimple(request, wiEvent.New),
				},
			},
		}
	case workitem.SystemIteration:
		e = &app.Event{
			Type: event.APIStringTypeEvents,
			ID:   &wiEvent.ID,
			Attributes: map[string]interface{}{
				"name":      wiEvent.Name,
				"new":       nil,
				"old":       nil,
				"timestamp": wiEvent.Timestamp,
			},

			Relationships: &app.EventRelations{
				Modifier: modifier,
				OldIteration: &app.RelationGeneric{
					Data: ConvertIterationSimple(request, wiEvent.Old),
				},
				NewIteration: &app.RelationGeneric{
					Data: ConvertIterationSimple(request, wiEvent.New),
				},
			},
		}
	case workitem.SystemAssignees:
		e = &app.Event{
			Type: event.APIStringTypeEvents,
			ID:   &wiEvent.ID,
			Attributes: map[string]interface{}{
				"name":      wiEvent.Name,
				"new":       nil,
				"old":       nil,
				"timestamp": wiEvent.Timestamp,
			},
			Relationships: &app.EventRelations{
				Modifier: modifier,
				OldAssignees: &app.RelationGenericList{
					Data: ConvertUsersSimple(request, wiEvent.Old.([]interface{})),
				},
				NewAssignees: &app.RelationGenericList{
					Data: ConvertUsersSimple(request, wiEvent.New.([]interface{})),
				},
			},
		}
	case workitem.SystemLabels:
		e = &app.Event{
			Type: event.APIStringTypeEvents,
			ID:   &wiEvent.ID,
			Attributes: map[string]interface{}{
				"name":      wiEvent.Name,
				"new":       nil,
				"old":       nil,
				"timestamp": wiEvent.Timestamp,
			},
			Relationships: &app.EventRelations{
				Modifier: modifier,
				OldLabels: &app.RelationGenericList{
					Data: ConvertLabelsSimple(request, wiEvent.Old.([]interface{})),
					Links: &app.GenericLinks{
						Related: &labelsRelated,
					},
				},
				NewLabels: &app.RelationGenericList{
					Data: ConvertLabelsSimple(request, wiEvent.New.([]interface{})),
					Links: &app.GenericLinks{
						Related: &labelsRelated,
					},
				},
			},
		}
	}
	return e
}
