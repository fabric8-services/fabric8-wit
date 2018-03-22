package event

import (
	"context"

	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/workitem"
)

// APIStringTypeEvents is the name of event type
const APIStringTypeEvents = "events"

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
		modifierID, err := r.identityRepo.Load(ctx, revisionList[1].ModifierIdentity)
		if err != nil {
			return nil, errs.Wrapf(err, "error during fetching event list")
		}
		for fieldName := range wiType.Fields {
			var previousAssignees interface{}
			var newAssignees interface{}
			switch fieldName {
			case workitem.SystemAssignees:
				var p []string
				var n []string

				if k == 0 {
					previousAssignees = nil
				} else {
					previousAssignees = revisionList[k-1].WorkItemFields[workitem.SystemAssignees]
				}
				newAssignees = revisionList[k].WorkItemFields[workitem.SystemAssignees]
				switch previousAssignees.(type) {
				case nil:
					p = []string{}
				case []interface{}:
					for _, v := range previousAssignees.([]interface{}) {
						prev := uuid.FromStringOrNil(v.(string))
						pAssignee, err := r.identityRepo.Load(ctx, prev)
						if err != nil {
							return nil, errs.Wrapf(err, "error during fetching event list")
						}
						p = append(p, pAssignee.Username)
					}
				}

				switch newAssignees.(type) {
				case nil:
					n = []string{}
				case []interface{}:
					for _, v := range newAssignees.([]interface{}) {
						new := uuid.FromStringOrNil(v.(string))
						nAssignee, err := r.identityRepo.Load(ctx, new)
						if err != nil {
							return nil, errs.Wrapf(err, "error during fetching event list")
						}
						n = append(n, nAssignee.Username)
					}

				}
				wie := WorkItemEvent{
					ID:                revisionList[k].ID,
					Name:              "assigned",
					Modifier:          modifierID.Username,
					PreviousAssignees: p,
					NewAssignees:      n,
				}
				eventList = append(eventList, wie)
			}
		}
	}

	return eventList, nil
}
