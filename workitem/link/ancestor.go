package link

import (
	"github.com/fabric8-services/fabric8-wit/id"
	uuid "github.com/satori/go.uuid"
)

// Ancestor is essentially an annotated work item ID. Each Ancestor knows for
// which original child it is the ancestor and whether or not itself is the
// root.
type Ancestor struct {
	ID              uuid.UUID `gorm:"column:ancestor" sql:"type:uuid"`
	DirectChildID   uuid.UUID `gorm:"column:direct_child" sql:"type:uuid"`
	OriginalChildID uuid.UUID `gorm:"column:original_child" sql:"type:uuid"`
	IsRoot          bool      `gorm:"column:is_root"`
	// Level encodes if an ancestor is a parent (1), grandparent (2),
	// grandgrandparent (3), and so forth for the given original child.
	Level int `gorm:"column:ancestor_level"`
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

// GetParentOf returns the immediated (level 1) ancestor (if any) of the given
// work item ID; otherwise nil is returned.
func (l AncestorList) GetParentOf(workItemID uuid.UUID) *Ancestor {
	return l.GetAncestorOf(workItemID, 1)
}

// GetAncestorOf returns the ancestor (if any) of the given work item ID at the
// given level; otherwise nil is returned.
func (l AncestorList) GetAncestorOf(workItemID uuid.UUID, level int) *Ancestor {
	for _, a := range l {
		if a.OriginalChildID == workItemID && a.Level == level {
			return &a
		}
	}
	return nil
}
