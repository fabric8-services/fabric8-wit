package workitem

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

const (
	// CREATE_WORKITEM_REVISION_TYPE a work item creation
	CREATE_WORKITEM_REVISION_TYPE = 1
	// DELETE_WORKITEM_REVISION_TYPE a work item deletion
	DELETE_WORKITEM_REVISION_TYPE = 2
	// UPDATE a work item update
	UPDATE_WORKITEM_REVISION_TYPE = 4
)

// WorkItemRevision represents a version of a work item
type WorkItemRevision struct {
	ID uint64 `gorm:"primary_key"`
	// the timestamp of the modification
	Time time.Time `gorm:"column:revision_time"`
	// the type of modification
	Type int `gorm:"column:revision_type"`
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
