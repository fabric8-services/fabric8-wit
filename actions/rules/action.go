package rules

import "github.com/fabric8-services/fabric8-wit/convert"

const (
	ActionKeyNil              = "Nil"
	ActionKeyFieldSet         = "FieldSet"
	ActionKeyStateToMetastate = "BidirectionalStateToColumn"

	ActionKeyStateToMetastateConfigMetastate = "metaState"
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
	OnChange(newContext convert.ChangeDetector, contextChanges []convert.Change, configuration string, actionChanges *[]convert.Change) (convert.ChangeDetector, []convert.Change, error)
}
