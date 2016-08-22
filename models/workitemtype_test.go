// +build unit
package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestJsonMarshalListType(t *testing.T) {
	lt := ListType{
		SimpleType: SimpleType{
			KindList},
		ComponentType: SimpleType{
			KindInteger},
	}

	field := FieldDefinition{
		Type:     lt,
		Required: false,
	}

	wt := WorkItemType{
		Name: "first type",
		Fields: map[string]FieldDefinition{
			"aListType": field},
	}

	bytes, err := json.Marshal(wt)
	if err != nil {
		t.Error(err)
	}

	var readType WorkItemType
	json.Unmarshal(bytes, &readType)

	if !reflect.DeepEqual(wt, readType) {
		t.Errorf("not the same type %v, %v", wt, readType)
	}
}

func TestMarshalEnumType(t *testing.T) {
	et := EnumType{
		SimpleType: SimpleType{KindEnum},
		Values:     []interface{}{"open", "done", "closed"},
	}
	fd := FieldDefinition{
		Type:     et,
		Required: true,
	}

	wt := WorkItemType{
		Name: "first type",
		Fields: map[string]FieldDefinition{
			"aListType": fd},
	}
	bytes, err := json.Marshal(wt)
	if err != nil {
		t.Error(err)
	}

	var readType WorkItemType
	json.Unmarshal(bytes, &readType)
	if !reflect.DeepEqual(wt, readType) {
		t.Errorf("not the same type: %v, %v", readType, wt)
	}
}

func TestMarshalFieldDef(t *testing.T) {
	et := EnumType{
		SimpleType: SimpleType{KindEnum},
		Values:     []interface{}{"open", "done", "closed"},
	}
	fd := FieldDefinition{
		Type:     et,
		Required: true,
	}

	bytes, err := json.Marshal(fd)
	if err != nil {
		t.Error(err)
	}

	var readField FieldDefinition
	json.Unmarshal(bytes, &readField)
	if !reflect.DeepEqual(fd, readField) {
		t.Errorf("not the same : %v, %v", readField, fd)
	}
}

func TestMarshalArray(t *testing.T) {
	original := []interface{}{float64(1), float64(2), float64(3)}
	bytes, err := json.Marshal(original)
	if err != nil {
		t.Error(err)
	}
	var read []interface{}
	json.Unmarshal(bytes, &read)
	if !reflect.DeepEqual(original, read) {
		fmt.Printf("cap=[%d, %d], len=[%d, %d]\n", cap(original), cap(read), len(original), len(read))
		t.Error("not equal")
	}
}

func TestConvertFieldTypes(t *testing.T) {
	types := []FieldType{
		SimpleType{KindInteger},
		ListType{SimpleType{KindList}, SimpleType{KindString}},
		EnumType{SimpleType{KindEnum}, SimpleType{KindString}, []interface{}{"foo", "bar"}},
	}

	for _, theType := range types {
		t.Logf("testing type %v", theType)
		if err := testConvertFieldType(theType); err != nil {
			t.Error(err.Error())
		}
	}
}

func testConvertFieldType(original FieldType) error {
	converted := convertFieldTypeFromModels(original)
	reconverted, _ := convertFieldTypeToModels(converted)
	if !reflect.DeepEqual(original, reconverted) {
		return fmt.Errorf("reconverted should be %v, but is %v", original, reconverted)
	}
	return nil
}
