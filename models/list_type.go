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

// ConvertToModel implements the FieldType interface
func (fieldType ListType) ConvertToModel(value interface{}) (interface{}, error) {
	// the assumption is that work item types do not change over time...only new ones can be created
	return convertList(func(fieldType FieldType, value interface{}) (interface{}, error) {
		return fieldType.ConvertToModel(value)
	}, fieldType, value)

}

// ConvertFromModel implements the FieldType interface
func (fieldType ListType) ConvertFromModel(value interface{}) (interface{}, error) {
	// the assumption is that work item types do not change over time...only new ones can be created
	return convertList(func(fieldType FieldType, value interface{}) (interface{}, error) {
		return fieldType.ConvertFromModel(value)
	}, fieldType, value)
}

type converter func(FieldType, interface{}) (interface{}, error)

func convertList(converter converter, fieldType ListType, value interface{}) (interface{}, error) {
	// the assumption is that work item types do not change over time...only new ones can be created
	valueType := reflect.TypeOf(value)

	if (valueType.Kind() != reflect.Array) && (valueType.Kind() != reflect.Slice) {
		return nil, fmt.Errorf("value %v should be %s, but is %s", value, "array/slice", valueType.Name())
	}
	valueArray := reflect.ValueOf(value)
	converted := make([]interface{}, valueArray.Len())
	for i := range converted {
		var err error
		// valueArray index value must be converted to Interface else it has TYPE=Value
		converted[i], err = converter(fieldType.ComponentType, valueArray.Index(i).Interface())
		if err != nil {
			return nil, fmt.Errorf("error converting list value: %s", err.Error())
		}
	}
	return converted, nil

}
