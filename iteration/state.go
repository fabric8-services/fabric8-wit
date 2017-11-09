package iteration

import (
	"database/sql/driver"
	"fmt"

	"github.com/fabric8-services/fabric8-wit/errors"
)

// State defines an iterations state
type State string

const (
	// StateNew represents a new iteration
	StateNew State = "new"
	// StateStart represents a started iteration
	StateStart State = "start"
	// StateClose represents a closed iteration
	StateClose State = "close"
)

// IsSet returns true if no state is specified
func (s State) IsSet() bool {
	return s != ""
}

// String implements the Stringer interface
func (s State) String() string { return string(s) }

// StringPtr returns a pointer to the string
func (s State) StringPtr() *string { tmp := string(s); return &tmp }

// Scan implements the https://golang.org/pkg/database/sql/#Scanner interface
// See also https://stackoverflow.com/a/25374979/835098 See also
// https://github.com/jinzhu/gorm/issues/302#issuecomment-80566841
func (s *State) Scan(value interface{}) error {
	switch value.(type) {
	case State:
		*s = value.(State)
	case string:
		*s = State(value.(string))
	case []byte:
		*s = State(value.([]byte))
	case nil:
		*s = State("illegal value: nil")
	default:
		*s = State(fmt.Sprintf("illegal value: %+v", value))
	}
	return s.CheckValid()
}

// Value implements the https://golang.org/pkg/database/sql/driver/#Valuer
// interface
func (s State) Value() (driver.Value, error) { return string(s), nil }

// CheckValid returns nil if the given iteration state is valid; otherwise a
// BadParameterError is returned.
func (s State) CheckValid() error {
	switch s {
	case StateNew, StateStart, StateClose:
		return nil
	default:
		return errors.NewBadParameterError("iteration state", s).Expected(StateNew + "|" + StateStart + "|" + StateClose)
	}
}
