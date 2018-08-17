package rules

import (
	"github.com/fabric8-services/fabric8-wit/actions/change"
)

// ActionNil is a dummy action rule that does nothing and has no sideffects.
type ActionNil struct {
}

// make sure the rule is implementing the interface.
var _ Action = ActionNil{}

// OnChange executes the action rule.
func (act ActionNil) OnChange(newContext change.Detector, contextChanges []change.Change, configuration string, actionChanges *[]change.Change) (change.Detector, []change.Change, error) {
	return newContext, nil, nil
}
