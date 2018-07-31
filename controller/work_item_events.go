package controller

import (
	"context"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/event"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
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
		return errs.Wrap(err, "list events model failed")
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	var convertedEvents []*app.Event
	return ctx.ConditionalEntities(eventList, c.config.GetCacheControlEvents, func() error {
		wi, err := c.db.WorkItems().LoadByID(ctx, ctx.WiID)
		if err != nil {
			return errs.Wrapf(err, "failed to load work item with ID: %s", ctx.WiID)
		}
		convertedEvents, err = ConvertEvents(ctx, c.db, ctx.Request, eventList, ctx.WiID, wi.SpaceID)
		if err != nil {
			return errs.Wrapf(err, "failed to convert events")
		}
		return ctx.OK(&app.EventList{
			Data: convertedEvents,
		})
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.OK(&app.EventList{
		Data: convertedEvents,
	})
}

// ConvertEvents from internal to external REST representation
func ConvertEvents(ctx context.Context, appl application.Application, request *http.Request, eventList []event.Event, wiID uuid.UUID, spaceID uuid.UUID) ([]*app.Event, error) {
	var ls = []*app.Event{}
	for _, i := range eventList {
		converted, err := ConvertEvent(ctx, appl, request, i, wiID, spaceID)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to convert event: %+v", i)
		}
		ls = append(ls, converted)
	}
	return ls, nil
}

// ConvertEvent converts from internal to external REST representation
func ConvertEvent(ctx context.Context, appl application.Application, req *http.Request, wiEvent event.Event, wiID uuid.UUID, spaceID uuid.UUID) (*app.Event, error) {
	// find out about background details on the field that was modified
	wit, err := appl.WorkItemTypes().Load(ctx, wiEvent.WorkItemTypeID)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to load work item type: %s", wiEvent.WorkItemTypeID)
	}
	fieldName := wiEvent.Name
	fieldDef, ok := wit.Fields[fieldName]
	if !ok {
		return nil, errs.Errorf("failed to find field \"%s\" in work item type: %s (%s)", fieldName, wit.Name, wit.ID)
	}

	e := app.Event{
		Type: event.APIStringTypeEvents,
		ID:   wiEvent.ID,
		Attributes: &app.EventAttributes{
			Name:      wiEvent.Name,
			Timestamp: wiEvent.Timestamp,
		},
		Relationships: &app.EventRelations{
			Modifier: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: ptr.String(APIStringTypeUser),
					ID:   ptr.String(wiEvent.Modifier.String()),
				},
			},
			WorkItemType: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Self: ptr.String(rest.AbsoluteURL(req, app.WorkitemtypeHref(wit.ID))),
				},
			},
		},
	}

	handle := func(kind workitem.Kind, val interface{}) (interface{}, bool) {
		switch kind {
		case workitem.KindString,
			workitem.KindInteger,
			workitem.KindFloat,
			workitem.KindBoolean,
			workitem.KindURL,
			workitem.KindMarkup,
			workitem.KindDuration, // TODO(kwk): get rid of duration
			workitem.KindInstant:
			return val, false
		case workitem.KindIteration:
			return ConvertIterationSimple(req, val), true
		case workitem.KindUser:
			return ConvertUserSimple(req, val), true
		case workitem.KindLabel:
			return ConvertLabelSimple(req, val), true
		case workitem.KindBoardColumn:
			return ConvertBoardColumnSimple(req, val), true
		case workitem.KindArea:
			return ConvertAreaSimple(req, val), true
		case workitem.KindCodebase:
			return ConvertCodebaseSimple(req, val), true
		}
		return nil, false
	}

	kind := fieldDef.Type.GetKind()
	if kind == workitem.KindEnum {
		enumType, ok := fieldDef.Type.(workitem.EnumType)
		if !ok {
			return nil, errs.Errorf("failed to convert field \"%s\" to enum type: %+v", fieldName, fieldDef)
		}
		kind = enumType.BaseType.GetKind()
	}

	// handle all single value fields (including enums)
	if kind != workitem.KindList {
		oldVal, useRel := handle(kind, wiEvent.Old)
		newVal, _ := handle(kind, wiEvent.New)
		// update the event with the given values and find out if
		if useRel {
			e.Relationships.OldValue = &app.RelationGenericList{
				Data: []*app.GenericData{
					oldVal.(*app.GenericData),
				},
			}
			e.Relationships.NewValue = &app.RelationGenericList{
				Data: []*app.GenericData{
					newVal.(*app.GenericData),
				},
			}
		} else {
			e.Attributes.OldValue = &oldVal
			e.Attributes.NewValue = &newVal
		}
		return &e, nil
	}

	// handle multi-value fields
	listType, ok := fieldDef.Type.(workitem.ListType)
	if !ok {
		return nil, errs.Errorf("failed to convert field \"%s\" to list type: %+v", fieldName, fieldDef)
	}
	componentTypeKind := listType.ComponentType.GetKind()

	arrOld, ok := wiEvent.Old.([]interface{})
	if !ok {
		return nil, errs.Errorf("failed to convert old value of field \"%s\" to []interface{}: %+v", fieldName, wiEvent.Old)
	}
	arrNew, ok := wiEvent.New.([]interface{})
	if !ok {
		return nil, errs.Errorf("failed to convert old value of field \"%s\" to []interface{}: %+v", fieldName, wiEvent.Old)
	}

	for i, o := range arrOld {
		oldVal, useRel := handle(componentTypeKind, o)
		if useRel {
			if i == 0 {
				e.Relationships.OldValue = &app.RelationGenericList{
					Data: make([]*app.GenericData, len(arrOld)),
				}
			}
			e.Relationships.OldValue.Data[i] = oldVal.(*app.GenericData)
		} else {
			if i == 0 {
				var ifObj interface{} = make([]interface{}, len(arrOld))
				e.Attributes.OldValue = &ifObj
			}
			(*e.Attributes.OldValue).([]interface{})[i] = oldVal
		}
	}

	for i, n := range arrNew {
		newVal, useRel := handle(componentTypeKind, n)
		if useRel {
			if i == 0 {
				e.Relationships.NewValue = &app.RelationGenericList{
					Data: make([]*app.GenericData, len(arrNew)),
				}
			}
			e.Relationships.NewValue.Data[i] = newVal.(*app.GenericData)
		} else {
			if i == 0 {
				var ifObj interface{} = make([]interface{}, len(arrNew))
				e.Attributes.NewValue = &ifObj
			}
			(*e.Attributes.NewValue).([]interface{})[i] = newVal
		}
	}
	return &e, nil
}
