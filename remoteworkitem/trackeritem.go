package remoteworkitem

import (
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	uuid "github.com/satori/go.uuid"
)

// TrackerItem represents a remote tracker item
// Staging area before pushing to work item
type TrackerItem struct {
	gormsupport.Lifecycle
	ID uint64 `gorm:"primary_key"`
	// Remote item ID - unique across multiple trackers
	RemoteItemID string `gorm:"not null;unique"`
	// the field values
	Item string
	// FK to tracker
	TrackerID uuid.UUID `gorm:"ForeignKey:Tracker"`
}
