package models

// WorkItem represents a work item as it is stored in the database
type WorkItem struct {
	Lifecycle
	ID uint64 `gorm:"primary_key"`
	// User Readable Name of this item
	Name string
	// Id of the type of this work item
	Type string
	// Version for optimistic concurrency control
	Version int
	// the field values
	Fields Fields `sql:"type:jsonb"`
}

// Ensure WorkItem implements the Equaler interface
var _ Equaler = WorkItem{}
var _ Equaler = (*WorkItem)(nil)

// Equal returns true if two WorkItem objects are equal; otherwise false is returned.
func (self WorkItem) Equal(u Equaler) bool {
	other,ok := u.(WorkItem)
	if !ok {
		return false
	}
	if !self.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if self.Name != other.Name {
		return false
	}
	if self.Type != other.Type {
		return false
	}
	if self.Version != other.Version {
		return false
	}
	return self.Fields.Equal(other.Fields)
}