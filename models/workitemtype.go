package models

// WorkItemType represents a work item type as it is stored in the db
type WorkItemType struct {
	Lifecycle
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
var _ Equaler = WorkItemType{}
var _ Equaler = (*WorkItemType)(nil)

// Equal returns true if two WorkItemType objects are equal; otherwise false is returned.
func (self WorkItemType) Equal(u Equaler) bool {
	other, ok := u.(WorkItemType)
	if !ok {
		return false
	}
	if !self.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if self.Version != other.Version {
		return false
	}
	if self.Name != other.Name {
		return false
	}
	if self.ParentPath != other.ParentPath {
		return false
	}
	if len(self.Fields) != len(other.Fields) {
		return false
	}
	for selfKey, selfVal := range self.Fields {
		otherVal, keyFound := other.Fields[selfKey]
		if !keyFound {
			return false
		}
		if !selfVal.Equal(otherVal) {
			return false
		}
	}
	return true
}
