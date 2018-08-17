package workitem

import (
	"reflect"
	"sort"
	"time"

	"github.com/fabric8-services/fabric8-wit/actions/change"
	"github.com/fabric8-services/fabric8-wit/log"

	errs "github.com/pkg/errors"
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
func (wi WorkItem) ChangeSet(older change.Detector) (change.Set, error) {
	if older == nil {
		// this is changeset for a new ChangeDetector, report all observed attributes to
		// the change set. This needs extension once we support more attributes.
		changeSet := change.Set{
			{
				AttributeName: SystemState,
				NewValue:      wi.Fields[SystemState],
				OldValue:      nil,
			},
		}
		if wi.Fields[SystemBoardcolumns] != nil && len(wi.Fields[SystemBoardcolumns].([]interface{})) != 0 {
			changeSet = append(changeSet, change.Change{
				AttributeName: SystemBoardcolumns,
				NewValue:      wi.Fields[SystemBoardcolumns],
				OldValue:      nil,
			})
		}
		return changeSet, nil
	}
	olderWorkItem, ok := older.(WorkItem)
	if !ok {
		return nil, errs.New("Other entity is not a WorkItem: " + reflect.TypeOf(older).String())
	}
	if wi.ID != olderWorkItem.ID {
		return nil, errs.New("Other entity has not the same ID: " + olderWorkItem.ID.String())
	}
	changes := []change.Change{}
	// CAUTION: we're only supporting changes to the system.state and to the
	// board position relationship for now. If we need to support more
	// attribute changes, this has to be added here. This will be likely
	// necessary when adding new Actions.
	// compare system.state
	if wi.Fields[SystemState] != olderWorkItem.Fields[SystemState] {
		changes = append(changes, change.Change{
			AttributeName: SystemState,
			NewValue:      wi.Fields[SystemState],
			OldValue:      olderWorkItem.Fields[SystemState],
		})
	}
	// compare system.boardcolumns
	// this field looks like this:
	// system.boardcolumns": ["43f9e838-3b4b-45e8-85eb-dd402e8324b5", "69699af8-cb28-4b90-b829-24c1aad12797"]
	if wi.Fields[SystemBoardcolumns] == nil && olderWorkItem.Fields[SystemBoardcolumns] == nil {
		return changes, nil
	}
	if wi.Fields[SystemBoardcolumns] == nil || olderWorkItem.Fields[SystemBoardcolumns] == nil {
		changes = append(changes, change.Change{
			AttributeName: SystemBoardcolumns,
			NewValue:      wi.Fields[SystemBoardcolumns],
			OldValue:      olderWorkItem.Fields[SystemBoardcolumns],
		})
		return changes, nil
	}
	if len(wi.Fields[SystemBoardcolumns].([]interface{})) == 0 || len(olderWorkItem.Fields[SystemBoardcolumns].([]interface{})) == 0 {
		if len(wi.Fields[SystemBoardcolumns].([]interface{})) == 0 && len(olderWorkItem.Fields[SystemBoardcolumns].([]interface{})) == 0 {
			// both lists are empty, return no change.
			return changes, nil
		}
		// one of the lists is empty, do return a change.
		changes = append(changes, change.Change{
			AttributeName: SystemBoardcolumns,
			NewValue:      wi.Fields[SystemBoardcolumns],
			OldValue:      olderWorkItem.Fields[SystemBoardcolumns],
		})
		return changes, nil
	}
	bcThis, ok1 := wi.Fields[SystemBoardcolumns].([]interface{})
	bcOlder, ok2 := olderWorkItem.Fields[SystemBoardcolumns].([]interface{})
	if !ok1 || !ok2 {
		return nil, errs.New("Boardcolumn slice is not a interface{} slice")
	}
	if len(bcThis) != len(bcOlder) {
		changes = append(changes, change.Change{
			AttributeName: SystemBoardcolumns,
			NewValue:      wi.Fields[SystemBoardcolumns],
			OldValue:      olderWorkItem.Fields[SystemBoardcolumns],
		})
		return changes, nil
	}
	// because of the handing of interface{}, we need to do manual conversion here.
	thisCopyStr := make([]string, len(bcThis))
	for i := range bcThis {
		thisCopyStr[i], ok = bcThis[i].(string)
		if !ok {
			return nil, errs.New("Boardcolumn slice values are not of type string")
		}
	}
	olderCopyStr := make([]string, len(bcOlder))
	for i := range bcOlder {
		olderCopyStr[i], ok = bcOlder[i].(string)
		if !ok {
			return nil, errs.New("Boardcolumn slice values are not of type string")
		}
	}
	sort.Strings(thisCopyStr)
	sort.Strings(olderCopyStr)
	if !reflect.DeepEqual(thisCopyStr, olderCopyStr) {
		changes = append(changes, change.Change{
			AttributeName: SystemBoardcolumns,
			NewValue:      wi.Fields[SystemBoardcolumns],
			OldValue:      olderWorkItem.Fields[SystemBoardcolumns],
		})
		return changes, nil
	}
	return changes, nil
}
