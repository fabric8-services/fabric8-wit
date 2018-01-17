package id

import uuid "github.com/satori/go.uuid"

// Map describes a map of UUID keys with empty structs as their values
type Map map[uuid.UUID]struct{}

// MapFromSlice constructs a map of empty struct values with keys taken from the
// given slice. As a consequence all potential IDs from the given slice are
// removed.
func MapFromSlice(s Slice) Map {
	m := Map{}
	for _, ID := range s {
		m[ID] = struct{}{}
	}
	return m
}

// ToSlice takes all keys from the given map and returns them as an array.
func (m Map) ToSlice() Slice {
	s := make(Slice, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	return s
}
