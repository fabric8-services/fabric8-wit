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
