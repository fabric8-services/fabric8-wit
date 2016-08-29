package models

// TrackerItem represents a remote tracker item
// Staging area before pushing to work item
type TrackerItem struct {
	ID uint64 `gorm:"primary_key"`
	// the field values
	Item string
	// Batch ID for earch running of tracker query (UUID V4)
	BatchID string
	// FK to trackey query
	TrackerQuery uint64 `gorm:"ForeignKey:TrackerQuery"`
}
