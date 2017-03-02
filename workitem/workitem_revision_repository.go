package workitem

import (
	"context"

	"fmt"

	"time"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// WorkItemRevisionRepository encapsulates storage & retrieval of historical versions of work items
type WorkItemRevisionRepository interface {
	// Create stores a new revision for the given work item.
	Create(ctx context.Context, modifierID uuid.UUID, revisionType RevisionType, workitem WorkItem) error
	// List retrieves all revisions for a given work item
	List(ctx context.Context, workitemID string) ([]WorkItemRevision, error)
}

// NewWorkItemRevisionRepository creates a GormWorkItemRevisionRepository
func NewWorkItemRevisionRepository(db *gorm.DB) *GormWorkItemRevisionRepository {
	repository := &GormWorkItemRevisionRepository{db}
	return repository
}

// GormWorkItemRevisionRepository implements WorkItemRevisionRepository using gorm
type GormWorkItemRevisionRepository struct {
	db *gorm.DB
}

// Create stores a new revision for the given work item.
func (r *GormWorkItemRevisionRepository) Create(ctx context.Context, modifierID uuid.UUID, revisionType RevisionType, workitem WorkItem) error {
	log.Info(nil, map[string]interface{}{
		"pkg":              "workitem",
		"ModifierIdentity": modifierID,
	}, "Storing a revision after operation on work item.")
	tx := r.db
	workitemRevision := &WorkItemRevision{
		Time:             time.Now(),
		ModifierIdentity: modifierID,
		Type:             revisionType,
		WorkItemID:       workitem.ID,
		WorkItemType:     workitem.Type,
		WorkItemVersion:  workitem.Version,
		WorkItemFields:   workitem.Fields,
	}

	if err := tx.Create(&workitemRevision).Error; err != nil {
		return errors.NewInternalError(fmt.Sprintf("Failed to create new work item revision: %s", err.Error()))
	}
	log.Debug(ctx, map[string]interface{}{"wi.ID": workitem.ID}, "Work item revision occurrence created")
	return nil
}

// List retrieves all revisions for a given work item
func (r *GormWorkItemRevisionRepository) List(ctx context.Context, workitemID string) ([]WorkItemRevision, error) {
	log.Debug(nil, map[string]interface{}{
		"pkg": "workitem",
	}, "List all revisions for work item with ID=%v", workitemID)
	revisions := make([]WorkItemRevision, 0)
	if err := r.db.Where("work_item_id = ?", workitemID).Order("work_item_version asc").Find(&revisions).Error; err != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("Failed to retrieve work item revisions: %s", err.Error()))
	}
	return revisions, nil
}
