package remoteworkitem

import (
	"github.com/fabric8-services/fabric8-wit/gormsupport"

	uuid "github.com/satori/go.uuid"
)

// TrackerQuery represents tracker query
type TrackerQuery struct {
	gormsupport.Lifecycle
	ID uuid.UUID `sql:"type:uuid" gorm:"primary_key"`
	// Search query of the tracker
	Query string
	// Schedule to fetch and import remote tracker items
	Schedule string
	// TrackerID is a foreign key for a tracker
	TrackerID uuid.UUID `gorm:"ForeignKey:TrackerID"`
	// SpaceID is a foreign key for a space
	SpaceID        uuid.UUID `gorm:"ForeignKey:SpaceID"`
	WorkItemTypeID uuid.UUID `gorm:"ForeignKey:WorkItemTypeID"`
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (t TrackerQuery) TableName() string {
	return trackerQueriesTableName
}
