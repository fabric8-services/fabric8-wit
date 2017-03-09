package link

import (
	"context"

	"fmt"

	"time"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// RevisionRepository encapsulates storage & retrieval of historical versions of work item links
type RevisionRepository interface {
	// Create stores a new revision for the given work item link.
	Create(ctx context.Context, modifierID uuid.UUID, revisionType RevisionType, l WorkItemLink) error
	// List retrieves all revisions for a given work item link
	List(ctx context.Context, workitemID uuid.UUID) ([]Revision, error)
}

// NewRevisionRepository creates a GormCommentRevisionRepository
func NewRevisionRepository(db *gorm.DB) *GormWorkItemLinkRevisionRepository {
	repository := &GormWorkItemLinkRevisionRepository{db}
	return repository
}

// GormCommentRevisionRepository implements CommentRevisionRepository using gorm
type GormWorkItemLinkRevisionRepository struct {
	db *gorm.DB
}

// Create stores a new revision for the given work item link.
func (r *GormWorkItemLinkRevisionRepository) Create(ctx context.Context, modifierID uuid.UUID, revisionType RevisionType, l WorkItemLink) error {
	log.Info(nil, map[string]interface{}{
		"modifier_id":   modifierID,
		"revision_type": revisionType,
	}, "Storing a revision after operation on work item link.")
	tx := r.db
	revision := &Revision{
		ModifierIdentity:     modifierID,
		Time:                 time.Now(),
		Type:                 revisionType,
		WorkItemLinkID:       l.ID,
		WorkItemLinkVersion:  l.Version,
		WorkItemLinkSourceID: l.SourceID,
		WorkItemLinkTargetID: l.TargetID,
		WorkItemLinkTypeID:   l.LinkTypeID,
	}
	if err := tx.Create(&revision).Error; err != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to create new work item link revision: %s", err.Error()))
	}
	log.Debug(ctx, map[string]interface{}{"workItemLink.ID": l.ID}, "work item link revision occurrence created")
	return nil
}

// List retrieves all revisions for a given work item link
func (r *GormWorkItemLinkRevisionRepository) List(ctx context.Context, commentID uuid.UUID) ([]Revision, error) {
	log.Debug(nil, map[string]interface{}{}, "List all revisions for work item link with ID=%v", commentID.String())
	revisions := make([]Revision, 0)
	if err := r.db.Where("work_item_link_id = ?", commentID.String()).Order("revision_time asc").Find(&revisions).Error; err != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("failed to retrieve work item link revisions: %s", err.Error()))
	}
	return revisions, nil
}
