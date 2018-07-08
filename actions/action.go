package actions

import (
	"github.com/fabric8-services/fabric8-wit/convert"
	"errors"
)

// Action defines an action on change of an entity. Executing an
// Action might have sideffects, but will always return the original
// given context with all changes of the Action to this context. Note
// that the execution may have sideeffects on other entities beyond the
// context.
type Action interface {
	// byChangeset executes this action by looking at a change set of
	// updated attributes. It returns the new context. Note that this
	// needs the new (after change) context and the old value(s) as
	// part of the changeset.
	onChange(newContext convert.ChangeDetector, contextChanges []convert.Change, configuration string, actionChanges *[]convert.Change) (convert.ChangeDetector, []convert.Change, error)
}

// ExecuteActionsByOldNew executes all actions given in the actionConfigList
// using the mapped configuration strings and returns the new context entity.
func ExecuteActionsByOldNew(oldContext convert.ChangeDetector, newContext convert.ChangeDetector, actionConfigList map[string]string) (convert.ChangeDetector, *[]convert.Change, error) {
	if oldContext == nil || newContext == nil {
		return nil, nil, errors.New("Execute actions called with nil entities")
	}
	contextChanges, err := oldContext.ChangeSet(newContext)
	if err != nil {
		return nil, nil, err
	}
	return ExecuteActionsByChangeset(newContext, contextChanges, actionConfigList)
}

// ExecuteActionsByChangeset executes all actions given in the actionConfigs
// using the mapped configuration strings and returns the new context entity.
func ExecuteActionsByChangeset(newContext convert.ChangeDetector, contextChanges []convert.Change, actionConfigs map[string]string) (convert.ChangeDetector, *[]convert.Change, error) {
	var actionChanges *[]convert.Change
	for actionKey := range actionConfigs {
		actionConfig := actionConfigs[actionKey]
		switch actionKey {
		case "StateToBoardposition":
			executeAction(new(ActionStateToMetaState), actionConfig, newContext, contextChanges, actionChanges)
		default:
			return nil, nil, errors.New("Action key " + actionKey + " is unknown")
		}
	}
	return newContext, actionChanges, nil
}

// executeAction executes the action given. The actionChanges contain the changes made by
// prior action executions. The execution is expected to add/update their changes on this
// change set.
func executeAction(action Action, configuration string, newContext convert.ChangeDetector, contextChanges []convert.Change, actionChanges *[]convert.Change) (convert.ChangeDetector, []convert.Change, error) {
	return action.onChange(newContext, contextChanges, configuration, actionChanges)
}
