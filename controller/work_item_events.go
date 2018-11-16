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
	var eventList event.List
	err := application.Transactional(c.db, func(appl application.Application) error {
		var err error
		eventList, err = appl.Events().List(ctx, ctx.WiID)
		if err != nil {
			return errs.Wrap(err, "list events model failed")
		}
		if ctx.RevisionID != nil {
			eventList = eventList.FilterByRevisionID(*ctx.RevisionID)
		}
		return nil
	})

	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	var convertedEvents []*app.Event
	return ctx.ConditionalEntities(eventList, c.config.GetCacheControlEvents, func() error {
		convertedEvents, err = ConvertEvents(ctx, c.db, ctx.Request, eventList, ctx.WiID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "failed to convert events"))
		}
		return ctx.OK(&app.EventList{
			Data: convertedEvents,
		})
	})
}

// ConvertEvents from internal to external REST representation
func ConvertEvents(ctx context.Context, appl application.Application, request *http.Request, eventList []event.Event, wiID uuid.UUID) ([]*app.Event, error) {
	var ls = []*app.Event{}
	for _, i := range eventList {
		converted, err := ConvertEvent(ctx, appl, request, i, wiID)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to convert event: %+v", i)
		}
		ls = append(ls, converted)
	}
	return ls, nil
}

// ConvertEvent converts from internal to external REST representation
func ConvertEvent(ctx context.Context, appl application.Application, req *http.Request, wiEvent event.Event, wiID uuid.UUID) (*app.Event, error) {
	// find out about background details on the field that was modified
	wit, err := appl.WorkItemTypes().Load(ctx, wiEvent.WorkItemTypeID)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to load work item type: %s", wiEvent.WorkItemTypeID)
	}
	modifierData, modifierLinks := ConvertUserSimple(req, wiEvent.Modifier)
	e := app.Event{
		Type: event.APIStringTypeEvents,
		ID:   uuid.NewV4(),
		Attributes: &app.EventAttributes{
			Name:       wiEvent.Name,
			Timestamp:  wiEvent.Timestamp,
			RevisionID: wiEvent.RevisionID,
		},
		Relationships: &app.EventRelations{
			Modifier: &app.RelationGeneric{
				Data:  modifierData,
				Links: modifierLinks,
			},
			WorkItemType: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Self: ptr.String(rest.AbsoluteURL(req, app.WorkitemtypeHref(wit.ID))),
				},
				Data: &app.GenericData{
					ID:   ptr.String(wit.ID.String()),
					Type: ptr.String(APIStringTypeWorkItemType),
				},
			},
		},
	}

	if wiEvent.Name == event.WorkitemTypeChangeEvent {
		oldTypeUUID, ok := wiEvent.Old.(uuid.UUID)
		if !ok {
			return nil, errs.Errorf("failed to convert old workitem type ID to UUID: %s", wiEvent.Old)
		}
		newTypeUUID, ok := wiEvent.New.(uuid.UUID)
		if !ok {
			return nil, errs.Errorf("failed to convert new workitem type ID to UUID: %s", wiEvent.New)
		}
		e.Relationships.OldValue = &app.RelationGenericList{
			Data: []*app.GenericData{
				{
					ID:   ptr.String(oldTypeUUID.String()),
					Type: ptr.String(APIStringTypeWorkItemType),
				},
			},
		}
		e.Relationships.NewValue = &app.RelationGenericList{
			Data: []*app.GenericData{
				{
					ID:   ptr.String(newTypeUUID.String()),
					Type: ptr.String(APIStringTypeWorkItemType),
				},
			},
		}
		return &e, nil
	}

	fieldName := wiEvent.Name
	fieldDef, ok := wit.Fields[fieldName]
	if !ok {
		return nil, errs.Errorf("failed to find field %q in work item type: %s (%s)", fieldName, wit.Name, wit.ID)
	}

	// convertVal returns the given value converted from storage space to
	// JSONAPI space. If the given value is supposed to be stored as a
	// relationship in JSONAPI, the second return value will be true.
	convertVal := func(kind workitem.Kind, val interface{}) (interface{}, bool) {
		switch kind {
		case workitem.KindString,
			workitem.KindInteger,
			workitem.KindFloat,
			workitem.KindBoolean,
			workitem.KindURL,
			workitem.KindMarkup,
			workitem.KindInstant:
			return val, false
		case workitem.KindIteration:
			data, _ := ConvertIterationSimple(req, val)
			return data, true
		case workitem.KindUser:
			data, _ := ConvertUserSimple(req, val)
			return data, true
		case workitem.KindLabel:
			data := ConvertLabelSimple(req, val)
			return data, true
		case workitem.KindBoardColumn:
			data := ConvertBoardColumnSimple(req, val)
			return data, true
		case workitem.KindArea:
			data, _ := ConvertAreaSimple(req, val)
			return data, true
		case workitem.KindCodebase:
			data, _ := ConvertCodebaseSimple(req, val)
			return data, true
		}
		return nil, false
	}

	kind := fieldDef.Type.GetKind()
	if kind == workitem.KindEnum {
		enumType, ok := fieldDef.Type.(workitem.EnumType)
		if !ok {
			return nil, errs.Errorf("failed to convert field %q to enum type: %+v", fieldName, fieldDef)
		}
		kind = enumType.BaseType.GetKind()
	}

	// handle all single value fields (including enums)
	if kind != workitem.KindList {
		oldVal, useRel := convertVal(kind, wiEvent.Old)
		newVal, _ := convertVal(kind, wiEvent.New)
		if useRel {
			if wiEvent.Old != nil {
				e.Relationships.OldValue = &app.RelationGenericList{Data: []*app.GenericData{oldVal.(*app.GenericData)}}
			}
			if wiEvent.New != nil {
				e.Relationships.NewValue = &app.RelationGenericList{Data: []*app.GenericData{newVal.(*app.GenericData)}}
			}
		} else {
			if oldVal != nil {
				e.Attributes.OldValue = &oldVal
			}
			if newVal != nil {
				e.Attributes.NewValue = &newVal
			}
		}
		return &e, nil
	}

	// handle multi-value fields
	listType, ok := fieldDef.Type.(workitem.ListType)
	if !ok {
		return nil, errs.Errorf("failed to convert field %q to list type: %+v", fieldName, fieldDef)
	}
	componentTypeKind := listType.ComponentType.GetKind()

	arrOld, ok := wiEvent.Old.([]interface{})
	if !ok {
		return nil, errs.Errorf("failed to convert old value of field %q to []interface{}: %+v", fieldName, wiEvent.Old)
	}
	arrNew, ok := wiEvent.New.([]interface{})
	if !ok {
		return nil, errs.Errorf("failed to convert new value of field %q to []interface{}: %+v", fieldName, wiEvent.New)
	}

	for i, v := range arrOld {
		oldVal, useRel := convertVal(componentTypeKind, v)
		if useRel {
			if i == 0 {
				e.Relationships.OldValue = &app.RelationGenericList{
					Data: make([]*app.GenericData, len(arrOld)),
				}
			}
			e.Relationships.OldValue.Data[i] = oldVal.(*app.GenericData)
		} else {
			if i == 0 {
				e.Attributes.OldValue = ptr.Interface(make([]interface{}, len(arrOld)))
			}
			(*e.Attributes.OldValue).([]interface{})[i] = oldVal
		}
	}

	for i, v := range arrNew {
		newVal, useRel := convertVal(componentTypeKind, v)
		if useRel {
			if i == 0 {
				e.Relationships.NewValue = &app.RelationGenericList{
					Data: make([]*app.GenericData, len(arrNew)),
				}
			}
			e.Relationships.NewValue.Data[i] = newVal.(*app.GenericData)
		} else {
			if i == 0 {
				e.Attributes.NewValue = ptr.Interface(make([]interface{}, len(arrNew)))
			}
			(*e.Attributes.NewValue).([]interface{})[i] = newVal
		}
	}

	return &e, nil
}
