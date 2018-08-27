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

// A List is an array of events with the possibility to filter events for
// a given revision through the FilterByRevision function.
type List []Event

// FilterByRevisionID returns a new list with just the events inside that match
// the given revision ID.
func (l List) FilterByRevisionID(revisionID uuid.UUID) List {
	res := List{}
	for _, x := range l {
		if x.RevisionID == revisionID {
			res = append(res, x)
		}
	}
	return res
}
