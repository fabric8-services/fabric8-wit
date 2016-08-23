package models

// Tracker represents tracker configuration
type Tracker struct {
	ID uint64 `gorm:"primary_key"`
	// URL of the tracker
	URL string
	// Type of the tracker (jira, github, bugzilla, trello etc.)
	Type string
}
