package workitem_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
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
