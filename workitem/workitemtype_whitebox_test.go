package workitem

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertWorkItemStorageToModelEmptyAssigneesAndLabels(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	lt := ListType{
		SimpleType:    SimpleType{Kind: KindList},
		ComponentType: SimpleType{Kind: KindString},
	}

	field := FieldDefinition{
		Type:     lt,
		Required: false,
	}

	wit := WorkItemType{
		Name: "Empty Values",
		Fields: map[string]FieldDefinition{
			SystemAssignees: field,
			SystemLabels:    field,
		}}

	wis := WorkItemStorage{Fields: map[string]interface{}{SystemAssignees: nil, SystemLabels: nil}}
	wi, err := wit.ConvertWorkItemStorageToModel(wis)
	require.Nil(t, err)
	assert.Equal(t, map[string]interface{}{}, wi.Fields)

	wis = WorkItemStorage{Fields: map[string]interface{}{SystemAssignees: []interface{}{"a", "b"}, SystemLabels: nil}}
	wi, err = wit.ConvertWorkItemStorageToModel(wis)
	require.Nil(t, err)
	assert.Equal(t, map[string]interface{}{"system.assignees": []interface{}{"a", "b"}, "system.order": float64(0)}, wi.Fields)

	wis = WorkItemStorage{Fields: map[string]interface{}{SystemAssignees: nil, SystemLabels: []interface{}{"a", "b"}}}
	wi, err = wit.ConvertWorkItemStorageToModel(wis)
	require.Nil(t, err)
	assert.Equal(t, map[string]interface{}{"system.labels": []interface{}{"a", "b"}, "system.order": float64(0)}, wi.Fields)
}
