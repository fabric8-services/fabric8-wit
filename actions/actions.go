package actions

import (
	"context"
	"github.com/fabric8-services/fabric8-wit/application"
	"errors"

	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/actions/rules"
	"github.com/fabric8-services/fabric8-wit/convert"
)

// ExecuteActionsByOldNew executes all actions given in the actionConfigList
// using the mapped configuration strings and returns the new context entity.
func ExecuteActionsByOldNew(ctx context.Context, db application.DB, userID uuid.UUID, oldContext convert.ChangeDetector, newContext convert.ChangeDetector, actionConfigList map[string]string) (convert.ChangeDetector, []convert.Change, error) {
	if oldContext == nil || newContext == nil {
		return nil, nil, errors.New("Execute actions called with nil entities")
	}
	contextChanges, err := oldContext.ChangeSet(newContext)
	if err != nil {
		return nil, nil, err
	}
	return ExecuteActionsByChangeset(ctx, db, userID, newContext, contextChanges, actionConfigList)
}

// ExecuteActionsByChangeset executes all actions given in the actionConfigs
// using the mapped configuration strings and returns the new context entity.
func ExecuteActionsByChangeset(ctx context.Context, db application.DB, userID uuid.UUID, newContext convert.ChangeDetector, contextChanges []convert.Change, actionConfigs map[string]string) (convert.ChangeDetector, []convert.Change, error) {
	var actionChanges []convert.Change
	var err error
	for actionKey := range actionConfigs {
		actionConfig := actionConfigs[actionKey]
		switch actionKey {
		case rules.ActionKeyNil:
			newContext, actionChanges, err = executeAction(rules.ActionNil{}, actionConfig, newContext, contextChanges, &actionChanges)
		case rules.ActionKeyFieldSet:
			newContext, actionChanges, err = executeAction(rules.ActionFieldSet{
				Db:     db,
				Ctx:    ctx,
				UserID: &userID,
			}, actionConfig, newContext, contextChanges, &actionChanges)
		case rules.ActionKeyStateToMetastate:
			// TODO(michaelkleinhenz): get db, ctx, and user.
			newContext, actionChanges, err = executeAction(rules.ActionStateToMetaState {
				Db:     db,
				Ctx:    ctx,
				UserID: &userID,
			}, actionConfig, newContext, contextChanges, &actionChanges)
		default:
			return nil, nil, errors.New("Action key " + actionKey + " is unknown")
		}
		if err != nil {
			return nil, nil, err
		}
	}
	return newContext, actionChanges, nil
}

// executeAction executes the action given. The actionChanges contain the changes made by
// prior action executions. The execution is expected to add/update their changes on this
// change set.
func executeAction(act rules.Action, configuration string, newContext convert.ChangeDetector, contextChanges []convert.Change, actionChanges *[]convert.Change) (convert.ChangeDetector, []convert.Change, error) {
	return act.OnChange(newContext, contextChanges, configuration, actionChanges)
}
