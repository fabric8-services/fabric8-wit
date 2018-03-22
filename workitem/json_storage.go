package workitem

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"

	"github.com/fabric8-services/fabric8-wit/convert"
	errs "github.com/pkg/errors"
)

type Fields map[string]interface{}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = Fields{}
var _ convert.Equaler = (*Fields)(nil)

// Ensure Fields implements the Scanner and Valuer interfaces
var _ sql.Scanner = (*Fields)(nil)
var _ driver.Valuer = (*Fields)(nil)

// Equal returns true if two Fields objects are equal; otherwise false is returned.
// TODO: (kwk) think about a better comparison for Fields map.
func (f Fields) Equal(u convert.Equaler) bool {
	other, ok := u.(Fields)
	if !ok {
		return false
	}
	return reflect.DeepEqual(f, other)
}

func (f Fields) Value() (driver.Value, error) {
	return toBytes(f)
}

// Scan implements the https://golang.org/pkg/database/sql/#Scanner interface
// See also https://stackoverflow.com/a/25374979/835098
// See also https://github.com/jinzhu/gorm/issues/302#issuecomment-80566841
func (f *Fields) Scan(src interface{}) error {
	return fromBytes(src, f)
}

type FieldDefinitions map[string]FieldDefinition

// Ensure FieldDefinitions implements the Scanner and Valuer interfaces
var _ sql.Scanner = (*FieldDefinitions)(nil)
var _ driver.Valuer = (*FieldDefinitions)(nil)

// Value implements the https://golang.org/pkg/database/sql/driver/#Valuer interface
func (j FieldDefinitions) Value() (driver.Value, error) {
	return toBytes(j)
}

// Scan implements the https://golang.org/pkg/database/sql/#Scanner interface
// See also https://stackoverflow.com/a/25374979/835098
// See also https://github.com/jinzhu/gorm/issues/302#issuecomment-80566841
func (j *FieldDefinitions) Scan(src interface{}) error {
	return fromBytes(src, j)
}

func toBytes(j interface{}) (driver.Value, error) {
	if j == nil {
		// log.Trace("returning null")
		return nil, nil
	}

	res, error := json.Marshal(j)
	return res, error
}

func fromBytes(src interface{}, target interface{}) error {
	if src == nil {
		target = nil
		return nil
	}
	s, ok := src.([]byte)
	if !ok {
		return errs.Errorf("scan source was not a string")
	}
	return json.Unmarshal(s, target)
}
