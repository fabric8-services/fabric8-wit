package event

import (
	"context"
	"strings"

	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/workitem"
)

// event types
const (
	APIStringTypeEvents = "events"

	Assignees   = "assignees"
	State       = "state"
	Labels      = "labels"
	Iteration   = "iteration"
	Title       = "title"
	Area        = "area"
	Description = "description"
)

// EventNameMap maps system key to a normal string
var EventNameMap = map[string]string{
	workitem.SystemAssignees:   Assignees,
	workitem.SystemLabels:      Labels,
	workitem.SystemState:       State,
	workitem.SystemIteration:   Iteration,
	workitem.SystemTitle:       Title,
	workitem.SystemArea:        Area,
	workitem.SystemDescription: Description,
}

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
	wiType, err := r.workItemTypeRepo.LoadByID(ctx, wi.Type)
	if err != nil {
		return nil, errs.Wrapf(err, "error during fetching event list")
	}

	eventList := []Event{}
	for k := 1; k < len(revisionList); k++ {
		modifierID, err := r.identityRepo.Load(ctx, revisionList[k].ModifierIdentity)
		if err != nil {
			return nil, errs.Wrapf(err, "error during fetching event list")
		}
		for fieldName := range wiType.Fields {
			switch fieldName {
			case workitem.SystemAssignees, workitem.SystemLabels:
				var p []string
				var n []string

				previousValues := revisionList[k-1].WorkItemFields[fieldName]
				newValues := revisionList[k].WorkItemFields[fieldName]
				switch previousValues.(type) {
				case nil:
					p = []string{}
				case []interface{}:
					for _, v := range previousValues.([]interface{}) {
						p = append(p, v.(string))
					}
				}

				switch newValues.(type) {
				case nil:
					n = []string{}
				case []interface{}:
					for _, v := range newValues.([]interface{}) {
						n = append(n, v.(string))
					}

				}
				if len(p) != 0 || len(n) != 0 {
					wie := Event{
						ID:        revisionList[k].ID,
						Name:      EventNameMap[fieldName],
						Timestamp: revisionList[k].Time,
						Modifier:  modifierID.ID,
						Old:       strings.Join(p, ","),
						New:       strings.Join(n, ","),
					}
					eventList = append(eventList, wie)
				}
			default:
				var p string
				var n string

				previousValue := revisionList[k-1].WorkItemFields[fieldName]
				newValue := revisionList[k].WorkItemFields[fieldName]

				switch previousValue.(type) {
				case nil:
					p = ""
				case interface{}:
					p = previousValue.(string)
				}

				switch newValue.(type) {
				case nil:
					n = ""
				case interface{}:
					n = newValue.(string)

				}

				if previousValue != newValue {
					wie := Event{
						ID:        revisionList[k].ID,
						Name:      EventNameMap[fieldName],
						Timestamp: revisionList[k].Time,
						Modifier:  modifierID.ID,
						Old:       p,
						New:       n,
					}
					eventList = append(eventList, wie)
				}
			}
		}
	}

	return eventList, nil
}
