package models

// The Equaler interface allows object that implement it, to be compared
type Equaler interface {
	Equal(Equaler) bool
}