package id

import (
	"strings"

	uuid "github.com/satori/go.uuid"
)

// Slice describes a slice of UUID objects
type Slice []uuid.UUID

// Diff returns the difference between this and the given slice.
func (s Slice) Diff(b Slice) Slice {
	slice := append(s, b...)
	encountered := map[uuid.UUID]int{}
	for _, v := range slice {
		encountered[v] = encountered[v] + 1
	}

	diff := []uuid.UUID{}
	for _, v := range slice {
		if encountered[v] == 1 {
			diff = append(diff, v)
		}
	}
	return diff
}

// Sub returns the result of removing elements in b from the given slice.
func (s Slice) Sub(b Slice) Slice {
	lut := map[uuid.UUID]struct{}{}
	for _, id := range b {
		lut[id] = struct{}{}
	}

	sub := []uuid.UUID{}
	for _, id := range s {
		if _, foundInB := lut[id]; !foundInB {
			sub = append(sub, id)
		}
	}
	return sub
}

// Unique returns a slice in which all duplicate elements have been removed.
func (s Slice) Unique(b Slice) Slice {
	return MapFromSlice(s).ToSlice()
}

// ToString returns all IDs as a string separated by the given separation
// string. If a callback is specified that callback will be called for every ID
// to generate a custom output string for that element.
func (s Slice) ToString(sep string, fn ...func(ID uuid.UUID) string) string {
	res := make([]string, len(s))
	i := 0
	for _, ID := range s {
		if len(fn) != 0 {
			res[i] = fn[0](ID)
		} else {
			res[i] = ID.String()
		}
		i++
	}
	return strings.Join(res, sep)
}

// String returns all IDs separated by ", ".
func (s Slice) String() string {
	return s.ToString(", ")
}
