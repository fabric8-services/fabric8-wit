package event

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// Event represents work item event
type Event struct {
	RevisionID     uuid.UUID
	Name           string
	WorkItemTypeID uuid.UUID
	Timestamp      time.Time
	Modifier       uuid.UUID
	Old            interface{}
	New            interface{}
}

// GetETagData returns the field values to use to generate the ETag
func (e Event) GetETagData() []interface{} {
	return []interface{}{e.RevisionID, e.Name}
}

// GetLastModified returns the last modification time
func (e Event) GetLastModified() time.Time {
	return e.Timestamp.Truncate(time.Second)
}
