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

	if valueType.Kind() != reflect.Array {
		return nil, fmt.Errorf("value %v should be %s, but is %s", value, "array", valueType.Name())
	}
	valueArray := value.([]interface{})
	converted := make([]interface{}, len(valueArray))
	for i, _ := range converted {
		var err error
		converted[i], err = converter(fieldType, valueArray[i])
		if err != nil {
			return nil, fmt.Errorf("error converting list value: %s", err.Error())
		}
	}
	return converted, nil

}
