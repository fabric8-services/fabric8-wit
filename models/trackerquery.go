package models

// TrackerQuery represents tracker query
type TrackerQuery struct {
	ID uint64 `gorm:"primary_key"`
	// Search query of the tracker
	Query string
	// Schedule to fetch and import remote tracker items
	Schedule string
	// Version for optimistic concurrency control
	Version int
	// Tracker is a foreign key for a issue tracker
	Tracker Tracker `gorm:"ForeignKey:TrackerRefer"`

	TrackerRefer int
}
