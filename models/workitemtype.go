package models

// WorkItemType represents a work item type as it is stored in the db
type WorkItemType struct {
	// internal id of this work item type
	ID uint64 `gorm:"primary_key"`
	// Version for optimistic concurrency control
	Version int
	// the name of this work item type. Does not have to be unique.
	Name string
	// the id's of the parents, separated with some separator
	ParentPath string
	// definitions of the fields this work item type supports
	Fields map[string]FieldDefinition `sql:"type:jsonb"`
}
