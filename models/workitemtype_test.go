package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestMarshalSimple(t *testing.T) {
	lt := ListType{SimpleType{8}, SimpleType{1}}
	var foo FieldTypes = FieldTypes{"x": lt}

	bytes, err := json.Marshal(foo)
	if err != nil {
		t.Error(err)
	}
	var bar FieldTypes
	err = json.Unmarshal(bytes, &bar)
	if err != nil {
		t.Error(fmt.Sprintf("problem unmarshaling:%s\n", err))
	}
	if !reflect.DeepEqual(foo, bar) {
		t.Error(fmt.Sprintf("not equal: %v, %v\n", foo, bar))
	}
}

func TestJsonMarshalling(t *testing.T) {
	lt := ListType{
		SimpleType: SimpleType{
			List},
		ComponentType: SimpleType{
			Integer},
	}

	wt := WorkItemType{
		Id:      1,
		Name:    "first type",
		Version: 1,
		Fields: FieldTypes{
			"aListType": lt},
	}

	bytes, err := json.Marshal(wt)
	if err != nil {
		t.Error(err)
	}

	var readType WorkItemType
	json.Unmarshal(bytes, &readType)

	if !reflect.DeepEqual(wt, readType) {
		t.Error("not the same type")
	}
}

func TestMarshalEnumType(t *testing.T) {
	et := EnumType{
		SimpleType: SimpleType{Enum},
		Values:     []interface{}{"open", "done", "closed"},
	}

	wt := WorkItemType{
		Id:      1,
		Name:    "first type",
		Version: 1,
		Fields: FieldTypes{
			"aListType": et},
	}
	bytes, err := json.Marshal(wt)
	if err != nil {
		t.Error(err)
	}

	var readType WorkItemType
	json.Unmarshal(bytes, &readType)
	if !reflect.DeepEqual(wt, readType) {
		t.Error("not the same type")
	}
}
