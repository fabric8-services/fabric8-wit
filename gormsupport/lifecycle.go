package gormsupport

import (
	"reflect"
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/jinzhu/gorm"
)

// The Lifecycle struct contains all the items from gorm.Model except the ID field,
// hence we can embed the Lifecycle struct into Models that needs soft delete and alike.
type Lifecycle struct {
	CreatedAt time.Time  `json:"created_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func init() {
	oldFunc := gorm.NowFunc
	// we use microsecond precision timestamps in the db, so also use ms precision timestamps in gorm callbacks.
	gorm.NowFunc = func() time.Time {
		return oldFunc().Round(time.Microsecond)
	}
}

// Ensure Lifecyle implements the Equaler interface
var _ convert.Equaler = Lifecycle{}
var _ convert.Equaler = (*Lifecycle)(nil)

// Equal returns true if two Lifecycle objects are equal; otherwise false is returned.
func (lc Lifecycle) Equal(u convert.Equaler) bool {
	other, ok := u.(Lifecycle)
	if !ok {
		return false
	}
	if !lc.CreatedAt.Equal(other.CreatedAt) {
		return false
	}
	if !lc.UpdatedAt.Equal(other.UpdatedAt) {
		return false
	}
	if !reflect.DeepEqual(lc.DeletedAt, other.DeletedAt) {
		return false
	}
	return true
}

// EqualValue implements convert.Equaler
func (lc Lifecycle) EqualValue(u convert.Equaler) bool {
	return lc.Equal(u)
}
