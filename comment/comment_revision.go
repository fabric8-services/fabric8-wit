package comment

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// CommentRevisionType defines the type of revision for a comment
type CommentRevisionType int

const (
	_ CommentRevisionType = iota // ignore first value by assigning to blank identifier
	// RevisionTypeCreate a work item creation
	RevisionTypeCreate // 1
	// RevisionTypeDelete a work item deletion
	RevisionTypeDelete // 2
	_                  // ignore 3rd value
	// RevisionTypeUpdate a work item update
	RevisionTypeUpdate // 4
)

// CommentRevision represents a version of a work item
type CommentRevision struct {
	ID uuid.UUID `gorm:"primary_key"`
	// the timestamp of the modification
	Time time.Time `gorm:"column:revision_time"`
	// the type of modification
	Type CommentRevisionType `gorm:"column:revision_type"`
	// the identity of author of the comment modification
	ModifierIdentity uuid.UUID `sql:"type:uuid" gorm:"column:modifier_id"`
	// the id of the comment that changed
	CommentID uuid.UUID `gorm:"column:comment_id"`
	// the body of the comment (nil when comment was deleted)
	CommentBody *string `gorm:"column:comment_body"`
	// the markup used to input the comment body (nil when comment was deleted)
	CommentMarkup *string `gorm:"column:comment_markup"`
}

const (
	CommentRevisionTableName = "comment_revisions"
)

// TableName implements gorm.tabler
func (w CommentRevision) TableName() string {
	return CommentRevisionTableName
}
