package models

// TrackerQuery represents tracker query
type TrackerQuery struct {
	ID uint64 `gorm:"primary_key"`
	// Search query of the tracker
	Query string
	// Schedule to fetch and import remote tracker items
	Schedule string
	// Tracker is a foreign key for a tracker
	Tracker int `gorm:"ForeignKey:Tracker"`
}
