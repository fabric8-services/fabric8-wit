package link

import (
	"github.com/fabric8-services/fabric8-wit/id"
	uuid "github.com/satori/go.uuid"
)

const (
	AncestorLevelAll              int64 = -1
	AncestorLevelParent           int64 = 1
	AncestorLevelGrandParent      int64 = 2
	AncestorLevelGreatGrandParent int64 = 3
)

// Ancestor is essentially an annotated work item ID. Each Ancestor knows for
// which original child it is the ancestor and whether or not itself is the
// root.
//
// NOTE: The sql columns noted here are purely virtual and not persitent, see
// the "working_table" in the query from GetAncestors() function to find out
// more about each column.
type Ancestor struct {
	ID              uuid.UUID `gorm:"column:ancestor" sql:"type:uuid"`
	DirectChildID   uuid.UUID `gorm:"column:direct_child" sql:"type:uuid"`
	OriginalChildID uuid.UUID `gorm:"column:original_child" sql:"type:uuid"`
	IsRoot          bool      `gorm:"column:is_root"`
	Level           int64     `gorm:"column:ancestor_level"`
}

// AncestorList is just an array of ancestor objects with additional
// functionality add to it.
type AncestorList []Ancestor

// GetDistinctAncestorIDs returns a list with distinct ancestor IDs.
func (l AncestorList) GetDistinctAncestorIDs() id.Slice {
	m := id.Map{}
	for _, ancestor := range l {
		m[ancestor.ID] = struct{}{}
	}
	return m.ToSlice()
}

// GetParentOf returns the first parent item that we can find in the list of
// ancestors; otherwise nil is returned.
func (l AncestorList) GetParentOf(workItemID uuid.UUID) *Ancestor {
	for _, a := range l {
		if a.DirectChildID == workItemID {
			return &a
		}
	}
	return nil
}
