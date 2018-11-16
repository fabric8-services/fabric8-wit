package convert

import (
	"runtime"
	"strings"
)

// The Equaler interface allows object that implement it, to be compared
type Equaler interface {
	// Equal returns true if the Equaler object is the same as this object;
	// otherwise false is returned. You need to convert the Equaler object using
	// type assertions: https://golang.org/ref/spec#Type_assertions
	Equal(Equaler) bool
	// EqualValue does the same as Equal but should ignore meta values like the
	// version of an object or times when it was created, last updated or
	// deleted.
	EqualValue(Equaler) bool
}

// EqualValue is an internal function and shouldn't be called from the outside
// but it needs to be exported for some calling logic that is embedeed into the
// CallEqualByName and IsCallerEqualValue functions.
func EqualValue(obj Equaler, other Equaler) bool {
	return obj.EqualValue(other)
}

// CascadeEqual is a helper function that you can use to automatically pick the
// correct equal function (Equal/EqualValue) depending on how the current equal
// function is called.
//
// CascadeEqual calls obj.EqualValue(other) if the call to the calling function
// ends was made from a function that ends with ".EqualValue"; otherwise
// obj.Equal(other) is returned.
func CascadeEqual(obj Equaler, other Equaler) bool {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		return false
	}
	if strings.HasSuffix(runtime.FuncForPC(pc).Name(), ".EqualValue") {
		// We use this indirection so that CascadeEqual can operator
		// successfully when called again.
		return EqualValue(obj, other)
	}
	return obj.Equal(other)
}

// DummyEqualer implements the Equaler interface and can be used by tests. Other
// than that it has not meaning.
type DummyEqualer struct {
}

// Ensure DummyEqualer implements the Equaler interface
var _ Equaler = DummyEqualer{}
var _ Equaler = (*DummyEqualer)(nil)

// Equal implements Equaler
func (d DummyEqualer) Equal(u Equaler) bool {
	_, ok := u.(DummyEqualer)
	return ok
}

// EqualValue implements Equaler
func (d DummyEqualer) EqualValue(u Equaler) bool {
	_, ok := u.(DummyEqualer)
	return ok
}
