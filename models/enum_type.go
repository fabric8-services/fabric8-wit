package models

import (
	"fmt"
)

type EnumType struct {
	SimpleType
	BaseType SimpleType
	Values   []interface{}
}

func (fieldType EnumType) ConvertToModel(value interface{}) (interface{}, error) {
	converted, err := fieldType.BaseType.ConvertToModel(value)
	if err != nil {
		return nil, fmt.Errorf("error converting enum value: %s", err.Error())
	}

	if !contains(fieldType.Values, converted) {
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

func (fieldType EnumType) ConvertFromModel(value interface{}) (interface{}, error) {
	converted, err := fieldType.BaseType.ConvertToModel(value)
	if err != nil {
		return nil, fmt.Errorf("error converting enum value: %s", err.Error())
	}
	return converted, nil
}
