package models

import (
	"encoding/json"
	"reflect"
	"testing"
	"fmt"
)

func TestJsonMarshalListType(t *testing.T) {
	lt := ListType{
		SimpleType: SimpleType{
			List},
		ComponentType: SimpleType{
			Integer},
	}

	field := FieldDefinition{
		Type:     lt,
		Required: false,
	}

	wt := WorkItemType{
		Id:      1,
		Name:    "first type",
		Version: 1,
		Fields: FieldDefinitions{
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
		SimpleType: SimpleType{Enum},
		Values:     []interface{}{"open", "done", "closed"},
	}
	fd := FieldDefinition{
		Type:     et,
		Required: true,
	}

	wt := WorkItemType{
		Id:      1,
		Name:    "first type",
		Version: 1,
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
		t.Error(fmt.Sprintf("not the same type: %v, %v", readType, wt))
	}
}

func TestMarshalFieldDef(t *testing.T) {
	et := EnumType{
		SimpleType: SimpleType{Enum},
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
		t.Error(fmt.Sprintf("not the same : %v, %v", readField, fd))
	}
}

func TestMarshalRawEnum(t *testing.T) {
	ret := rawEnumType{
		BaseType: SimpleType{Kind: Integer},
		Values:   []interface{}{float64(2), float64(4), float64(4)},
	}

	bytes, err := json.Marshal(ret)
	if err != nil {
		t.Error(err)
	}

	var readField rawEnumType
	json.Unmarshal(bytes, &readField)

	if !reflect.DeepEqual(readField.Values, ret.Values) {
		t.Error("values not equal\n")
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
