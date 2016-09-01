package models

// The Equaler interface allows object that implement it, to be compared
type Equaler interface {
	// Returns true if the Equaler object is the same as this object; otherwise false is returned.
	// You need to convert the Equaler object using type assertions: https://golang.org/ref/spec#Type_assertions
	Equal(Equaler) bool
}