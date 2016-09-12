package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
	"time"
)

// TestJsonMarshalListType constructs a work item type, writes it to JSON (marshalling),
// and converts it back from JSON into a work item type (unmarshalling)
func TestJsonMarshalListType(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	lt := ListType{
		SimpleType:    SimpleType{KindList},
		ComponentType: SimpleType{KindInteger},
	}

	field := FieldDefinition{
		Type:     lt,
		Required: false,
	}

	expectedWIT := WorkItemType{
		Name: "first type",
		Fields: map[string]FieldDefinition{
			"aListType": field},
	}

	bytes, err := json.Marshal(expectedWIT)
	if err != nil {
		t.Error(err)
	}

	var parsedWIT WorkItemType
	json.Unmarshal(bytes, &parsedWIT)

	if !expectedWIT.Equal(parsedWIT) {
		t.Errorf("Unmarshalled work item type: \n %v \n has not the same type as \"normal\" workitem type: \n %v \n", parsedWIT, expectedWIT)
	}
}

func TestMarshalEnumType(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	et := EnumType{
		SimpleType: SimpleType{KindEnum},
		Values:     []interface{}{"open", "done", "closed"},
	}
	fd := FieldDefinition{
		Type:     et,
		Required: true,
	}

	expectedWIT := WorkItemType{
		Name: "first type",
		Fields: map[string]FieldDefinition{
			"aListType": fd},
	}
	bytes, err := json.Marshal(expectedWIT)
	if err != nil {
		t.Error(err)
	}

	var parsedWIT WorkItemType
	json.Unmarshal(bytes, &parsedWIT)

	if !expectedWIT.Equal(parsedWIT) {
		t.Errorf("Unmarshalled work item type: \n %v \n has not the same type as \"normal\" workitem type: \n %v \n", parsedWIT, expectedWIT)
	}
}

func TestWorkItemTypeNotEqual(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	fd := FieldDefinition{
		Type: EnumType{
			SimpleType: SimpleType{KindEnum},
			Values:     []interface{}{"open", "done", "closed"},
		},
		Required: true,
	}

	a := WorkItemType{
		Name: "foo",
		Fields: map[string]FieldDefinition{
			"aListType": fd,
		},
	}

	// Test types
	b := DummyEqualer{}
	assert.False(t, a.Equal(b))

	// Test lifecycle
	c := a
	c.Lifecycle = Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	assert.False(t, a.Equal(c))

	// Test version
	d := a
	d.Version += 1
	assert.False(t, a.Equal(d))

	// Test version
	e := a
	e.Name = "bar"
	assert.False(t, a.Equal(e))

	// Test parent path
	f := a
	f.ParentPath = "foobar"
	assert.False(t, a.Equal(f))

	// Test field array length
	g := a
	g.Fields = map[string]FieldDefinition{}
	assert.False(t, a.Equal(g))

	// Test field key existence
	h := WorkItemType{
		Name: "foo",
		Fields: map[string]FieldDefinition{
			"bar": fd,
		},
	}
	assert.False(t, a.Equal(h))

	// Test field difference
	i := WorkItemType{
		Name: "foo",
		Fields: map[string]FieldDefinition{
			"aListType": FieldDefinition{
				Type: EnumType{
					SimpleType: SimpleType{KindEnum},
					Values:     []interface{}{"open", "done", "closed"},
				},
				Required: false,
			},
		},
	}
	assert.False(t, a.Equal(i))

}

func TestMarshalFieldDef(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	et := EnumType{
		SimpleType: SimpleType{KindEnum},
		Values:     []interface{}{"open", "done", "closed"},
	}
	expectedFieldDef := FieldDefinition{
		Type:     et,
		Required: true,
	}

	bytes, err := json.Marshal(expectedFieldDef)
	if err != nil {
		t.Error(err)
	}

	var parsedFieldDef FieldDefinition
	json.Unmarshal(bytes, &parsedFieldDef)
	if !expectedFieldDef.Equal(parsedFieldDef) {
		t.Errorf("Unmarshalled field definition: \n %v \n has not the same type as \"normal\" field definition: \n %v \n", parsedFieldDef, expectedFieldDef)
	}
}

func TestMarshalArray(t *testing.T) {
	resource.Require(t, resource.UnitTest)

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
	resource.Require(t, resource.UnitTest)
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
