package rules

import (
	"encoding/json"
	"context"
	"errors"

	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/convert"
)

type ActionStateToMetaState struct {
	Db application.DB
	Ctx context.Context
	UserID *uuid.UUID
}

// make sure the rule is implementing the interface.
var _ Action = ActionStateToMetaState{}

func (act ActionStateToMetaState) contains(s []uuid.UUID, e uuid.UUID) bool {
	for _, a := range s {
			if a == e {
					return true
			}
	}
	return false
}

func (act ActionStateToMetaState) removeElement(s []uuid.UUID, e uuid.UUID) []uuid.UUID {
	for idx, a := range s {
			if a == e {
				s = append(s[:idx], s[idx+1:]...)
				// we don't return here as there may be multiple copies of e in s.
			}
	}
	return s
}

func (act ActionStateToMetaState) difference(old []uuid.UUID, new []uuid.UUID) ([]uuid.UUID, []uuid.UUID) {
	var added []uuid.UUID
	var removed []uuid.UUID
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
	if act.Ctx == nil {
		return nil, errors.New("Context is nil")
	}
	if act.Db == nil {
		return nil, errors.New("Database is nil")
	}
	space, err := act.Db.Spaces().Load(act.Ctx, spaceID)
	if err != nil {
		return nil, errors.New("Error loading space: " + err.Error())
	}
	boards, err := act.Db.Boards().List(act.Ctx, space.SpaceTemplateID)
	if err != nil {
		return nil, errors.New("Error loading work item type: " + err.Error())
	}
	return boards, nil
}

func (act ActionStateToMetaState) loadWorkItemTypeGroupsBySpaceID(spaceID uuid.UUID) ([]*workitem.WorkItemTypeGroup, error) {
	if act.Ctx == nil {
		return nil, errors.New("Context is nil")
	}
	if act.Db == nil {
		return nil, errors.New("Database is nil")
	}
	space, err := act.Db.Spaces().Load(act.Ctx, spaceID)
	if err != nil {
		return nil, errors.New("Error loading space: " + err.Error())
	}
	groups, err := act.Db.WorkItemTypeGroups().List(act.Ctx, space.SpaceTemplateID)
	if err != nil {
		return nil, errors.New("Error loading work item type: " + err.Error())
	}
	return groups, nil
}

func (act ActionStateToMetaState) loadWorkItemTypeByID(id uuid.UUID) (*workitem.WorkItemType, error) {
	if act.Ctx == nil {
		return nil, errors.New("Context is nil")
	}
	if act.Db == nil {
		return nil, errors.New("Database is nil")
	}
	wit, err := act.Db.WorkItemTypes().Load(act.Ctx, id)
	if err != nil {
		return nil, errors.New("Error loading work item type: " + err.Error())
	}
	return wit, nil
}

func (act ActionStateToMetaState) loadWorkItemByID(id uuid.UUID) (*workitem.WorkItem, error) {
	if act.Ctx == nil {
		return nil, errors.New("Context is nil")
	}
	if act.Db == nil {
		return nil, errors.New("Database is nil")
	}
	wi, err := act.Db.WorkItems().LoadByID(act.Ctx, id)
	if err != nil {
		return nil, err
	}
	return wi, nil
}

func (act ActionStateToMetaState) storeWorkItem(workitem *workitem.WorkItem) (*workitem.WorkItem, error) {
	if act.Ctx == nil {
		return nil, errors.New("Context is nil")
	}
	if act.Db == nil {
		return nil, errors.New("Database is nil")
	}
	if act.UserID == nil {
		return nil, errors.New("UserID is nil")
	}
	err := application.Transactional(act.Db, func(appl application.Application) error {
		var err error
		workitem, err = appl.WorkItems().Save(act.Ctx, workitem.SpaceID, *workitem, *act.UserID)
		if err != nil {
			return errors.New("Error updating work item")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return workitem, nil
}

func (act ActionStateToMetaState) getValueListFromFieldType(wit *workitem.WorkItemType, fieldName string) ([]interface{}, error) {
	fieldType := wit.Fields[fieldName].Type
	switch t := fieldType.(type) {
	case workitem.EnumType:
		return t.Values, nil
	} 
	return nil, errors.New("Given field on workitemtype " + wit.ID.String() + " is not an enum field: " + fieldName)
}

func (act ActionStateToMetaState) getStateToMetastateMap(workitemTypeID uuid.UUID) (map[string]string, error) {
	wit, err := act.loadWorkItemTypeByID(workitemTypeID)
	if err != nil {
		return nil, err
	}
	stateList, err := act.getValueListFromFieldType(wit, workitem.SystemState)
	if err != nil {
		return nil, err
	}
	metastateList, err := act.getValueListFromFieldType(wit, workitem.SystemMetaState)
	if err != nil {
		return nil, err
	}
	stateToMetastateMap := make(map[string]string)
	for idx := range stateList {
		thisState, ok := stateList[idx].(string)
		if !ok {
			return nil, errors.New("State value in value list is not of type string")
		}
		thisMetastate, ok := metastateList[idx].(string)
		if !ok {
			return nil, errors.New("Metastate value in value list is not of type string")
		}
		stateToMetastateMap[thisState] = thisMetastate	
	}
	return stateToMetastateMap, nil
}

func (act ActionStateToMetaState) getMetastateToStateMap(workitemTypeID uuid.UUID) (map[string]string, error) {
	stateToMetastate, err := act.getStateToMetastateMap(workitemTypeID)
	if err != nil {
		return nil, err
	}
	metastateToStateMap := make(map[string]string)
	for state, metastate := range stateToMetastate {
		metastateToStateMap[metastate] = state
	}
	return metastateToStateMap, nil
}

func (act ActionStateToMetaState) addOrUpdateChange(changes *[]convert.Change, attributeName string, oldValue interface{}, newValue interface{}) []convert.Change {
	for _, change := range *changes {
		if (change.AttributeName == attributeName) {
			change.NewValue = newValue
			return *changes
		}
	}
	newChanges := append(*changes, convert.Change {
		AttributeName: attributeName,
		OldValue: oldValue,
		NewValue: newValue,
	})
	return newChanges
}

// OnChange executes the action rule.
func (act ActionStateToMetaState) OnChange(newContext convert.ChangeDetector, contextChanges []convert.Change, configuration string, actionChanges *[]convert.Change) (convert.ChangeDetector, []convert.Change, error) {
	if len(contextChanges) == 0 {
		// no changes, just return what we have.
		return newContext, *actionChanges, nil
	}
	for _, change := range contextChanges {
		if (change.AttributeName == workitem.SystemState) {
			return act.OnStateChange(newContext, contextChanges, configuration, actionChanges)
		}
		if (change.AttributeName == workitem.SystemBoardcolumns) {
			return act.OnBoardColumnsChange(newContext, contextChanges, configuration, actionChanges)
		}
	}
	// no changes that match this rule.
	return newContext, *actionChanges, nil
}

// OnBoardColumnsChange is executed when the columns change. It eventually updates the metastate and the state.
func (act ActionStateToMetaState) OnBoardColumnsChange(newContext convert.ChangeDetector, contextChanges []convert.Change, configuration string, actionChanges *[]convert.Change) (convert.ChangeDetector, []convert.Change, error) {
	// we already assume that the rule applies, this needs to be checked in the controller.
	// there is no additional check on the rule key.
	wi, ok := newContext.(workitem.WorkItem)
	if !ok {
		return nil, nil, errors.New("Given context is not a WorkItem instance")
	}
	// extract columns that changed from oldValue newValue
	var columnsAdded []uuid.UUID
	for _, change := range contextChanges {
		if change.AttributeName == workitem.SystemBoardcolumns {
			columnsAdded, _ = act.difference(change.OldValue.([]uuid.UUID), change.NewValue.([]uuid.UUID))
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
	var changes []convert.Change
	// go over all added columns.
	for _, columnID := range columnsAdded {
		// TODO WHAT HAPPENS IF MULTIPLE COLUMNS ARE IN HERE?
		// get board and column by columnID.
		var thisColumn workitem.BoardColumn
		boards, err := act.loadWorkItemBoardsBySpaceID(wi.SpaceID)
		if err != nil {
			return nil, nil, err
		}
		for _, board := range boards {
			for _, column := range board.Columns {
				if columnID == column.ID {
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
			changes = act.addOrUpdateChange(actionChanges, workitem.SystemMetaState, wi.Fields[workitem.SystemMetaState], metaState)
			wi.Fields[workitem.SystemMetaState] = metaState
			wiDirty = true
			// next, check if the state needs to change as well from the metastate.
			if wi.Fields[workitem.SystemState] != mapping[metaState] {
				// yes, the state changes as well.
				changes = act.addOrUpdateChange(actionChanges, workitem.SystemState, wi.Fields[workitem.SystemState], mapping[metaState])
				wi.Fields[workitem.SystemState] = mapping[metaState]
			}
		}
	}
	// finally, store the new work item state if something changed.
	if wiDirty {
		act.storeWorkItem(&wi)
	}
	// return to sender
	return newContext, changes, nil
}

// OnStateChange is executed when the state changes. It eventually updates the metastate and the boardcolumns.
func (act ActionStateToMetaState) OnStateChange(newContext convert.ChangeDetector, contextChanges []convert.Change, configuration string, actionChanges *[]convert.Change) (convert.ChangeDetector, []convert.Change, error) {
	wi, ok := newContext.(workitem.WorkItem)
	if !ok {
		return nil, nil, errors.New("Given context is not a WorkItem instance")
	}
	// get the mapping.
	mapping, err := act.getStateToMetastateMap(wi.Type)
	if err != nil {
		return nil, nil, err
	}
	// create a dirty flag. Note that we can not use len(actionChanges) as this
	// may contain previous changes from the action chain.
	wiDirty := false
	// update the workitem accordingly.
	if (wi.Fields[workitem.SystemMetaState] == mapping[workitem.SystemState]) {
		// metastate remains stable, nothing to do.
		return newContext, *actionChanges, nil
	}
	// otherwise, update the metastate from the state.
	changes := act.addOrUpdateChange(actionChanges, workitem.SystemMetaState, wi.Fields[workitem.SystemMetaState], mapping[workitem.SystemState])
	wi.Fields[workitem.SystemMetaState] = mapping[workitem.SystemState]
	wiDirty = true
	// next, get the columns of the workitem and see if these needs to be updated.
	boards, err := act.loadWorkItemBoardsBySpaceID(wi.SpaceID)
	if err != nil {
		return nil, nil, err
	}
	// next, check which boards are relevant for this WI.
	groups, err := act.loadWorkItemTypeGroupsBySpaceID(wi.SpaceID)
	if err != nil {
		return nil, nil, err
	}
	var relevantBoards []*workitem.Board
	for _, board := range(boards) {
		// this rule is only dealing with TypeLevelContext boards right now
		// this may need to be extended when we allow other boards.
		if board.ContextType == "TypeLevelContext" {
			// now check if the type level in the Context includes the current WIs type
			thisBoardContext, err := uuid.FromString(board.Context)
			if err != nil {
				return nil, nil, err
			}
			for _, group := range(groups) {
				if group.ID == thisBoardContext && act.contains(group.TypeList, wi.Type) {
					// this board is relevant.
					relevantBoards = append(relevantBoards, board)
				}
			}
		} 
	}
	// next, iterate over all relevant boards, checking their rule config
	// and update the WI position accordingly.
	oldColumnsConfig := make([]uuid.UUID, len(wi.Fields[workitem.SystemBoardcolumns].([]uuid.UUID)))
	columnsChanged := false
	copy(oldColumnsConfig, wi.Fields[workitem.SystemBoardcolumns].([]uuid.UUID))
	for _, board := range(relevantBoards) {
		for _, column := range(board.Columns) {
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
					if metaState == wi.Fields[workitem.SystemMetaState] {
						// the column config matches the *new* metastate, so the WI needs to 
						// appear in this column.
						wi.Fields[workitem.SystemBoardcolumns] = append(wi.Fields[workitem.SystemBoardcolumns].([]uuid.UUID), column.ID)
						columnsChanged = true
					} else {
						// the column *does not* match the *new* metastate, so the column has to
						// be removed from the WIs columns. Note that we can't just remove all
						// entries in wi.Fields[workitem.SystemBoardcolumns] as there may be
						// other columns from non-relevant boards in it that need to be left
						// untouched.
						wi.Fields[workitem.SystemBoardcolumns] = act.removeElement(wi.Fields[workitem.SystemBoardcolumns].([]uuid.UUID), column.ID)
						columnsChanged = true
					}
				} else {
					return nil, nil, errors.New("Invalid configuration for transRuleKey '" + ActionKeyStateToMetastate + "': " + columnRuleConfig)
				}				
			}
		}
	}
	// if the column set has changed, create an entry for the change set.
	if columnsChanged {
		changes = act.addOrUpdateChange(actionChanges, workitem.SystemBoardcolumns, oldColumnsConfig, wi.Fields[workitem.SystemBoardcolumns])
	}
	// finally, store the new work item state if something changed.
	if wiDirty {
		act.storeWorkItem(&wi)
	}
	// and return to sender.
	return wi, changes, nil
}
