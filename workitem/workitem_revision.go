package workitem

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// RevisionType defines the type of revision for a work item
type RevisionType int

const (
	_ RevisionType = iota // ignore first value by assigning to blank identifier
	// RevisionTypeWorkItemCreate a work item creation
	RevisionTypeWorkItemCreate // 1
	// RevisionTypeWorkItemDelete a work item deletion
	RevisionTypeWorkItemDelete // 2
	_                          // ignore 3rd value
	// RevisionTypeWorkItemUpdate a work item update
	RevisionTypeWorkItemUpdate // 4
)

// WorkItemRevision represents a version of a work item
type WorkItemRevision struct {
	ID uuid.UUID `gorm:"primary_key"`
	// the timestamp of the modification
	Time time.Time `gorm:"column:revision_time"`
	// the type of modification
	Type RevisionType `gorm:"column:revision_type"`
	// the identity of author of the workitem modification
	ModifierIdentity uuid.UUID `sql:"type:uuid" gorm:"column:modifier_id"`
	// the id of the work item that changed
	WorkItemID uint64 `gorm:"column:work_item_id"`
	// Id of the type of this work item
	WorkItemType string `gorm:"column:work_item_type"`
	// Version of the workitem that was modified
	WorkItemVersion int `gorm:"column:work_item_version"`
	// the field values
	WorkItemFields Fields `gorm:"column:work_item_fields" sql:"type:jsonb"`
}

const (
	workitemRevisionTableName = "work_item_revisions"
)

// TableName implements gorm.tabler
func (w WorkItemRevision) TableName() string {
	return workitemRevisionTableName
}
