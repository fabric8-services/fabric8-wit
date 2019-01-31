package workitem_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	desc := "some description"
	expectedWIT := workitem.WorkItemType{
		Name:        "first type",
		Description: &desc,
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
func TestWorkItemType_EqualAndEqualValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// given
	fd := workitem.FieldDefinition{
		Type: workitem.EnumType{
			SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
			Values:     []interface{}{"open", "done", "closed"},
		},
		Required: true,
	}
	desc := "some description"
	a := workitem.WorkItemType{
		SpaceTemplateID: uuid.NewV4(),
		Name:            "foo",
		Description:     &desc,
		Icon:            "fa-bug",
		Fields: map[string]workitem.FieldDefinition{
			"aListType": fd,
		},
		ChildTypeIDs: []uuid.UUID{uuid.NewV4(), uuid.NewV4()},
		CanConstruct: false,
	}
	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		b := a
		assert.True(t, a.Equal(b))
		assert.True(t, a.EqualValue(b))
	})
	t.Run("space template ID", func(t *testing.T) {
		t.Parallel()
		b := a
		b.SpaceTemplateID = uuid.NewV4()
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})
	t.Run("type", func(t *testing.T) {
		t.Parallel()
		b := convert.DummyEqualer{}
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})
	t.Run("lifecycle", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
		assert.False(t, a.Equal(b))
		assert.True(t, a.EqualValue(b))
	})
	t.Run("version", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Version += 1
		assert.False(t, a.Equal(b))
		assert.True(t, a.EqualValue(b))
	})
	t.Run("name", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Name = "bar"
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})
	t.Run("extends", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Extends = uuid.NewV4()
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})
	t.Run("parent path", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Path = "foobar"
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})
	t.Run("ChildTypeIDs", func(t *testing.T) {
		t.Parallel()
		b := a
		// different IDs
		b.ChildTypeIDs = []uuid.UUID{uuid.NewV4(), uuid.NewV4()}
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
		// different length
		b.ChildTypeIDs = []uuid.UUID{uuid.NewV4(), uuid.NewV4(), uuid.NewV4()}
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})
	t.Run("field array length", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Fields = map[string]workitem.FieldDefinition{}
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})
	t.Run("field key existence", func(t *testing.T) {
		t.Parallel()
		b := workitem.WorkItemType{
			Name: "foo",
			Fields: map[string]workitem.FieldDefinition{
				"bar": fd,
			},
		}
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})
	t.Run("field difference", func(t *testing.T) {
		t.Parallel()
		b := workitem.WorkItemType{
			Name:        "foo",
			Description: &desc,
			Fields: map[string]workitem.FieldDefinition{
				"aListType": {
					Type: workitem.EnumType{
						SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
						Values:     []interface{}{"open", "done", "closed"},
					},
					Required: false,
				},
			},
		}
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})
	t.Run("description", func(t *testing.T) {
		t.Parallel()
		t.Run("different value", func(t *testing.T) {
			b := a
			b.Description = ptr.String("some other description")
			assert.False(t, a.Equal(b))
			assert.False(t, a.EqualValue(b))
		})
		t.Run("different pointer but same value", func(t *testing.T) {
			b := a
			desc2 := desc
			b.Description = &desc2
			assert.True(t, a.Equal(b))
			assert.True(t, a.EqualValue(b))
		})
	})
	t.Run("icon", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Icon = "fa-cog"
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})
	t.Run("can construct", func(t *testing.T) {
		t.Parallel()
		b := a
		b.CanConstruct = true
		assert.False(t, a.Equal(b))
		assert.False(t, a.EqualValue(b))
	})
}
func TestMarshalFieldDef(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	et := workitem.EnumType{
		SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
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

	// Prepare some UUIDs for use in tests
	id1 := uuid.FromStringOrNil("68e90fa9-dba1-4448-99a4-ae70fb2b45f9")
	id2 := uuid.FromStringOrNil("aa6ef831-36db-4e99-9e33-6f793472f769")
	id3 := uuid.FromStringOrNil("3566837f-aa98-4792-bce1-75c995d4e98c")
	id4 := uuid.FromStringOrNil("c88e6669-53f9-4aa1-be98-877b850daf88")
	// Prepare the ltree nodes based on the IDs
	node1 := workitem.LtreeSafeID(id1)
	node2 := workitem.LtreeSafeID(id2)
	node3 := workitem.LtreeSafeID(id3)

	// Test types and subtypes
	assert.True(t, workitem.WorkItemType{ID: id1, Path: node1}.IsTypeOrSubtypeOf(id1))
	assert.True(t, workitem.WorkItemType{ID: id2, Path: node1 + "." + node2}.IsTypeOrSubtypeOf(id1))
	assert.True(t, workitem.WorkItemType{ID: id2, Path: node1 + "." + node2}.IsTypeOrSubtypeOf(id2))
	assert.True(t, workitem.WorkItemType{ID: id3, Path: node1 + "." + node2 + "." + node3}.IsTypeOrSubtypeOf(id1))
	assert.True(t, workitem.WorkItemType{ID: id3, Path: node1 + "." + node2 + "." + node3}.IsTypeOrSubtypeOf(id2))
	assert.True(t, workitem.WorkItemType{ID: id3, Path: node1 + "." + node2 + "." + node3}.IsTypeOrSubtypeOf(id3))

	// Test we actually do return false someNodees
	assert.False(t, workitem.WorkItemType{ID: id3, Path: node1 + "." + node2 + "." + node3}.IsTypeOrSubtypeOf(id4))
	assert.False(t, workitem.WorkItemType{ID: id1, Path: node1}.IsTypeOrSubtypeOf(id4))
}

// TestConstants exists in order to avoid accidental changes to constants
func TestConstants(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	require.Equal(t, "version", workitem.SystemVersion)
	require.Equal(t, "system.remote_item_id", workitem.SystemRemoteItemID)
	require.Equal(t, "system.number", workitem.SystemNumber)
	require.Equal(t, "system.title", workitem.SystemTitle)
	require.Equal(t, "system.description", workitem.SystemDescription)
	require.Equal(t, "system.description.markup", workitem.SystemDescriptionMarkup)
	require.Equal(t, "system.description.rendered", workitem.SystemDescriptionRendered)
	require.Equal(t, "system.state", workitem.SystemState)
	require.Equal(t, "system.assignees", workitem.SystemAssignees)
	require.Equal(t, "system.creator", workitem.SystemCreator)
	require.Equal(t, "system.created_at", workitem.SystemCreatedAt)
	require.Equal(t, "system.updated_at", workitem.SystemUpdatedAt)
	require.Equal(t, "system.order", workitem.SystemOrder)
	require.Equal(t, "system.iteration", workitem.SystemIteration)
	require.Equal(t, "system.area", workitem.SystemArea)
	require.Equal(t, "system.codebase", workitem.SystemCodebase)
	require.Equal(t, "system.labels", workitem.SystemLabels)
	require.Equal(t, "system.boardcolumns", workitem.SystemBoardcolumns)
	require.Equal(t, "Board", workitem.SystemBoard)
	require.Equal(t, "open", workitem.SystemStateOpen)
	require.Equal(t, "new", workitem.SystemStateNew)
	require.Equal(t, "in progress", workitem.SystemStateInProgress)
	require.Equal(t, "resolved", workitem.SystemStateResolved)
	require.Equal(t, "closed", workitem.SystemStateClosed)
}
