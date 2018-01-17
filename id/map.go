package id

import (
	"strings"

	uuid "github.com/satori/go.uuid"
)

// Map describes a map of UUID keys with empty structs as their values
type Map map[uuid.UUID]struct{}

// MapFromSlice constructs a map of empty struct values with keys taken from the
// given slice.
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

// Copy returns a standalone copy of this map.
func (m Map) Copy() Map {
	res := Map{}
	for k, v := range m {
		res[k] = v
	}
	return res
}

// ToString returns all IDs as a string separated by the given separation
// string. If a callback is specified that callback will be called for every ID
// to generate a custom output string for that element.
func (m Map) ToString(sep string, fn ...func(ID uuid.UUID) string) string {
	res := make([]string, len(m))
	i := 0
	for ID := range m {
		if len(fn) != 0 {
			res[i] = fn[0](ID)
		} else {
			res[i] = ID.String()
		}
		i++
	}
	return strings.Join(res, sep)
}

// String returns all map keys separated by ", ".
func (m Map) String() string {
	return m.ToString(", ")
}
