package workitem_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

func testFieldDefinitionMarshalUnmarshal(t *testing.T, def workitem.FieldDefinition) {
	bytes, err := json.Marshal(def)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	t.Logf("bytes are %s", string(bytes))
	unmarshalled := workitem.FieldDefinition{}
	json.Unmarshal(bytes, &unmarshalled)

	if !reflect.DeepEqual(def, unmarshalled) {
		t.Errorf("field should be %v, but is %v", def, unmarshalled)
	}
}

func TestFieldDefinition_Marshalling(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	t.Run("simple boolean type", func(t *testing.T) {
		t.Parallel()
		def := workitem.FieldDefinition{
			Required:    true,
			Label:       "Salt",
			Description: "Put it in your soup",
			Type: workitem.SimpleType{
				Kind: workitem.KindBoolean,
			},
		}
		testFieldDefinitionMarshalUnmarshal(t, def)
	})

	t.Run("list type", func(t *testing.T) {
		t.Parallel()
		def := workitem.FieldDefinition{
			Required:    true,
			Label:       "Salt",
			Description: "Put it in your soup",
			Type: workitem.ListType{
				SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
				ComponentType: workitem.SimpleType{Kind: workitem.KindString},
			},
		}
		testFieldDefinitionMarshalUnmarshal(t, def)
	})

	t.Run("enum type", func(t *testing.T) {
		t.Parallel()
		def := workitem.FieldDefinition{
			Required:    true,
			Label:       "Salt",
			Description: "Put it in your soup",
			Type: workitem.EnumType{
				SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
				BaseType:   workitem.SimpleType{Kind: workitem.KindString},
				Values: []interface{}{
					"foo",
					"bar",
				},
			},
		}
		testFieldDefinitionMarshalUnmarshal(t, def)
	})
}

func TestFieldDefinition_IsRelational(t *testing.T) {
	// relational kinds
	require.True(t, workitem.KindLabel.IsRelational())
	require.True(t, workitem.KindArea.IsRelational())
	require.True(t, workitem.KindIteration.IsRelational())
	require.True(t, workitem.KindBoardColumn.IsRelational())
	require.True(t, workitem.KindUser.IsRelational())
	require.True(t, workitem.KindCodebase.IsRelational())
	// composite kinds
	require.False(t, workitem.KindList.IsRelational())
	require.False(t, workitem.KindEnum.IsRelational())
	// non-relational kinds
	require.False(t, workitem.KindString.IsRelational())
	require.False(t, workitem.KindInteger.IsRelational())
	require.False(t, workitem.KindInstant.IsRelational())
	require.False(t, workitem.KindFloat.IsRelational())
	require.False(t, workitem.KindBoolean.IsRelational())
	// random
	require.False(t, workitem.Kind(uuid.NewV4().String()).IsRelational())
}
