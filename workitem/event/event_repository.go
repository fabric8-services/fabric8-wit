package event

import (
	"context"
	"fmt"

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
				case workitem.KindLabel, workitem.KindUser:
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
					if len(p) != 0 || len(n) != 0 {
						fmt.Println("label", fieldName)
						wie := Event{
							ID:        uuid.NewV4(),
							Name:      fieldName,
							Timestamp: revisionList[k].Time,
							Modifier:  modifierID.ID,
							Old:       p,
							New:       n,
						}
						eventList = append(eventList, wie)
					}
				default:
					return nil, errors.NewNotFoundError("Unkown field:", fieldName)

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

				if previousValue != newValue {
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
					previousValue := revisionList[k-1].WorkItemFields[fieldName]
					newValue := revisionList[k].WorkItemFields[fieldName]

					pv := rendering.NewMarkupContentFromMap(previousValue.(map[string]interface{}))
					nv := rendering.NewMarkupContentFromMap(newValue.(map[string]interface{}))
					if pv.Content != nv.Content {
						wie := Event{
							ID:        revisionList[k].ID,
							Name:      fieldName,
							Timestamp: revisionList[k].Time,
							Modifier:  modifierID.ID,
							Old:       pv.Content,
							New:       nv.Content,
						}
						eventList = append(eventList, wie)
					}
				case workitem.KindString:
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

					if previousValue != newValue {
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
				return nil, errors.NewNotFoundError("Unkown field:", fieldName)
			}

		}
	}
	return eventList, nil
}
