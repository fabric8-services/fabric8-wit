package remoteworkitem

import (
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	uuid "github.com/satori/go.uuid"
)

// Tracker represents tracker configuration
type Tracker struct {
	gormsupport.Lifecycle
	ID uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	// URL of the tracker
	URL string
	// Type of the tracker (jira, github, bugzilla, trello etc.)
	Type string
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (t Tracker) TableName() string {
	return trackersTableName
}
