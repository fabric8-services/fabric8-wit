package remoteworkitem

import "github.com/almighty/almighty-core/models"

// Tracker represents tracker configuration
type Tracker struct {
	models.Lifecycle
	ID uint64 `gorm:"primary_key"`
	// URL of the tracker
	URL string
	// Type of the tracker (jira, github, bugzilla, trello etc.)
	Type string
}
