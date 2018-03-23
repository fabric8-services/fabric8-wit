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

const (

	// APIStringTypeEvents is the name of event type
	APIStringTypeEvents = "events"

	Assignees = "assignees"
	State     = "state"
)

// WorkItemEventRepository encapsulates retrieval of work item events
type WorkItemEventRepository interface {
	//repository.Exister
	List(ctx context.Context, wiID uuid.UUID) ([]WorkItemEvent, error)
}

// NewWorkItemEventRepository creates a work item event repository based on gorm
func NewWorkItemEventRepository(db *gorm.DB) *GormWorkItemEventRepository {
	return &GormWorkItemEventRepository{
		db:               db,
		workItemRepo:     workitem.NewWorkItemRepository(db),
		wiRevisionRepo:   workitem.NewRevisionRepository(db),
		workItemTypeRepo: workitem.NewWorkItemTypeRepository(db),
		identityRepo:     account.NewIdentityRepository(db),
	}
}

// GormWorkItemEventRepository represents the Gorm model
type GormWorkItemEventRepository struct {
	db               *gorm.DB
	workItemRepo     *workitem.GormWorkItemRepository
	wiRevisionRepo   *workitem.GormRevisionRepository
	workItemTypeRepo *workitem.GormWorkItemTypeRepository
	identityRepo     *account.GormIdentityRepository
}

// List return the events
func (r *GormWorkItemEventRepository) List(ctx context.Context, wiID uuid.UUID) ([]WorkItemEvent, error) {
	revisionList, err := r.wiRevisionRepo.List(ctx, wiID)
	if err != nil {
		return nil, errs.Wrapf(err, "error during fetching event list")
	}
	if revisionList == nil {
		return []WorkItemEvent{}, nil
	}
	wi, err := r.workItemRepo.LoadByID(ctx, wiID)
	if err != nil {
		return nil, errs.Wrapf(err, "error during fetching event list")
	}
	wiType, err := r.workItemTypeRepo.LoadByID(ctx, wi.Type)
	if err != nil {
		return nil, errs.Wrapf(err, "error during fetching event list")
	}

	eventList := []WorkItemEvent{}
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
					wie := WorkItemEvent{
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
					wie := WorkItemEvent{
						ID:        revisionList[k].ID,
						Name:      State,
						Timestamp: revisionList[k].Time,
						Modifier:  modifierID.ID,
						Old:       previousState,
						New:       newState,
					}
					eventList = append(eventList, wie)
				}
			}
		}
	}

	return eventList, nil
}
