package change

// Set is a set of changes to an entitiy.
type Set []Change

// Detector defines funcs for getting a changeset from two
// instances of a class. This interface has to be implemented by
// all entities that should trigger action rule runs.
type Detector interface {
	ChangeSet(older Detector) (Set, error)
}

// Change defines a set of changed values in an entity. It holds
// the attribute name as the key and old and new values.
type Change struct {
	AttributeName string
	NewValue      interface{}
	OldValue      interface{}
}
