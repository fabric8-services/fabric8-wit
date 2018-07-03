package event

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/workitem"
)

// APIStringTypeEvents represent the type of event
const APIStringTypeEvents = "events"

// Repository encapsulates retrieval of work item events
type Repository interface {
	//repository.Exister
	List(ctx context.Context, wiID uuid.UUID) ([]Event, error)
}

// NewEventRepository creates a work item event repository based on gorm
func NewEventRepository(db *gorm.DB) *GormEventRepository {
	return &GormEventRepository{
		db:               db,
		workItemRepo:     workitem.NewWorkItemRepository(db),
		wiRevisionRepo:   workitem.NewRevisionRepository(db),
		workItemTypeRepo: workitem.NewWorkItemTypeRepository(db),
		identityRepo:     account.NewIdentityRepository(db),
	}
}

// GormEventRepository represents the Gorm model
type GormEventRepository struct {
	db               *gorm.DB
	workItemRepo     *workitem.GormWorkItemRepository
	wiRevisionRepo   *workitem.GormRevisionRepository
	workItemTypeRepo *workitem.GormWorkItemTypeRepository
	identityRepo     *account.GormIdentityRepository
}

// List return the events
func (r *GormEventRepository) List(ctx context.Context, wiID uuid.UUID) ([]Event, error) {
	revisionList, err := r.wiRevisionRepo.List(ctx, wiID)
	if err != nil {
		return nil, errs.Wrapf(err, "error during fetching event list")
	}
	if revisionList == nil {
		return []Event{}, nil
	}
	wi, err := r.workItemRepo.LoadByID(ctx, wiID)
	if err != nil {
		return nil, errs.Wrapf(err, "error during fetching event list")
	}
	wiType, err := r.workItemTypeRepo.Load(ctx, wi.Type)
	if err != nil {
		return nil, errs.Wrapf(err, "error during fetching event list")
	}

	eventList := []Event{}
	for k := 1; k < len(revisionList); k++ {
		modifierID, err := r.identityRepo.Load(ctx, revisionList[k].ModifierIdentity)
		if err != nil {
			return nil, errs.Wrapf(err, "error during fetching event list")
		}
		for fieldName, field := range wiType.Fields {
			switch fieldType := field.Type.(type) {
			case workitem.ListType:
				switch fieldType.ComponentType.Kind {
				case workitem.KindLabel, workitem.KindUser, workitem.KindBoardColumn:
					var p []interface{}
					var n []interface{}

					previousValues := revisionList[k-1].WorkItemFields[fieldName]
					newValues := revisionList[k].WorkItemFields[fieldName]
					switch previousValues.(type) {
					case nil:
						p = []interface{}{}
					case []interface{}:
						for _, v := range previousValues.([]interface{}) {
							p = append(p, v)
						}
					}

					switch newValues.(type) {
					case nil:
						n = []interface{}{}
					case []interface{}:
						for _, v := range newValues.([]interface{}) {
							n = append(n, v)
						}

					}

					// Avoid duplicate entries for empty labels or assignees
					if reflect.DeepEqual(p, n) == false {
						wie := Event{
							ID:        revisionList[k].ID,
							Name:      fieldName,
							Timestamp: revisionList[k].Time,
							Modifier:  modifierID.ID,
							Old:       p,
							New:       n,
						}
						eventList = append(eventList, wie)
					}
				default:
					return nil, errors.NewNotFoundError("Unknown field:", fieldName)
				}
			case workitem.EnumType:
				var p string
				var n string

				previousValue := revisionList[k-1].WorkItemFields[fieldName]
				newValue := revisionList[k].WorkItemFields[fieldName]

				switch previousValue.(type) {
				case nil:
					p = ""
				case interface{}:
					p, _ = previousValue.(string)
				}

				switch newValue.(type) {
				case nil:
					n = ""
				case interface{}:
					n, _ = newValue.(string)

				}
				if p != n {
					wie := Event{
						ID:        revisionList[k].ID,
						Name:      fieldName,
						Timestamp: revisionList[k].Time,
						Modifier:  modifierID.ID,
						Old:       p,
						New:       n,
					}
					eventList = append(eventList, wie)
				}
			case workitem.SimpleType:
				switch fieldType.Kind {
				case workitem.KindMarkup:
					var p string
					var n string

					previousValue := revisionList[k-1].WorkItemFields[fieldName]
					newValue := revisionList[k].WorkItemFields[fieldName]

					switch previousValue.(type) {
					case nil:
						p = ""
					case map[string]interface{}:
						pv := rendering.NewMarkupContentFromMap(previousValue.(map[string]interface{}))
						p = pv.Content
					}

					switch newValue.(type) {
					case nil:
						n = ""
					case map[string]interface{}:
						nv := rendering.NewMarkupContentFromMap(newValue.(map[string]interface{}))
						n = nv.Content

					}

					if p != n {
						wie := Event{
							ID:        revisionList[k].ID,
							Name:      fieldName,
							Timestamp: revisionList[k].Time,
							Modifier:  modifierID.ID,
							Old:       p,
							New:       n,
						}
						eventList = append(eventList, wie)
					}
				case workitem.KindString, workitem.KindIteration, workitem.KindArea, workitem.KindFloat, workitem.KindInteger:
					var p string
					var n string

					previousValue := revisionList[k-1].WorkItemFields[fieldName]
					newValue := revisionList[k].WorkItemFields[fieldName]

					switch v := previousValue.(type) {
					case nil:
						p = ""
					case float32, float64, int:
						p = fmt.Sprintf("%g", previousValue)
					case string:
						p = v
					default:
						return nil, errors.NewConversionError("Failed to convert")
					}

					switch v := newValue.(type) {
					case nil:
						n = ""
					case float32, float64, int:
						n = fmt.Sprintf("%g", newValue)
					case string:
						n = v
					default:
						return nil, errors.NewConversionError("Failed to convert")
					}
					if p != n {
						wie := Event{
							ID:        revisionList[k].ID,
							Name:      fieldName,
							Timestamp: revisionList[k].Time,
							Modifier:  modifierID.ID,
							Old:       p,
							New:       n,
						}
						eventList = append(eventList, wie)
					}
				}
			default:
				return nil, errors.NewNotFoundError("Unknown field:", fieldName)
			}
		}
	}
	return eventList, nil
}
