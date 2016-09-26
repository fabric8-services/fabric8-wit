package models_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"time"

	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

// TestJsonMarshalListType constructs a work item type, writes it to JSON (marshalling),
// and converts it back from JSON into a work item type (unmarshalling)
func TestJsonMarshalListType(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	lt := models.ListType{
		SimpleType:    models.SimpleType{Kind: models.KindList},
		ComponentType: models.SimpleType{Kind: models.KindInteger},
	}

	field := models.FieldDefinition{
		Type:     lt,
		Required: false,
	}

	expectedWIT := models.WorkItemType{
		Name: "first type",
		Fields: map[string]models.FieldDefinition{
			"aListType": field},
	}

	bytes, err := json.Marshal(expectedWIT)
	if err != nil {
		t.Error(err)
	}

	var parsedWIT models.WorkItemType
	json.Unmarshal(bytes, &parsedWIT)

	if !expectedWIT.Equal(parsedWIT) {
		t.Errorf("Unmarshalled work item type: \n %v \n has not the same type as \"normal\" workitem type: \n %v \n", parsedWIT, expectedWIT)
	}
}

func TestMarshalEnumType(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	et := models.EnumType{
		SimpleType: models.SimpleType{Kind: models.KindEnum},
		Values:     []interface{}{"open", "done", "closed"},
	}
	fd := models.FieldDefinition{
		Type:     et,
		Required: true,
	}

	expectedWIT := models.WorkItemType{
		Name: "first type",
		Fields: map[string]models.FieldDefinition{
			"aListType": fd},
	}
	bytes, err := json.Marshal(expectedWIT)
	if err != nil {
		t.Error(err)
	}

	var parsedWIT models.WorkItemType
	json.Unmarshal(bytes, &parsedWIT)

	if !expectedWIT.Equal(parsedWIT) {
		t.Errorf("Unmarshalled work item type: \n %v \n has not the same type as \"normal\" workitem type: \n %v \n", parsedWIT, expectedWIT)
	}
}

func TestWorkItemType_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	fd := models.FieldDefinition{
		Type: models.EnumType{
			SimpleType: models.SimpleType{Kind: models.KindEnum},
			Values:     []interface{}{"open", "done", "closed"},
		},
		Required: true,
	}

	a := models.WorkItemType{
		Name: "foo",
		Fields: map[string]models.FieldDefinition{
			"aListType": fd,
		},
	}

	// Test types
	b := models.DummyEqualer{}
	assert.False(t, a.Equal(b))

	// Test lifecycle
	c := a
	c.Lifecycle = models.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
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
	g.Fields = map[string]models.FieldDefinition{}
	assert.False(t, a.Equal(g))

	// Test field key existence
	h := models.WorkItemType{
		Name: "foo",
		Fields: map[string]models.FieldDefinition{
			"bar": fd,
		},
	}
	assert.False(t, a.Equal(h))

	// Test field difference
	i := models.WorkItemType{
		Name: "foo",
		Fields: map[string]models.FieldDefinition{
			"aListType": models.FieldDefinition{
				Type: models.EnumType{
					SimpleType: models.SimpleType{Kind: models.KindEnum},
					Values:     []interface{}{"open", "done", "closed"},
				},
				Required: false,
			},
		},
	}
	assert.False(t, a.Equal(i))

}

func TestMarshalFieldDef(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	et := models.EnumType{
		SimpleType: models.SimpleType{models.KindEnum},
		Values:     []interface{}{"open", "done", "closed"},
	}
	expectedFieldDef := models.FieldDefinition{
		Type:     et,
		Required: true,
	}

	bytes, err := json.Marshal(expectedFieldDef)
	if err != nil {
		t.Error(err)
	}

	var parsedFieldDef models.FieldDefinition
	json.Unmarshal(bytes, &parsedFieldDef)
	if !expectedFieldDef.Equal(parsedFieldDef) {
		t.Errorf("Unmarshalled field definition: \n %v \n has not the same type as \"normal\" field definition: \n %v \n", parsedFieldDef, expectedFieldDef)
	}
}

func TestMarshalArray(t *testing.T) {
	t.Parallel()
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
