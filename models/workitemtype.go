package models

// WorkItemType represents a work item type as it is stored in the db
type WorkItemType struct {
	Lifecycle
	// internal id of this work item type
	ID uint64
	// the name of this work item type. Does not have to be unique.
	Name string
	// definitions of the fields this work item type supports
	Fields map[string]FieldDefinition `sql:"type:jsonb"`
}

// Ensure Fields implements the Equaler interface
var _ Equaler = WorkItemType{}
var _ Equaler = (*WorkItemType)(nil)

// Equal returns true if two WorkItemType objects are equal; otherwise false is returned.
func (self WorkItemType) Equal(u Equaler) bool {
	other := u.(WorkItemType)
	if !self.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if self.ID != other.ID {
		return false
	}
	if self.Name != other.Name {
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