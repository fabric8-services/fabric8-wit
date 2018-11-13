package workitem

import (
	"fmt"
	"reflect"

	"github.com/fabric8-services/fabric8-wit/convert"
	errs "github.com/pkg/errors"
)

// The EnumType defines the members that make up an enum field type definition.
// The SimpleType is set to KindEnum and the BaseType is set to whatever type of
// enum you want to have (e.g. an enum of strings or integers). The Values array
// specifies what the allowed values in this enum are. If RewritableValues is
// set to true, this type can be overwritten by a work item type that also
// defines a field of the same name with the same type, except with different
// allowed values inside. A classic example for this is the state field that can
// be overwritten by every work item type to fit its needs.
type EnumType struct {
	SimpleType       `json:"simple_type"`
	BaseType         SimpleType    `json:"base_type"`
	Values           []interface{} `json:"values"`
	RewritableValues bool          `json:"rewritable_values"`
	DefaultValue     interface{}   `json:"default_value,omitempty"`
}

// Ensure EnumType implements the FieldType interface
var _ FieldType = EnumType{}
var _ FieldType = (*EnumType)(nil)

// Validate checks that the type of the enum is "enum", that the base type
// itself a simple type (e.g. not a list or an enum), that the default value
// matches the Kind of the BaseType, that the default value is in the list of
// allowed values and that the Values are all of the base type.
func (t EnumType) Validate() error {
	if t.Kind != KindEnum {
		return errs.Errorf(`enum has a base type "%s" but needs "%s"`, t.Kind, KindEnum)
	}
	if !t.BaseType.Kind.IsSimpleType() {
		return errs.Errorf(`enum type must have a simple component type and not "%s"`, t.Kind)
	}
	_, err := t.SetDefaultValue(t.DefaultValue)
	if err != nil {
		return errs.Wrapf(err, "failed to validate default value for kind %s: %+v (%[1]T)", t.Kind, t.DefaultValue)
	}
	// verify that we have a set of permitted values
	if t.Values == nil || len(t.Values) <= 0 {
		return errs.Errorf("enum type has no values: %+v", t)
	}
	for i, v := range t.Values {
		_, err := t.ConvertToModel(v)
		if err != nil {
			return errs.Wrapf(err, `failed to convert value at position %d to kind "%s": %+v`, i, t.BaseType, v)
		}
	}
	return nil
}

// SetDefaultValue implements FieldType
func (t EnumType) SetDefaultValue(v interface{}) (FieldType, error) {
	if v == nil {
		t.DefaultValue = nil
		return t, nil
	}
	defVal, err := t.ConvertToModel(v)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to set default value of enum type to %+v (%[1]T)", v)
	}
	t.DefaultValue = defVal
	return t, nil
}

// GetDefaultValue implements FieldType
func (t EnumType) GetDefaultValue() interface{} {
	// manual default value has precedence over first value in list of allowed
	// values
	if t.DefaultValue != nil {
		return t.DefaultValue
	}
	// fallback to first permitted element
	if len(t.Values) > 0 {
		return t.Values[0]
	}
	return nil
}

// Ensure EnumType implements the Equaler interface
var _ convert.Equaler = EnumType{}
var _ convert.Equaler = (*EnumType)(nil)

// Equal returns true if two EnumType objects are equal; otherwise false is returned.
func (t EnumType) Equal(u convert.Equaler) bool {
	other, ok := u.(EnumType)
	if !ok {
		return false
	}
	if !convert.CascadeEqual(t.SimpleType, other.SimpleType) {
		return false
	}
	if !convert.CascadeEqual(t.BaseType, other.BaseType) {
		return false
	}
	if !t.RewritableValues {
		if !reflect.DeepEqual(t.Values, other.Values) {
			return false
		}
	}
	if !reflect.DeepEqual(t.DefaultValue, other.DefaultValue) {
		return false
	}
	return true
}

// EqualValue implements convert.Equaler
func (t EnumType) EqualValue(u convert.Equaler) bool {
	return t.Equal(u)
}

// EqualEnclosing returns true if two EnumType objects are equal and/or the
// values set is enclosing (larger and containing) the other values set.
func (t EnumType) EqualEnclosing(other EnumType) bool {
	if !t.SimpleType.Equal(other.SimpleType) {
		return false
	}
	if !t.BaseType.Equal(other.BaseType) {
		return false
	}
	// if the local list of values is completely contained
	// in the other values set, consider it enclosing.
	if !t.RewritableValues {
		return containsAll(t.Values, other.Values)
	}
	return true
}

func (t EnumType) ConvertToModel(value interface{}) (interface{}, error) {
	converted, err := t.BaseType.ConvertToModel(value)
	if err != nil {
		return nil, errs.Errorf("error converting enum value: %s", err.Error())
	}

	if !contains(t.Values, converted) {
		return nil, fmt.Errorf("value: %+v (%[1]T) is not part of allowed enum values: %+v", value, t.Values)
	}
	return converted, nil
}

// ConvertToStringArray implements the FieldType interface
func (t EnumType) ConvertToStringArray(value interface{}) ([]string, error) {
	if value != nil && !contains(t.Values, value) {
		return nil, errs.Errorf("value: %+v (%[1]T) is not part of allowed enum values: %+v", value, t.Values)
	}
	converted, err := t.BaseType.ConvertToStringArray(value)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to convert enum value to string: %+v", value)
	}
	return converted, nil
}

func contains(a []interface{}, v interface{}) bool {
	for _, element := range a {
		if element == v {
			return true
		}
	}
	return false
}

func containsAll(a []interface{}, v []interface{}) bool {
	result := true
	for _, element := range v {
		result = result && contains(a, element)
	}
	return result
}

// ConvertFromModel implements the FieldType interface
func (t EnumType) ConvertFromModel(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	converted, err := t.BaseType.ConvertFromModel(value)
	if err != nil {
		return nil, errs.Errorf("error converting enum value: %s", err.Error())
	}
	if !contains(t.Values, converted) {
		return nil, errs.Errorf("value: %+v (%[1]T) is not part of allowed enum values: %+v", value, t.Values)
	}
	return converted, nil
}
