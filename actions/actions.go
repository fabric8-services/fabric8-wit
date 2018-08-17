package actions

/*
	 The actions system is a key component for process automation in WIT. It provides
	 a way of executing user-configurable, dynamic process steps depending on user
	 settings, schema settings and events in the WIT.

	 The idea here is to provide a simple, yet powerful "publish-subscribe" system that
	 can connect any "event" in the system to any "action" with a clear decoupling
	 of events and actions with the goal of making the associations later dynamic and
	 configurable by the user ("user connects this event to this action"). Think
	 of a "IFTTT for WIT" (https://en.wikipedia.org/wiki/IFTTT).

	 Actions are generic and atomic execution steps that do exactly one task and
	 are configurable. The actions system around the actions provide a key-based
	 execution of the actions.

	 Some examples for an application of this system would be:
		- closing all children of a parent WI that is being closed (the user connects the
		"close" attribute change event of a WI to an action that closes all WIs of
		a matching query).
		- sending out notifications for mentions on markdown (the system executes an
		action "send notification" for every mention found in markdown values).
		- moving all WIs from one iteration to the next in the time sequence when
		the original iteration is closed.

		For all these automations, the actions system provides a re-usable, flexible
		and later user configurable way of doing that without creating lots of
		custom code and/or custom process implementations that are hardcoded in the
		WIT.
*/

import (
	"context"

	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/actions/change"
	"github.com/fabric8-services/fabric8-wit/actions/rules"
	"github.com/fabric8-services/fabric8-wit/application"
)

// ExecuteActionsByOldNew executes all actions given in the actionConfigList
// using the mapped configuration strings and returns the new context entity.
// It takes the old version and the new version of the context entity, comparing them.
func ExecuteActionsByOldNew(ctx context.Context, db application.DB, userID uuid.UUID, oldContext change.Detector, newContext change.Detector, actionConfigList map[string]string) (change.Detector, change.Set, error) {
	if oldContext == nil || newContext == nil {
		return nil, nil, errs.New("execute actions called with nil entities")
	}
	contextChanges, err := oldContext.ChangeSet(newContext)
	if err != nil {
		return nil, nil, err
	}
	return ExecuteActionsByChangeset(ctx, db, userID, newContext, contextChanges, actionConfigList)
}

// ExecuteActionsByChangeset executes all actions given in the actionConfigs
// using the mapped configuration strings and returns the new context entity.
// It takes a []Change that describes the differences between the old and the new context.
func ExecuteActionsByChangeset(ctx context.Context, db application.DB, userID uuid.UUID, newContext change.Detector, contextChanges []change.Change, actionConfigs map[string]string) (change.Detector, change.Set, error) {
	var actionChanges change.Set
	for actionKey := range actionConfigs {
		var err error
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
			newContext, actionChanges, err = executeAction(rules.ActionStateToMetaState{
				Db:     db,
				Ctx:    ctx,
				UserID: &userID,
			}, actionConfig, newContext, contextChanges, &actionChanges)
		default:
			return nil, nil, errs.New("action key " + actionKey + " is unknown")
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
func executeAction(act rules.Action, configuration string, newContext change.Detector, contextChanges change.Set, actionChanges *change.Set) (change.Detector, change.Set, error) {
	if act == nil {
		return nil, nil, errs.New("rule can not be nil")
	}
	return act.OnChange(newContext, contextChanges, configuration, actionChanges)
}
