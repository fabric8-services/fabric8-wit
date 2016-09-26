package models

import (
	"fmt"
	"reflect"
)

//ListType describes a list of SimpleType values
type ListType struct {
	SimpleType
	ComponentType SimpleType
}

// Ensure ListType implements the Equaler interface
var _ Equaler = ListType{}
var _ Equaler = (*ListType)(nil)

// Equal returns true if two ListType objects are equal; otherwise false is returned.
func (self ListType) Equal(u Equaler) bool {
	other, ok := u.(ListType)
	if !ok {
		return false
	}
	if !self.SimpleType.Equal(other.SimpleType) {
		return false
	}
	return self.ComponentType.Equal(other.ComponentType)
}

// ConvertToModel implements the FieldType interface
func (fieldType ListType) ConvertToModel(value interface{}) (interface{}, error) {
	// the assumption is that work item types do not change over time...only new ones can be created
	return convertList(func(fieldType FieldType, value interface{}) (interface{}, error) {
		return fieldType.ConvertToModel(value)
	}, fieldType.ComponentType, value)

}

// ConvertFromModel implements the FieldType interface
func (fieldType ListType) ConvertFromModel(value interface{}) (interface{}, error) {
	// the assumption is that work item types do not change over time...only new ones can be created
	return convertList(func(fieldType FieldType, value interface{}) (interface{}, error) {
		return fieldType.ConvertFromModel(value)
	}, fieldType.ComponentType, value)
}

type converter func(FieldType, interface{}) (interface{}, error)

const (
	stErrorNotArrayOrSlice = "value %v should be array/slice, but is %s"
	stErrorConvertingList  = "error converting list value: %s"
)

func convertList(converter converter, componentType SimpleType, value interface{}) ([]interface{}, error) {
	// the assumption is that work item types do not change over time...only new ones can be created
	valueType := reflect.TypeOf(value)

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
