package models

import (
	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
)

// String constants for the local work item types.
const (
	SystemRemoteItemID = "system.remote_item_id"
	SystemTitle        = "system.title"
	SystemDescription  = "system.description"
	SystemState        = "system.state"
	SystemAssignee     = "system.assignee"
	SystemCreator      = "system.creator"

	SystemUserStory        = "system.userstory"
	SystemValueProposition = "system.valueproposition"
	SystemFundamental      = "system.fundamental"
	SystemExperience       = "system.experience"
	SystemFeature          = "system.feature"
	SystemBug              = "system.bug"

	SystemStateOpen       = "open"
	SystemStateNew        = "new"
	SystemStateInProgress = "in progress"
	SystemStateResolved   = "resolved"
	SystemStateClosed     = "closed"
)

// WorkItemType represents a work item type as it is stored in the db
type WorkItemType struct {
	gormsupport.Lifecycle
	// the unique name of this work item type.
	Name string `gorm:"primary_key"`
	// Version for optimistic concurrency control
	Version int
	// the id's of the parents, separated with some separator
	ParentPath string
	// definitions of the fields this work item type supports
	Fields FieldDefinitions `sql:"type:jsonb"`
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = WorkItemType{}
var _ convert.Equaler = (*WorkItemType)(nil)

// Equal returns true if two WorkItemType objects are equal; otherwise false is returned.
func (wit WorkItemType) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemType)
	if !ok {
		return false
	}
	if !wit.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if wit.Version != other.Version {
		return false
	}
	if wit.Name != other.Name {
		return false
	}
	if wit.ParentPath != other.ParentPath {
		return false
	}
	if len(wit.Fields) != len(other.Fields) {
		return false
	}
	for witKey, witVal := range wit.Fields {
		otherVal, keyFound := other.Fields[witKey]
		if !keyFound {
			return false
		}
		if !witVal.Equal(otherVal) {
			return false
		}
	}
	return true
}
