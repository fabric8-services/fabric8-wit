package rules

import "github.com/fabric8-services/fabric8-wit/actions/change"

const (
	// ActionKeyNil is the key for the ActionKeyNil action rule.
	ActionKeyNil = "Nil"
	// ActionKeyFieldSet is the key for the ActionKeyFieldSet action rule.
	ActionKeyFieldSet = "FieldSet"
	// ActionKeyStateToMetastate is the key for the ActionKeyStateToMetastate action rule.
	ActionKeyStateToMetastate = "BidirectionalStateToColumn"

	// ActionKeyStateToMetastateConfigMetastate is the key for the ActionKeyStateToMetastateConfigMetastate config parameter.
	ActionKeyStateToMetastateConfigMetastate = "metaState"
)

// Action defines an action on change of an entity. Executing an
// Action might have sideffects, but will always return the original
// given context with all changes of the Action to this context. Note
// that the execution may have sideeffects on other entities beyond the
// context.
type Action interface {
	// OnChange executes this action by looking at a change set of
	// updated attributes. It returns the new context. Note that this
	// needs the new (after change) context and the old value(s) as
	// part of the changeset.
	OnChange(newContext change.Detector, contextChanges change.Set, configuration string, actionChanges *change.Set) (change.Detector, change.Set, error)
}
