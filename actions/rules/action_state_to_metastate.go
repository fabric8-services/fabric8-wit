package rules

import (
	"context"
	"encoding/json"
	"github.com/fabric8-services/fabric8-wit/actions/change"
	"github.com/fabric8-services/fabric8-wit/id"

	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/workitem"
)

// ActionStateToMetaState implements the bidirectional mapping between state and column.
type ActionStateToMetaState struct {
	Db     application.DB
	Ctx    context.Context
	UserID *uuid.UUID
}

// make sure the rule is implementing the interface.
var _ Action = ActionStateToMetaState{}

func (act ActionStateToMetaState) contains(s []interface{}, e interface{}) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (act ActionStateToMetaState) removeElement(s []interface{}, e interface{}) []interface{} {
	for idx, a := range s {
		if a == e {
			copy(s[idx:], s[idx+1:])
			s[len(s)-1] = nil
			s = s[:len(s)-1]
			// we don't return here as there may be multiple copies of e in s.
		}
	}
	return s
}

// difference returns the differences between two slices. It returns two slices containing
// the added and removed elements compared from old and new arguments.
func (act ActionStateToMetaState) difference(old []interface{}, new []interface{}) ([]interface{}, []interface{}) {
	var added []interface{}
	var removed []interface{}
	// added in slice2
	for _, elem := range new {
		if !act.contains(old, elem) {
			added = append(added, elem)
		}
	}
	// removed in slice2
	for _, elem := range old {
		if !act.contains(new, elem) {
			removed = append(removed, elem)
		}
	}
	return added, removed
}

func (act ActionStateToMetaState) loadWorkItemBoardsBySpaceID(spaceID uuid.UUID) ([]*workitem.Board, error) {
	space, err := act.Db.Spaces().Load(act.Ctx, spaceID)
	if err != nil {
		return nil, errs.Wrap(err, "error loading space: "+err.Error())
	}
	boards, err := act.Db.Boards().List(act.Ctx, space.SpaceTemplateID)
	if err != nil {
		return nil, errs.Wrap(err, "error loading work item type: "+err.Error())
	}
	return boards, nil
}

func (act ActionStateToMetaState) loadWorkItemByID(id uuid.UUID) (*workitem.WorkItem, error) {
	wi, err := act.Db.WorkItems().LoadByID(act.Ctx, id)
	if err != nil {
		return nil, errs.Wrap(err, "error loading work item: "+err.Error())
	}
	return wi, nil
}

func (act ActionStateToMetaState) storeWorkItem(workitem *workitem.WorkItem) (*workitem.WorkItem, error) {
	err := application.Transactional(act.Db, func(appl application.Application) error {
		var err error
		workitem, _, err = appl.WorkItems().Save(act.Ctx, workitem.SpaceID, *workitem, *act.UserID)
		if err != nil {
			return errs.Wrap(err, "error updating work item")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return workitem, nil
}

func (act ActionStateToMetaState) getValueListFromEnumField(wit *workitem.WorkItemType, fieldName string) ([]interface{}, error) {
	enumFieldType := wit.Fields[fieldName].Type
	switch t := enumFieldType.(type) {
	case workitem.EnumType:
		return t.Values, nil
	}
	return nil, errs.New("given field on workitemtype " + wit.ID.String() + " is not an enum field: " + fieldName)
}

// getStateToMetastateMap returns the mapping from state to metastate values read from the template.
func (act ActionStateToMetaState) getStateToMetastateMap(workitemTypeID uuid.UUID) (map[string]string, error) {
	wit, err := act.Db.WorkItemTypes().Load(act.Ctx, workitemTypeID)
	if err != nil {
		return nil, err
	}
	stateList, err := act.getValueListFromEnumField(wit, workitem.SystemState)
	if err != nil {
		return nil, err
	}
	metastateList, err := act.getValueListFromEnumField(wit, workitem.SystemMetaState)
	if err != nil {
		return nil, err
	}
	if len(stateList) != len(metastateList) {
		return nil, errs.Errorf("inconsistent number of states and metatstates in the current work item type (must be equal): %d states != %d metastates", len(stateList), len(metastateList))
	}
	stateToMetastateMap := make(map[string]string)
	for idx := range stateList {
		thisState, ok := stateList[idx].(string)
		if !ok {
			return nil, errs.New("state value in value list is not of type string")
		}
		thisMetastate, ok := metastateList[idx].(string)
		if !ok {
			return nil, errs.New("metastate value in value list is not of type string")
		}
		// only the first state<->metatstate mapping needs to be taken as
		// per definition. So we're only adding a new entry here if there
		// is not an entry already existing.
		if _, ok := stateToMetastateMap[thisState]; !ok {
			stateToMetastateMap[thisState] = thisMetastate
		}
	}
	return stateToMetastateMap, nil
}

// getMetastateToStateMap returns the mapping from metastate to state values read from the template.
func (act ActionStateToMetaState) getMetastateToStateMap(workitemTypeID uuid.UUID) (map[string]string, error) {
	stateToMetastate, err := act.getStateToMetastateMap(workitemTypeID)
	if err != nil {
		return nil, err
	}
	metastateToStateMap := make(map[string]string)
	for state, metastate := range stateToMetastate {
		// this is important: we only add a new entry if there
		// is not an entry already existing. Therefore, we're only
		// using the first mapping, satisfying the req for getting the
		// first matching state for a metastate.
		if _, ok := metastateToStateMap[metastate]; !ok {
			metastateToStateMap[metastate] = state
		}
	}
	return metastateToStateMap, nil
}

func (act ActionStateToMetaState) addOrUpdateChange(changes change.Set, attributeName string, oldValue interface{}, newValue interface{}) change.Set {
	for _, change := range changes {
		if change.AttributeName == attributeName {
			change.NewValue = newValue
			return changes
		}
	}
	newChanges := append(changes, change.Change{
		AttributeName: attributeName,
		OldValue:      oldValue,
		NewValue:      newValue,
	})
	return newChanges
}

func (act ActionStateToMetaState) fuseChanges(c1 change.Set, c2 change.Set) change.Set {
	for _, change := range c2 {
		c1 = act.addOrUpdateChange(c1, change.AttributeName, change.OldValue, change.NewValue)
	}
	return c1
}

// OnChange executes the action rule. It implements rules.Action. Returns the Work Item with eventual changes applied.
func (act ActionStateToMetaState) OnChange(newContext change.Detector, contextChanges change.Set, configuration string, actionChanges *change.Set) (change.Detector, change.Set, error) {
	if act.Ctx == nil {
		return nil, nil, errs.New("context is nil")
	}
	if act.Db == nil {
		return nil, nil, errs.New("database is nil")
	}
	if act.UserID == nil {
		return nil, nil, errs.New("userID is nil")
	}
	if actionChanges == nil {
		return nil, nil, errs.New("given actionChanges is nil")
	}
	if len(contextChanges) == 0 {
		// no changes, just return what we have.
		return newContext, *actionChanges, nil
	}
	// if we have multiple changes, this iterates over them and cherrypicks, fusing the results together.
	var err error
	var executionChanges change.Set
	for _, change := range contextChanges {
		if change.AttributeName == workitem.SystemState {
			newContext, executionChanges, err = act.onStateChange(newContext, contextChanges, configuration, actionChanges)
		}
		if change.AttributeName == workitem.SystemBoardcolumns {
			newContext, executionChanges, err = act.onBoardColumnsChange(newContext, contextChanges, configuration, actionChanges)
		}
		if err != nil {
			return nil, nil, err
		}
		*actionChanges = act.fuseChanges(*actionChanges, executionChanges)
	}
	// return the result
	return newContext, *actionChanges, nil
}

// onBoardColumnsChange is executed when the columns change. It eventually updates the metastate and the state. Returns the Work Item with eventual changes applied.
func (act ActionStateToMetaState) onBoardColumnsChange(newContext change.Detector, contextChanges change.Set, configuration string, actionChanges *change.Set) (change.Detector, change.Set, error) {
	// we already assume that the rule applies, this needs to be checked in the controller.
	// there is no additional check on the rule key.
	wi, ok := newContext.(workitem.WorkItem)
	if !ok {
		return nil, nil, errs.New("given context is not a WorkItem instance")
	}
	// extract columns that changed from oldValue newValue
	var columnsAdded []interface{}
	for _, change := range contextChanges {
		if change.AttributeName == workitem.SystemBoardcolumns {
			var oldValue []interface{}
			oldValue, ok := change.OldValue.([]interface{})
			if !ok && change.OldValue != nil {
				// OldValue may be nil, so only throw error when non-nil and
				// can not convert.
				return nil, nil, errs.New("error converting oldValue set")
			}
			newValue, ok := change.NewValue.([]interface{})
			if !ok {
				return nil, nil, errs.New("error converting newValue set")
			}
			columnsAdded, _ = act.difference(oldValue, newValue)
		}
	}
	// from here on, we ignore the removed columns as a possible metastate transition is
	// only happening on *adding* a new column. Removing column does not trigger a change.
	if len(columnsAdded) == 0 {
		// somehow, no actual changes on the columns.
		return newContext, *actionChanges, nil
	}
	// get the mapping.
	mapping, err := act.getMetastateToStateMap(wi.Type)
	if err != nil {
		return nil, nil, err
	}
	// create a dirty flag. Note that we can not use len(actionChanges) as this
	// may contain previous changes from the action chain.
	wiDirty := false
	var changes change.Set
	// go over all added columns. We support multiple added columns at once.
	for _, columnID := range columnsAdded {
		// get board and column by columnID.
		var thisColumn workitem.BoardColumn
		boards, err := act.loadWorkItemBoardsBySpaceID(wi.SpaceID)
		if err != nil {
			return nil, nil, err
		}
		for _, board := range boards {
			for _, column := range board.Columns {
				if columnID == column.ID.String() {
					thisColumn = column
				}
			}
		}
		// at this point, we don't check if the board was
		// relevant (matching the type group) as the move into
		// the column has already happened. We just make sure
		// this rule applies to the column.
		if thisColumn.TransRuleKey != ActionKeyStateToMetastate {
			// this is a column that does not apply to the rule, we don't apply here.
			return newContext, *actionChanges, nil
		}
		// unmarshall the configuration.
		config := map[string]string{}
		err = json.Unmarshal([]byte(thisColumn.TransRuleArgument), &config)
		if err != nil {
			return nil, nil, err
		}
		// extract the column's metastate config.
		if metaState, ok := config[ActionKeyStateToMetastateConfigMetastate]; ok {
			if metaState == wi.Fields[workitem.SystemMetaState] {
				// the WIs metastate is already the same as the columns
				// metastate, so nothing to do.
				return newContext, *actionChanges, nil
			}
			// the metatstate changes, so set it on the WI.
			changes = act.addOrUpdateChange(changes, workitem.SystemMetaState, wi.Fields[workitem.SystemMetaState], metaState)
			wi.Fields[workitem.SystemMetaState] = metaState
			wiDirty = true
			// next, check if the state needs to change as well from the metastate.
			if wi.Fields[workitem.SystemState] != mapping[metaState] {
				// yes, the state changes as well.
				changes = act.addOrUpdateChange(changes, workitem.SystemState, wi.Fields[workitem.SystemState], mapping[metaState])
				wi.Fields[workitem.SystemState] = mapping[metaState]
			}
		}
	}
	// finally, store the new work item state if something changed.
	var resultingWorkItem workitem.WorkItem
	if wiDirty {
		result, err := act.storeWorkItem(&wi)
		resultingWorkItem = *result
		if err != nil {
			return nil, nil, err
		}
	}
	// return to sender
	return resultingWorkItem, changes, nil
}

// onStateChange is executed when the state changes. It eventually updates the metastate and the boardcolumns. Returns the Work Item with eventual changes applied.
func (act ActionStateToMetaState) onStateChange(newContext change.Detector, contextChanges change.Set, configuration string, actionChanges *change.Set) (change.Detector, change.Set, error) {
	wi, ok := newContext.(workitem.WorkItem)
	if !ok {
		return nil, nil, errs.New("given context is not a WorkItem instance")
	}
	// get the mapping.
	mapping, err := act.getStateToMetastateMap(wi.Type)
	if err != nil {
		return nil, nil, err
	}
	// update the workitem accordingly.
	wiState := wi.Fields[workitem.SystemState].(string)
	if wi.Fields[workitem.SystemMetaState] == mapping[wiState] {
		// metastate remains stable, nothing to do.
		return newContext, *actionChanges, nil
	}
	// otherwise, update the metastate from the state.
	changes := act.addOrUpdateChange(*actionChanges, workitem.SystemMetaState, wi.Fields[workitem.SystemMetaState], mapping[wiState])
	wi.Fields[workitem.SystemMetaState] = mapping[wiState]
	// next, get the columns of the workitem and see if these needs to be updated.
	boards, err := act.loadWorkItemBoardsBySpaceID(wi.SpaceID)
	if err != nil {
		return nil, nil, err
	}
	// next, check which boards are relevant for this WI.
	space, err := act.Db.Spaces().Load(act.Ctx, wi.SpaceID)
	if err != nil {
		return nil, nil, errs.Wrap(err, "error loading space")
	}
	groups, err := act.Db.WorkItemTypeGroups().List(act.Ctx, space.SpaceTemplateID)
	if err != nil {
		return nil, nil, errs.Wrap(err, "error loading type groups")
	}
	var relevantBoards []*workitem.Board
	for _, board := range boards {
		// this rule is only dealing with TypeLevelContext boards right now
		// this may need to be extended when we allow other boards.
		if board.ContextType == "TypeLevelContext" {
			// now check if the type level in the Context includes the current WIs type
			thisBoardContext, err := uuid.FromString(board.Context)
			if err != nil {
				return nil, nil, err
			}
			for _, group := range groups {
				var typeListSlice id.Slice = group.TypeList
				if group.ID == thisBoardContext && typeListSlice.Contains(wi.Type) {
					// this board is relevant.
					relevantBoards = append(relevantBoards, board)
				}
			}
		}
	}
	// next, iterate over all relevant boards, checking their rule config
	// and update the WI position accordingly.
	var systemBoardColumns []interface{}
	systemBoardColumns, ok = wi.Fields[workitem.SystemBoardcolumns].([]interface{})
	if !ok && wi.Fields[workitem.SystemBoardcolumns] != nil {
		// wi.Fields[workitem.SystemBoardcolumns] may be empty, so we do a fallback.
		return nil, nil, errs.New("type conversion failed for boardcolumns")
	}
	oldColumnsConfig := make([]interface{}, len(systemBoardColumns))
	copy(oldColumnsConfig, systemBoardColumns)
	columnsChanged := false
	for _, board := range relevantBoards {
		// each WI can only be in exactly one columns on each board.
		// we apply the first column that matches. If that happens, we
		// flip alreadyPlacedInColumn and all subsequent matching columns
		// are non-matching.
		alreadyPlacedInColumn := false
		for _, column := range board.Columns {
			columnRuleKey := column.TransRuleKey
			columnRuleConfig := column.TransRuleArgument
			if columnRuleKey == ActionKeyStateToMetastate {
				// unmarshall the configuration.
				config := map[string]string{}
				err := json.Unmarshal([]byte(columnRuleConfig), &config)
				if err != nil {
					return nil, nil, err
				}
				if metaState, ok := config[ActionKeyStateToMetastateConfigMetastate]; ok {
					if metaState == wi.Fields[workitem.SystemMetaState] && !alreadyPlacedInColumn {
						// the column config matches the *new* metastate, so the WI needs to
						// appear in this column.
						var currentSystemBoardColumn []interface{}
						currentSystemBoardColumn, ok = wi.Fields[workitem.SystemBoardcolumns].([]interface{})
						// flip alreadyPlacedInColumn, so this will be the new column. Other
						// matching columns are ignored on this board.
						alreadyPlacedInColumn = true
						if !ok && wi.Fields[workitem.SystemBoardcolumns] != nil {
							// again, wi.Fields[workitem.SystemBoardcolumns] may be empty, so we
							// only fail here when the conversion fails AND the slice is non nil.
							return nil, nil, errs.New("error converting SystemBoardcolumns set")
						}
						if !act.contains(currentSystemBoardColumn, column.ID.String()) {
							wi.Fields[workitem.SystemBoardcolumns] = append(currentSystemBoardColumn, column.ID.String())
						}
						columnsChanged = true
					} else {
						// the column *does not* match the *new* metastate, so the column has to
						// be removed from the WIs columns. Note that we can't just remove all
						// entries in wi.Fields[workitem.SystemBoardcolumns] as there may be
						// other columns from non-relevant boards in it that need to be left
						// untouched.
						var currentSystemBoardColumn []interface{}
						currentSystemBoardColumn, ok = wi.Fields[workitem.SystemBoardcolumns].([]interface{})
						if !ok && wi.Fields[workitem.SystemBoardcolumns] != nil {
							// again, wi.Fields[workitem.SystemBoardcolumns] may be empty, so we
							// only fail here when the conversion fails AND the slice is non nil.
							return nil, nil, errs.New("error converting SystemBoardcolumns set")
						}
						wi.Fields[workitem.SystemBoardcolumns] = act.removeElement(currentSystemBoardColumn, column.ID.String())
						columnsChanged = true
					}
				} else {
					return nil, nil, errs.New("invalid configuration for transRuleKey '" + ActionKeyStateToMetastate + "': " + columnRuleConfig)
				}
			}
		}
	}
	// if the column set has changed, create an entry for the change set.
	if columnsChanged {
		changes = act.addOrUpdateChange(changes, workitem.SystemBoardcolumns, oldColumnsConfig, wi.Fields[workitem.SystemBoardcolumns])
	}
	// finally, store the new work item state if something changed.
	result, err := act.storeWorkItem(&wi)
	if err != nil {
		return nil, nil, err
	}
	// and return to sender.
	return *result, changes, nil
}
