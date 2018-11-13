package workitem

import (
	"fmt"
	"reflect"

	"github.com/fabric8-services/fabric8-wit/convert"
	errs "github.com/pkg/errors"
)

// ListType describes a list of SimpleType values
type ListType struct {
	SimpleType    `json:"simple_type"`
	ComponentType SimpleType  `json:"component_type"`
	DefaultValue  interface{} `json:"default_value,omitempty"`
}

// Ensure ListType implements the FieldType interface
var _ FieldType = ListType{}
var _ FieldType = (*ListType)(nil)

// Ensure ListType implements the Equaler interface
var _ convert.Equaler = ListType{}
var _ convert.Equaler = (*ListType)(nil)

// Validate checks that the type of the list is "list", that the component type
// iteself a simple tpye (e.g. not a list or an enum) and that the default value
// matches the Kind of the ComponentType.
func (t ListType) Validate() error {
	if t.Kind != KindList {
		return errs.Errorf(`list type cannot have a base type "%s" but needs "%s"`, t.Kind, KindList)
	}
	if !t.ComponentType.Kind.IsSimpleType() {
		return errs.Errorf(`list type must have a simple component type and not "%s"`, t.Kind)
	}
	_, err := t.SetDefaultValue(t.DefaultValue)
	if err != nil {
		return errs.Wrapf(err, "failed to validate default value for kind %s: %+v (%[1]T)", t.Kind, t.DefaultValue)
	}
	return nil
}

// SetDefaultValue implements FieldType
func (t ListType) SetDefaultValue(v interface{}) (FieldType, error) {
	if v == nil {
		t.DefaultValue = nil
		return t, nil
	}
	defVal, err := t.ComponentType.ConvertToModel(v)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to set default value of list type to %+v (%[1]T)", v)
	}
	t.DefaultValue = defVal
	return t, nil
}

// GetDefaultValue implements FieldType
func (t ListType) GetDefaultValue() interface{} {
	return t.DefaultValue
}

// Equal returns true if two ListType objects are equal; otherwise false is returned.
func (t ListType) Equal(u convert.Equaler) bool {
	other, ok := u.(ListType)
	if !ok {
		return false
	}
	if !convert.CascadeEqual(t.SimpleType, other.SimpleType) {
		return false
	}
	if !reflect.DeepEqual(t.DefaultValue, other.DefaultValue) {
		return false
	}
	return convert.CascadeEqual(t.ComponentType, other.ComponentType)
}

// EqualValue implements convert.Equaler interface
func (t ListType) EqualValue(u convert.Equaler) bool {
	return t.Equal(u)
}

// ConvertToModel implements the FieldType interface
func (t ListType) ConvertToModel(value interface{}) (interface{}, error) {
	// the assumption is that work item types do not change over time...only new ones can be created
	return ConvertList(func(fieldType FieldType, value interface{}) (interface{}, error) {
		return fieldType.ConvertToModel(value)
	}, t.ComponentType, value)

}

// ConvertFromModel implements the FieldType interface
func (t ListType) ConvertFromModel(value interface{}) (interface{}, error) {
	// the assumption is that work item types do not change over time...only new ones can be created
	return ConvertList(func(fieldType FieldType, value interface{}) (interface{}, error) {
		return fieldType.ConvertFromModel(value)
	}, t.ComponentType, value)
}

// ConvertToStringArray implements the FieldType interface
func (t ListType) ConvertToStringArray(value interface{}) ([]string, error) {
	if value == nil {
		return []string{}, nil
	}
	valueList, err := ConvertList(func(fieldType FieldType, value interface{}) (interface{}, error) {
		return fieldType.ConvertToStringArray(value)
	}, t.ComponentType, value)
	if err != nil {
		return nil, errs.Wrapf(err, "Failed to convert list type")
	}
	if (len(valueList)) == 0 {
		return []string{}, nil
	}
	buffer := make([]string, len(valueList))
	for i := range valueList {
		strValueList := valueList[i].([]string)
		if len(strValueList) != 1 {
			return nil, errs.Errorf("String conversion of base type did not return exactly one value")
		}
		buffer[i] = strValueList[0]
	}
	return buffer, nil
}

type Converter func(FieldType, interface{}) (interface{}, error)

const (
	stErrorNotArrayOrSlice = "value %v should be array/slice, but is %s"
	stErrorConvertingList  = "error converting list value: %s"
)

func ConvertList(converter Converter, componentType SimpleType, value interface{}) ([]interface{}, error) {
	// the assumption is that work item types do not change over time...only new ones can be created
	valueType := reflect.TypeOf(value)

	if value == nil {
		return nil, nil
	}
	if (valueType.Kind() != reflect.Array) && (valueType.Kind() != reflect.Slice) {
		return nil, fmt.Errorf(stErrorNotArrayOrSlice, value, valueType.Name())
	}
	valueArray := reflect.ValueOf(value)
	converted := make([]interface{}, valueArray.Len())
	for i := range converted {
		var err error
		// valueArray index value must be converted to Interface else it has TYPE=Value
		converted[i], err = converter(componentType, valueArray.Index(i).Interface())
		if err != nil {
			return nil, fmt.Errorf(stErrorConvertingList, err.Error())
		}
	}
	return converted, nil

}
