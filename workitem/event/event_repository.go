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

	Assignees = "assignees"
	State     = "state"
	Labels    = "labels"
	Iteration = "iteration"
)

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
			case workitem.SystemAssignees:
				var p []string
				var n []string

				previousAssignees := revisionList[k-1].WorkItemFields[workitem.SystemAssignees]
				newAssignees := revisionList[k].WorkItemFields[workitem.SystemAssignees]
				switch previousAssignees.(type) {
				case nil:
					p = []string{}
				case []interface{}:
					for _, v := range previousAssignees.([]interface{}) {
						p = append(p, v.(string))
					}
				}

				switch newAssignees.(type) {
				case nil:
					n = []string{}
				case []interface{}:
					for _, v := range newAssignees.([]interface{}) {
						n = append(n, v.(string))
					}

				}
				if len(p) != 0 || len(n) != 0 {
					wie := Event{
						ID:        revisionList[k].ID,
						Name:      Assignees,
						Timestamp: revisionList[k].Time,
						Modifier:  modifierID.ID,
						Old:       strings.Join(p, ","),
						New:       strings.Join(n, ","),
					}
					eventList = append(eventList, wie)
				}
			case workitem.SystemState:
				previousState := revisionList[k-1].WorkItemFields[workitem.SystemState].(string)
				newState := revisionList[k].WorkItemFields[workitem.SystemState].(string)
				if previousState != newState {
					wie := Event{
						ID:        revisionList[k].ID,
						Name:      State,
						Timestamp: revisionList[k].Time,
						Modifier:  modifierID.ID,
						Old:       previousState,
						New:       newState,
					}
					eventList = append(eventList, wie)
				}
			case workitem.SystemLabels:
				var p []string
				var n []string

				previousLabels := revisionList[k-1].WorkItemFields[workitem.SystemLabels]
				newLabels := revisionList[k].WorkItemFields[workitem.SystemLabels]
				switch previousLabels.(type) {
				case nil:
					p = []string{}
				case []interface{}:
					for _, v := range previousLabels.([]interface{}) {
						p = append(p, v.(string))
					}
				}

				switch newLabels.(type) {
				case nil:
					n = []string{}
				case []interface{}:
					for _, v := range newLabels.([]interface{}) {
						n = append(n, v.(string))
					}

				}
				if len(p) != 0 || len(n) != 0 {
					wie := Event{
						ID:        revisionList[k].ID,
						Name:      Labels,
						Timestamp: revisionList[k].Time,
						Modifier:  modifierID.ID,
						Old:       strings.Join(p, ","),
						New:       strings.Join(n, ","),
					}
					eventList = append(eventList, wie)
				}
			case workitem.SystemIteration:
				var p string
				var n string

				previousIteration := revisionList[k-1].WorkItemFields[workitem.SystemIteration]
				newIteration := revisionList[k].WorkItemFields[workitem.SystemIteration]

				switch previousIteration.(type) {
				case nil:
					p = ""
				case interface{}:
					p = previousIteration.(string)
				}

				switch newIteration.(type) {
				case nil:
					n = ""
				case interface{}:
					n = newIteration.(string)
				}
				if previousIteration != newIteration {
					wie := Event{
						ID:        revisionList[k].ID,
						Name:      Iteration,
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
