package link

import (
	"database/sql/driver"
	"strconv"

	"github.com/fabric8-services/fabric8-wit/id"
	uuid "github.com/satori/go.uuid"
)

// AncestorLevel defines up to which level the GetAncestors function returns
// ancestors.
type AncestorLevel int64

const (
	AncestorLevelAll              AncestorLevel = -1
	AncestorLevelParent           AncestorLevel = 1
	AncestorLevelGrandParent      AncestorLevel = 2
	AncestorLevelGreatGrandParent AncestorLevel = 3
)

func (l AncestorLevel) String() string { return strconv.FormatInt(int64(l), 10) }

// Scan implements the https://golang.org/pkg/database/sql/#Scanner interface
// See also https://stackoverflow.com/a/25374979/835098
// See also https://github.com/jinzhu/gorm/issues/302#issuecomment-80566841
func (l *AncestorLevel) Scan(value interface{}) error { *l = AncestorLevel(value.(int64)); return nil }

// Value implements the https://golang.org/pkg/database/sql/driver/#Valuer interface
func (l AncestorLevel) Value() (driver.Value, error) { return int64(l), nil }

// Ancestor is essentially an annotated work item ID. Each Ancestor knows for
// which original child it is the ancestor and whether or not itself is the
// root.
//
// NOTE: The sql columns noted here are purely virtual and not persitent, see
// the "working_table" in the query from GetAncestors() function to find out
// more about each column.
type Ancestor struct {
	ID              uuid.UUID     `gorm:"column:ancestor" sql:"type:uuid"`
	DirectChildID   uuid.UUID     `gorm:"column:direct_child" sql:"type:uuid"`
	OriginalChildID uuid.UUID     `gorm:"column:original_child" sql:"type:uuid"`
	IsRoot          bool          `gorm:"column:is_root"`
	Level           AncestorLevel `gorm:"column:ancestor_level"`
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
func (l AncestorList) GetAncestorOf(workItemID uuid.UUID, level AncestorLevel) *Ancestor {
	for _, a := range l {
		if a.OriginalChildID == workItemID && a.Level == level {
			return &a
		}
	}
	return nil
}
