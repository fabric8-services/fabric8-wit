package models

import (
	"fmt"
	"reflect"
)

type ListType struct {
	SimpleType
	ComponentType SimpleType
}

func (fieldType ListType) ConvertToModel(value interface{}) (interface{}, error) {
		// the assumption is that work item types do not change over time...only new ones can be created
	access := func(t FieldType) converter {
		return t.ConvertToModel
	}
	return convertList(access, fieldType, value)

}

func (fieldType ListType) ConvertFromModel(value interface{}) (interface{}, error) {
	// the assumption is that work item types do not change over time...only new ones can be created
	access := func(t FieldType) converter {
		return t.ConvertFromModel
	}
	return convertList(access, fieldType, value)
}

type converter func(interface{}) (interface{}, error)

func convertList(converterAccess func(baseType FieldType) converter, fieldType ListType, value interface{}) (interface{}, error) {
	// the assumption is that work item types do not change over time...only new ones can be created
	valueType := reflect.TypeOf(value)

	if valueType.Kind() != reflect.Array {
		return nil, fmt.Errorf("value %v should be %s, but is %s", value, "array", valueType.Name())
	}
	valueArray := value.([]interface{})
	converted := make([]interface{}, len(valueArray))
	for i, _ := range converted {
		var err error
		convert := converterAccess(fieldType.ComponentType)
		converted[i], err = convert(valueArray[i])
		if err != nil {
			return nil, fmt.Errorf("error converting list value: %s", err.Error())
		}
	}
	return converted, nil

}
