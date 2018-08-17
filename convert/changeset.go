package convert

// ChangeDetector defines funcs for getting a changeset from two
// instances of a class. This interface has to be implemented by
// all
type ChangeDetector interface {
	ChangeSet(older ChangeDetector) ([]Change, error)
}

// Change defines a set of changed values in an entity. It holds
// the attribute name as the key and old and new values.
type Change struct {
	AttributeName string
	NewValue      interface{}
	OldValue      interface{}
}
