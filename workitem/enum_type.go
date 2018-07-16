package workitem

import (
	"fmt"
	"reflect"

	"github.com/fabric8-services/fabric8-wit/convert"
	errs "github.com/pkg/errors"
)

type EnumType struct {
	SimpleType       `json:"simple_type"`
	BaseType         SimpleType    `json:"base_type"`
	Values           []interface{} `json:"values"`
	RewritableValues bool          `json:"rewritable_values"`
}

// Ensure EnumType implements the FieldType interface
var _ FieldType = EnumType{}
var _ FieldType = (*EnumType)(nil)

// DefaultValue implements FieldType
func (t EnumType) DefaultValue(value interface{}) (interface{}, error) {
	if value != nil {
		return value, nil
	}
	if t.Values == nil || len(t.Values) <= 0 {
		return nil, errs.Errorf("enum has no values")
	}
	return t.Values[0], nil
}

// Ensure EnumType implements the FieldType interface
var _ FieldType = EnumType{}
var _ FieldType = (*EnumType)(nil)

// Ensure EnumType implements the Equaler interface
var _ convert.Equaler = EnumType{}
var _ convert.Equaler = (*EnumType)(nil)

// Equal returns true if two EnumType objects are equal; otherwise false is returned.
func (t EnumType) Equal(u convert.Equaler) bool {
	// for the EnumType, we consider enclosed Values as an equality to
	// allow changes to the template to be adding new values to the
	// enum. If a default Equal() behaviour is needed, use DefaultEqual().
	return t.EqualEnclosing(u)
}

// DefaultEqual returns true if two EnumType objects are equal; otherwise false is returned.
func (t EnumType) DefaultEqual(u convert.Equaler) bool {
	other, ok := u.(EnumType)
	if !ok {
		return false
	}
	if !t.SimpleType.Equal(other.SimpleType) {
		return false
	}
	if !t.BaseType.Equal(other.BaseType) {
		return false
	}
	if !t.RewritableValues {
		return reflect.DeepEqual(t.Values, other.Values)
	}
	return true
}

// EqualEnclosing returns true if two EnumType objects are equal and/or the
// values set is enclosing (larger and containing) the other values set.
func (t EnumType) EqualEnclosing(u convert.Equaler) bool {
	other, ok := u.(EnumType)
	if !ok {
		return false
	}
	if !t.SimpleType.Equal(other.SimpleType) {
		return false
	}
	if !t.BaseType.Equal(other.BaseType) {
		return false
	}
	if t.RewritableValues != other.RewritableValues {
		return false
	}
	// if the local list of values is completely contained
	// in the other values set, consider it enclosing.
	return containsAll(t.Values, other.Values)
}

func (t EnumType) ConvertToModel(value interface{}) (interface{}, error) {
	converted, err := t.BaseType.ConvertToModel(value)
	if err != nil {
		return nil, fmt.Errorf("error converting enum value: %s", err.Error())
	}

	if !contains(t.Values, converted) {
		return nil, fmt.Errorf("not an enum value: %v", value)
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

func (t EnumType) ConvertFromModel(value interface{}) (interface{}, error) {
	converted, err := t.BaseType.ConvertToModel(value)
	if err != nil {
		return nil, fmt.Errorf("error converting enum value: %s", err.Error())
	}
	return converted, nil
}
