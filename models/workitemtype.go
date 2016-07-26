package models

type WorkItemType struct {
	// internal id of this work item type
	Id uint64
	// the name of this work item type. Does not have to be unique.
	Name string
	// definitions of the fields this work item type supports
	Fields map[string]FieldDefinition `sql:"type:jsonb"`
}
