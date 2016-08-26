package models

// TrackerItem represents a remote tracker item
// Staging area before pushing to work item
type TrackerItem struct {
	ID uint64 `gorm:"primary_key"`
	// the field values
	Fields string
	// FK to trackey query
	TrackerQuery TrackerQuery `gorm:"ForeignKey:TrackerQueryRefer"`

	TrackerQueryRefer int
}
