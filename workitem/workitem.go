package workitem

import (
	"time"

	"github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

// WorkItem the model structure for the work item.
type WorkItem struct {
	// unique id per installation
	ID string
	// ID of the type of this work item
	Type uuid.UUID
	// Version for optimistic concurrency control
	Version int
	// ID of the space to which this work item belongs
	SpaceID uuid.UUID
	// The field values, according to the field type
	Fields map[string]interface{}
}

// WICountsPerIteration counting work item states by iteration
type WICountsPerIteration struct {
	IterationID string `gorm:"column:iterationid"`
	Total       int
	Closed      int
}

// GetETagData returns the field values to use to generate the ETag
func (wi WorkItem) GetETagData() []interface{} {
	return []interface{}{wi.ID, wi.Version}
}

// GetLastModified returns the last modification time
func (wi WorkItem) GetLastModified() time.Time {
	if updatedAt, ok := wi.Fields[SystemUpdatedAt]; ok {
		switch updatedAt := updatedAt.(type) {
		case time.Time:
			return updatedAt
		default:
			logrus.Info("'system.update_at' field value is not a valid time for work item with ID=" + wi.ID)
		}
	}
	// fallback value if none/no valid data was found.
	return time.Now()
}
