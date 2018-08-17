package rules

import (
	"github.com/fabric8-services/fabric8-wit/convert"
)

// ActionNil is a dummy action rule that does nothing and has no sideffects.
type ActionNil struct {
}

// make sure the rule is implementing the interface.
var _ Action = ActionNil{}

// OnChange executes the action rule.
func (act ActionNil) OnChange(newContext convert.ChangeDetector, contextChanges []convert.Change, configuration string, actionChanges *[]convert.Change) (convert.ChangeDetector, []convert.Change, error) {
	return newContext, nil, nil
}
