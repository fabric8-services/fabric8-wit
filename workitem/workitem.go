package workitem

import uuid "github.com/satori/go.uuid"

// WorkItem the model structure for the work item.
type WorkItem struct {
	// unique id per installation
	ID string
	// ID of the type of this work item
	Type uuid.UUID
	// Version for optimistic concurrency control
	Version int
	// ID of the space to which this work item belongs
	SpaceID uuid.UUID
	// The field values, according to the field type
	Fields map[string]interface{}
}

type WICountsPerIteration struct {
	IterationId string `gorm:"column:iterationid"`
	Total       int
	Closed      int
}
