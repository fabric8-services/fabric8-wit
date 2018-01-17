package id

import uuid "github.com/satori/go.uuid"

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
