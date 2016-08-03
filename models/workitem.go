package models

// WorkItem represents a work item as it is stored in the database
type WorkItem struct {
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
