package workitem

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/fabric8-services/fabric8-wit/convert"
	errs "github.com/pkg/errors"
)

// constants for describing possible field types
const (
	// non-relational
	KindString   Kind = "string"
	KindInteger  Kind = "integer"
	KindFloat    Kind = "float"
	KindBoolean  Kind = "bool"
	KindInstant  Kind = "instant"
	KindDuration Kind = "duration"
	KindURL      Kind = "url"
	KindMarkup   Kind = "markup"
	// relational
	KindIteration   Kind = "iteration"
	KindUser        Kind = "user"
	KindLabel       Kind = "label"
	KindBoardColumn Kind = "boardcolumn"
	KindArea        Kind = "area"
	KindCodebase    Kind = "codebase"
	// composite
	KindEnum Kind = "enum"
	KindList Kind = "list"
)

// Kind is the kind of field type
type Kind string

// IsSimpleType returns 'true' if the kind is simple, i.e., not a list nor an enum
func (k Kind) IsSimpleType() bool {
	return k != KindEnum && k != KindList
}

// IsRelational returns 'true' if the kind must be represented with a
// relationship.
func (k Kind) IsRelational() bool {
	switch k {
	case KindIteration,
		KindUser,
		KindLabel,
		KindBoardColumn,
		KindArea,
		KindCodebase:
		return true
	}
	return false
}

// String implements the Stringer interface and returns the kind as a string
// object.
func (k Kind) String() string {
	return string(k)
}

// FieldType describes the possible values of a FieldDefinition
type FieldType interface {
	GetKind() Kind
	// ConvertToModel converts a field value for use in the persistence layer
	ConvertToModel(value interface{}) (interface{}, error)
	// ConvertFromModel converts a field value for use in the REST API layer
	ConvertFromModel(value interface{}) (interface{}, error)
	// Implement the Equaler interface
	Equal(u convert.Equaler) bool
	// DefaultValue is called if a field is not specified. In it's simplest form
	// the DefaultValue returns the given input value without any conversion.
	DefaultValue(value interface{}) (interface{}, error)
}

// FieldDefinition describes type & other restrictions of a field
type FieldDefinition struct {
	Required    bool      `json:"required"`
	ReadOnly    bool      `json:"read_only"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
	Type        FieldType `json:"type"`
}

// Ensure FieldDefinition implements the Equaler interface
var _ convert.Equaler = FieldDefinition{}
var _ convert.Equaler = (*FieldDefinition)(nil)

// Ensure FieldDefinition implements the json.Unmarshaler interface
var _ json.Unmarshaler = (*FieldDefinition)(nil)

// // Ensure FieldDefinition implements the yaml.Unmarshaler interface
// var _ yaml.Unmarshaler = (*FieldDefinition)(nil)

// Equal returns true if two FieldDefinition objects are equal; otherwise false is returned.
func (f FieldDefinition) Equal(u convert.Equaler) bool {
	other, ok := u.(FieldDefinition)
	if !ok {
		return false
	}
	if f.Required != other.Required {
		return false
	}
	if f.ReadOnly != other.ReadOnly {
		return false
	}
	if f.Label != other.Label {
		return false
	}
	if f.Description != other.Description {
		return false
	}
	return f.Type.Equal(other.Type)
}

// ConvertToModel converts a field value for use in the persistence layer
func (f FieldDefinition) ConvertToModel(name string, value interface{}) (interface{}, error) {
	// Overwrite value if default value if none was provided
	if value == nil {
		defValue, err := f.Type.DefaultValue(value)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to get default value for field \"%s\"", name)
		}
		value = defValue
	}

	if f.Required {
		if value == nil {
			return nil, fmt.Errorf("value for field \"%s\" must not be nil", name)
		}
		if f.Type.GetKind() == KindString {
			sVal, ok := value.(string)
			if !ok {
				return nil, errs.Errorf("failed to convert '%+v' to string", spew.Sdump(value))
			}
			if strings.TrimSpace(sVal) == "" {
				return nil, errs.Errorf("value for field \"%s\" must not be empty: \"%+v\"", name, value)
			}
		}
	}
	return f.Type.ConvertToModel(value)
}

// ConvertFromModel converts a field value for use in the REST API layer
func (f FieldDefinition) ConvertFromModel(name string, value interface{}) (interface{}, error) {
	if f.Required && value == nil {
		return nil, fmt.Errorf("Value %s is required", name)
	}
	return f.Type.ConvertFromModel(value)
}

type rawFieldDef struct {
	Required    bool             `json:"required"`
	ReadOnly    bool             `json:"read_only"`
	Label       string           `json:"label"`
	Description string           `json:"description"`
	Type        *json.RawMessage `json:"type"`
}

// Ensure rawFieldDef implements the Equaler interface
var _ convert.Equaler = rawFieldDef{}
var _ convert.Equaler = (*rawFieldDef)(nil)

// Equal returns true if two rawFieldDef objects are equal; otherwise false is returned.
func (f rawFieldDef) Equal(u convert.Equaler) bool {
	other, ok := u.(rawFieldDef)
	if !ok {
		return false
	}
	if f.Required != other.Required {
		return false
	}
	if f.ReadOnly != other.ReadOnly {
		return false
	}
	if f.Label != other.Label {
		return false
	}
	if f.Description != other.Description {
		return false
	}
	if f.Type == nil && other.Type == nil {
		return true
	}
	if f.Type != nil && other.Type != nil {
		return reflect.DeepEqual(f.Type, other.Type)
	}
	return false
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (f *FieldDefinition) UnmarshalJSON(bytes []byte) error {
	temp := rawFieldDef{}

	err := json.Unmarshal(bytes, &temp)
	if err != nil {
		return errs.Wrapf(err, "failed to unmarshall field definition into rawFieldDef")
	}
	rawType := map[string]interface{}{}
	err = json.Unmarshal(*temp.Type, &rawType)
	if err != nil {
		return errs.Wrapf(err, "failed to unmarshall from json.RawMessage to a map: %+v", *temp.Type)
	}

	var rawKind interface{}
	rawKind, hasRawKind := rawType["kind"]
	if !hasRawKind {
		simpleType, hasSimpleType := rawType["simple_type"]
		if hasSimpleType {
			simpleTypeMap, ok := simpleType.(map[string]interface{})
			if ok {
				rawKind = simpleTypeMap["kind"]
			}
		}
	}
	kind, err := ConvertAnyToKind(rawKind)
	if err != nil {
		// return the first error anyway
		return errs.Wrapf(err, "failed to convert any '%+v' to kind", rawKind)
	}

	switch *kind {
	case KindList:
		theType := ListType{}
		err = json.Unmarshal(*temp.Type, &theType)
		if err != nil {
			return errs.WithStack(err)
		}
		*f = FieldDefinition{Type: theType, Required: temp.Required, ReadOnly: temp.ReadOnly, Label: temp.Label, Description: temp.Description}
	case KindEnum:
		theType := EnumType{}
		err = json.Unmarshal(*temp.Type, &theType)
		if err != nil {
			return errs.WithStack(err)
		}
		*f = FieldDefinition{Type: theType, Required: temp.Required, ReadOnly: temp.ReadOnly, Label: temp.Label, Description: temp.Description}
	default:
		theType := SimpleType{}
		err = json.Unmarshal(*temp.Type, &theType)
		if err != nil {
			return errs.WithStack(err)
		}
		*f = FieldDefinition{Type: theType, Required: temp.Required, ReadOnly: temp.ReadOnly, Label: temp.Label, Description: temp.Description}
	}
	return nil
}

func ConvertAnyToKind(any interface{}) (*Kind, error) {
	k, ok := any.(string)
	if !ok {
		return nil, errs.Errorf("kind is not a string value %v", any)
	}
	return ConvertStringToKind(k)
}

// ConvertStringToKind returns the given string as a Kind object
func ConvertStringToKind(k string) (*Kind, error) {
	kind := Kind(k)
	switch kind {
	case KindString, KindInteger, KindFloat, KindInstant, KindDuration, KindURL, KindUser, KindEnum, KindList, KindIteration, KindMarkup, KindArea, KindCodebase, KindLabel, KindBoardColumn, KindBoolean:
		return &kind, nil
	}
	return nil, errs.Errorf("kind '%s' is not a simple type", k)
}

// compatibleFields returns true if the existing and new field are compatible;
// otherwise false is returned. It does so by comparing all members of the field
// definition except for the label and description.
func compatibleFields(existing FieldDefinition, new FieldDefinition) bool {
	if existing.Required != new.Required {
		return false
	}
	return existing.Type.Equal(new.Type)
}
