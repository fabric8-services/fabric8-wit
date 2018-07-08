package workitem

import (
	"sort"
	"errors"
	"reflect"
	"time"

	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/convert"

	uuid "github.com/satori/go.uuid"
)

// WorkItem the model structure for the work item.
type WorkItem struct {
	// unique id per installation (used for references at the DB level)
	ID uuid.UUID
	// unique number per _space_
	Number int
	// ID of the type of this work item
	Type uuid.UUID
	// Version for optimistic concurrency control
	Version int
	// ID of the space to which this work item belongs
	SpaceID uuid.UUID
	// The field values, according to the field type
	Fields map[string]interface{}
	// optional, private timestamp of the latest addition/removal of a relationship with this workitem
	// this field is used to generate the `ETag` and `Last-Modified` values in the HTTP responses and conditional requests processing
	relationShipsChangedAt *time.Time
}

// WICountsPerIteration counting work item states by iteration
type WICountsPerIteration struct {
	IterationID string `gorm:"column:iterationid"`
	Total       int
	Closed      int
}

// GetETagData returns the field values to use to generate the ETag
func (wi WorkItem) GetETagData() []interface{} {
	return []interface{}{wi.ID, wi.Version, wi.relationShipsChangedAt}
}

// GetLastModified returns the last modification time
func (wi WorkItem) GetLastModified() time.Time {
	var lastModified *time.Time // default value
	if updatedAt, ok := wi.Fields[SystemUpdatedAt].(time.Time); ok {
		lastModified = &updatedAt
	}
	// also check the optional 'relationShipsChangedAt' field
	if wi.relationShipsChangedAt != nil && (lastModified == nil || wi.relationShipsChangedAt.After(*lastModified)) {
		lastModified = wi.relationShipsChangedAt
	}

	log.Debug(nil, map[string]interface{}{"wi_id": wi.ID}, "Last modified value: %v", lastModified)
	return *lastModified
}

// ChangeSet derives a changeset between this workitem and a given workitem.
func (wi WorkItem) ChangeSet(older convert.ChangeDetector) ([]convert.Change, error) {
	olderWorkItem, ok := older.(WorkItem)
	if !ok {
		return nil, errors.New("Other entity is not a WorkItem: " + reflect.TypeOf(older).String())
	}
	if wi.ID != olderWorkItem.ID {
		return nil, errors.New("Other entity has not the same ID: " + olderWorkItem.ID.String())
	}
	changes := []convert.Change{}
	// CAUTION: we're only supporting changes to the system.state and to the
	// board position relationship for now. If we need to support more
	// attribute changes, this has to be added here. This will be likely
	// necessary when adding new Actions.
	// compare system.state
	if wi.Fields["system.state"] != olderWorkItem.Fields["system.state"] {
		changes = append(changes, convert.Change{
			AttributeName: "system.state",
			NewValue:      wi.Fields["system.state"],
			OldValue:      olderWorkItem.Fields["system.state"],
		})
	}
	// compare system.boardcolumns
	// this field looks like this:
	// system.boardcolumns": ["43f9e838-3b4b-45e8-85eb-dd402e8324b5", "69699af8-cb28-4b90-b829-24c1aad12797"]
	if wi.Fields["system.boardcolumns"] == nil && olderWorkItem.Fields["system.boardcolumns"] == nil {
		return changes, nil
	}
	if wi.Fields["system.boardcolumns"] == nil || olderWorkItem.Fields["system.boardcolumns"] == nil {
		changes = append(changes, convert.Change{
			AttributeName: "system.boardcolumns",
			NewValue:      wi.Fields["system.state"],
			OldValue:      olderWorkItem.Fields["system.state"],
		})
		return changes, nil
	}
	bcThis, ok := wi.Fields["system.boardcolumns"].([]string)
	bcOlder, ok := olderWorkItem.Fields["system.boardcolumns"].([]string)
	if len(bcThis) != len(bcOlder) {
		changes = append(changes, convert.Change{
			AttributeName: "system.boardcolumns",
			NewValue:      wi.Fields["system.state"],
			OldValue:      olderWorkItem.Fields["system.state"],
		})
		return changes, nil
	}
	thisCopy := make([]string, len(bcThis))
	olderCopy := make([]string, len(bcOlder))
	copy(thisCopy, bcThis)
	copy(olderCopy, bcOlder)
	sort.Strings(thisCopy)
	sort.Strings(olderCopy)
	if !reflect.DeepEqual(thisCopy, olderCopy) {
		changes = append(changes, convert.Change{
			AttributeName: "system.boardcolumns",
			NewValue:      wi.Fields["system.state"],
			OldValue:      olderWorkItem.Fields["system.state"],
		})
		return changes, nil
	}
	return changes, nil
}
