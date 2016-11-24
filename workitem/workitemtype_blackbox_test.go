package workitem_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"time"

	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/workitem"
	"github.com/stretchr/testify/assert"
)

// TestJsonMarshalListType constructs a work item type, writes it to JSON (marshalling),
// and converts it back from JSON into a work item type (unmarshalling)
func TestJsonMarshalListType(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	lt := workitem.ListType{
		SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
		ComponentType: workitem.SimpleType{Kind: workitem.KindInteger},
	}

	field := workitem.FieldDefinition{
		Type:     lt,
		Required: false,
	}

	expectedWIT := workitem.WorkItemType{
		Name: "first type",
		Fields: map[string]workitem.FieldDefinition{
			"aListType": field},
	}

	bytes, err := json.Marshal(expectedWIT)
	if err != nil {
		t.Error(err)
	}

	var parsedWIT workitem.WorkItemType
	json.Unmarshal(bytes, &parsedWIT)

	if !expectedWIT.Equal(parsedWIT) {
		t.Errorf("Unmarshalled work item type: \n %v \n has not the same type as \"normal\" workitem type: \n %v \n", parsedWIT, expectedWIT)
	}
}

func TestMarshalEnumType(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	et := workitem.EnumType{
		SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
		Values:     []interface{}{"open", "done", "closed"},
	}
	fd := workitem.FieldDefinition{
		Type:     et,
		Required: true,
	}

	expectedWIT := workitem.WorkItemType{
		Name: "first type",
		Fields: map[string]workitem.FieldDefinition{
			"aListType": fd},
	}
	bytes, err := json.Marshal(expectedWIT)
	if err != nil {
		t.Error(err)
	}

	var parsedWIT workitem.WorkItemType
	json.Unmarshal(bytes, &parsedWIT)

	if !expectedWIT.Equal(parsedWIT) {
		t.Errorf("Unmarshalled work item type: \n %v \n has not the same type as \"normal\" workitem type: \n %v \n", parsedWIT, expectedWIT)
	}
}

func TestWorkItemType_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	fd := workitem.FieldDefinition{
		Type: workitem.EnumType{
			SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
			Values:     []interface{}{"open", "done", "closed"},
		},
		Required: true,
	}

	a := workitem.WorkItemType{
		Name: "foo",
		Fields: map[string]workitem.FieldDefinition{
			"aListType": fd,
		},
	}

	// Test types
	b := convert.DummyEqualer{}
	assert.False(t, a.Equal(b))

	// Test lifecycle
	c := a
	c.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
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
	f.Path = "foobar"
	assert.False(t, a.Equal(f))

	// Test field array length
	g := a
	g.Fields = map[string]workitem.FieldDefinition{}
	assert.False(t, a.Equal(g))

	// Test field key existence
	h := workitem.WorkItemType{
		Name: "foo",
		Fields: map[string]workitem.FieldDefinition{
			"bar": fd,
		},
	}
	assert.False(t, a.Equal(h))

	// Test field difference
	i := workitem.WorkItemType{
		Name: "foo",
		Fields: map[string]workitem.FieldDefinition{
			"aListType": workitem.FieldDefinition{
				Type: workitem.EnumType{
					SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
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

	et := workitem.EnumType{
		SimpleType: workitem.SimpleType{workitem.KindEnum},
		Values:     []interface{}{"open", "done", "closed"},
	}
	expectedFieldDef := workitem.FieldDefinition{
		Type:     et,
		Required: true,
	}

	bytes, err := json.Marshal(expectedFieldDef)
	if err != nil {
		t.Error(err)
	}

	var parsedFieldDef workitem.FieldDefinition
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

func TestWorkItemTypeIsTypeOrSubtypeOf(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// Test types and subtypes
	assert.True(t, workitem.WorkItemType{Name: "foo", Path: "/foo"}.IsTypeOrSubtypeOf("foo"))
	assert.True(t, workitem.WorkItemType{Name: "bar", Path: "/foo/bar"}.IsTypeOrSubtypeOf("foo"))
	assert.True(t, workitem.WorkItemType{Name: "bar", Path: "/foo/bar"}.IsTypeOrSubtypeOf("bar"))
	assert.True(t, workitem.WorkItemType{Name: "cake", Path: "/foo/bar/cake"}.IsTypeOrSubtypeOf("foo"))
	assert.True(t, workitem.WorkItemType{Name: "cake", Path: "/foo/bar/cake"}.IsTypeOrSubtypeOf("bar"))
	assert.True(t, workitem.WorkItemType{Name: "cake", Path: "/foo/bar/cake"}.IsTypeOrSubtypeOf("cake"))

	// Test we actually do return false sometimes
	assert.False(t, workitem.WorkItemType{Name: "cake", Path: "/foo/bar/cake"}.IsTypeOrSubtypeOf("fo"))
	assert.False(t, workitem.WorkItemType{Name: "foo", Path: "/foo"}.IsTypeOrSubtypeOf("fo"))

	// Test wrong argument with prefixed and trailing slashes
	assert.False(t, workitem.WorkItemType{Name: "foo", Path: "/foo"}.IsTypeOrSubtypeOf(""))
	assert.False(t, workitem.WorkItemType{Name: "foo", Path: "/foo"}.IsTypeOrSubtypeOf("/"))
	assert.True(t, workitem.WorkItemType{Name: "foo", Path: "/foo"}.IsTypeOrSubtypeOf("/foo"))
	assert.True(t, workitem.WorkItemType{Name: "foo", Path: "/foo"}.IsTypeOrSubtypeOf("/foo/"))
	assert.True(t, workitem.WorkItemType{Name: "foo", Path: "/foo"}.IsTypeOrSubtypeOf("foo/"))
}

func TestConvertFromModel(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	wi := models.WorkItem{}
	wit := models.WorkItemType{}
	result, err := wit.ConvertFromModel(wi)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Fields[models.SystemCreatedAt])
}
