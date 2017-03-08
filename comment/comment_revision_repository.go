package comment

import (
	"context"

	"fmt"

	"time"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// RevisionRepository encapsulates storage & retrieval of historical versions of comments
type RevisionRepository interface {
	// Create stores a new revision for the given comment.
	Create(ctx context.Context, modifierID uuid.UUID, revisionType RevisionType, comment Comment) error
	// List retrieves all revisions for a given comment
	List(ctx context.Context, workitemID uuid.UUID) ([]Revision, error)
}

// NewRevisionRepository creates a GormCommentRevisionRepository
func NewRevisionRepository(db *gorm.DB) *GormCommentRevisionRepository {
	repository := &GormCommentRevisionRepository{db}
	return repository
}

// GormCommentRevisionRepository implements CommentRevisionRepository using gorm
type GormCommentRevisionRepository struct {
	db *gorm.DB
}

// Create stores a new revision for the given comment.
func (r *GormCommentRevisionRepository) Create(ctx context.Context, modifierID uuid.UUID, revisionType RevisionType, c Comment) error {
	log.Debug(nil, map[string]interface{}{
		"modifier_id":   modifierID,
		"revision_type": revisionType,
	}, "Storing a revision after operation on comment.")
	tx := r.db
	revision := &Revision{
		ModifierIdentity: modifierID,
		Time:             time.Now(),
		Type:             revisionType,
		CommentID:        c.ID,
		CommentParentID:  c.ParentID,
		CommentBody:      &c.Body,
		CommentMarkup:    &c.Markup,
	}
	if revision.Type == RevisionTypeDelete {
		revision.CommentBody = nil
		revision.CommentMarkup = nil
	}

	if err := tx.Create(&revision).Error; err != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to create new comment revision: %s", err.Error()))
	}
	log.Debug(ctx, map[string]interface{}{"commentID": c.ID}, "comment revision occurrence created")
	return nil
}

// List retrieves all revisions for a given comment
func (r *GormCommentRevisionRepository) List(ctx context.Context, commentID uuid.UUID) ([]Revision, error) {
	log.Debug(nil, map[string]interface{}{}, "List all revisions for comment with ID=%v", commentID.String())
	revisions := make([]Revision, 0)
	if err := r.db.Where("comment_id = ?", commentID.String()).Order("revision_time asc").Find(&revisions).Error; err != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("failed to retrieve comment revisions: %s", err.Error()))
	}
	return revisions, nil
}
