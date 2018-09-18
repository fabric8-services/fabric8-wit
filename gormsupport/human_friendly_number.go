package gormsupport

import (
	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// The HumanFriendlyNumber struct can be embedded in all model structs that want
// to have an automatically incremented human friendly number (e.g. 1,10,23).
// Such a number is unique within the space and for the given table name (e.g.
// 'work_items', 'iterations', 'areas').
type HumanFriendlyNumber struct {
	Number    int       `json:"number,omitempty"`
	spaceID   uuid.UUID `gorm:"-"`
	tableName string    `gorm:"-"`
}

// NewHumanFriendlyNumber TODO(kwk): document me
func NewHumanFriendlyNumber(spaceID uuid.UUID, tableName string, number ...int) HumanFriendlyNumber {
	n := 0
	if len(number) > 0 {
		n = number[0]
	}
	return HumanFriendlyNumber{
		Number:    n,
		spaceID:   spaceID,
		tableName: tableName,
	}
}

// Ensure Equaler implements the Equaler interface
var _ convert.Equaler = HumanFriendlyNumber{}
var _ convert.Equaler = (*HumanFriendlyNumber)(nil)

// Equal implements convert.Equaler
func (n HumanFriendlyNumber) Equal(u convert.Equaler) bool {
	other, ok := u.(HumanFriendlyNumber)
	if !ok {
		return false
	}
	if n.Number != other.Number {
		return false
	}
	if n.spaceID != other.spaceID {
		return false
	}
	if n.tableName != other.tableName {
		return false
	}
	return true
}

// EqualValue implements convert.Equaler
func (n HumanFriendlyNumber) EqualValue(u convert.Equaler) bool {
	return n.Equal(u)
}

// BeforeCreate is a GORM callback (see http://doc.gorm.io/callbacks.html) that
// will be called before creating the model. We use it to determine the next
// human readable number for the model and set it automatically in the CREATE
// query.
func (n *HumanFriendlyNumber) BeforeCreate(scope *gorm.Scope) error {
	upsertStmt := `
		INSERT INTO number_sequences (space_id, table_name, current_val)
		VALUES ($1, $2, 1)
		ON CONFLICT (space_id, table_name)
		DO UPDATE SET current_val = number_sequences.current_val + EXCLUDED.current_val
		RETURNING current_val
	`
	var currentVal int
	err := scope.NewDB().Debug().CommonDB().QueryRow(upsertStmt, n.spaceID, n.tableName).Scan(&currentVal)
	if err != nil {
		return errs.Wrapf(err, "failed to obtain next val for space %q and table %q", n.spaceID, n.tableName)
	}
	log.Debug(nil, map[string]interface{}{
		"space_id":   n.spaceID,
		"table_name": n.tableName,
		"next_val":   currentVal,
	}, "computed nextVal")

	n.Number = currentVal
	return scope.SetColumn("number", n.Number)
}
