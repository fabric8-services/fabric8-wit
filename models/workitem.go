package models

import (
	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
)

// WorkItem represents a work item as it is stored in the database
type WorkItem struct {
	gormsupport.Lifecycle
	ID uint64 `gorm:"primary_key"`
	// Id of the type of this work item
	Type string
	// Version for optimistic concurrency control
	Version int
	// the field values
	Fields Fields `sql:"type:jsonb"`
}

// TableName implements gorm.tabler
func (w WorkItem) TableName() string {
	return "work_items"
}

// Ensure WorkItem implements the Equaler interface
var _ convert.Equaler = WorkItem{}
var _ convert.Equaler = (*WorkItem)(nil)

// Equal returns true if two WorkItem objects are equal; otherwise false is returned.
func (wi WorkItem) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItem)
	if !ok {
		return false
	}
	if !wi.Lifecycle.Equal(other.Lifecycle) {
		return false
	}

	if wi.Type != other.Type {
		return false
	}
	if wi.ID != other.ID {
		return false
	}
	if wi.Version != other.Version {
		return false
	}
	return wi.Fields.Equal(other.Fields)
}
