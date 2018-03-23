package event

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// WorkItemEvent represents work item event
type WorkItemEvent struct {
	ID        uuid.UUID
	Name      string
	Timestamp time.Time
	Modifier  uuid.UUID
	Old       string
	New       string
}
