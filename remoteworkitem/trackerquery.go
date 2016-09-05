package remoteworkitem

import "github.com/almighty/almighty-core/models"

// TrackerQuery represents tracker query
type TrackerQuery struct {
	models.Lifecycle
	ID uint64 `gorm:"primary_key"`
	// Search query of the tracker
	Query string
	// Schedule to fetch and import remote tracker items
	Schedule string
	// TrackerID is a foreign key for a tracker
	TrackerID uint64 `gorm:"ForeignKey:Tracker"`
}
