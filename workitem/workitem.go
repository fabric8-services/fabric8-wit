package workitem

import uuid "github.com/satori/go.uuid"

//TODO: remove tags
// WorkItem the model structure for the work item.
type WorkItem struct {
	// Order of the workitem in the list
	ExecutionOrder float64 `form:"ExecutionOrder" json:"ExecutionOrder" xml:"ExecutionOrder"`
	// The field values, according to the field type
	Fields map[string]interface{} `form:"fields" json:"fields" xml:"fields"`
	// unique id per installation
	ID            string                 `form:"id" json:"id" xml:"id"`
	Relationships *WorkItemRelationships `form:"relationships" json:"relationships" xml:"relationships"`
	// ID of the type of this work item
	Type uuid.UUID `form:"type" json:"type" xml:"type"`
	// Version for optimistic concurrency control
	Version int `form:"version" json:"version" xml:"version"`
}

// WorkItemRelationships relation ship with a Space
type WorkItemRelationships struct {
	SpaceID uuid.UUID
}
