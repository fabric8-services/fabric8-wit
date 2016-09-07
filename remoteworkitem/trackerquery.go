package remoteworkitem

import "github.com/almighty/almighty-core/models"

// TrackerQuery represents tracker query
type TrackerQuery struct {
	models.Lifecycle
	ID string `gorm:"primary_key"`
	// Search query of the tracker
	Query string
	// Schedule to fetch and import remote tracker items
	Schedule string
	// TrackerID is a foreign key for a tracker
	TrackerID string `gorm:"ForeignKey:Tracker"`
}
