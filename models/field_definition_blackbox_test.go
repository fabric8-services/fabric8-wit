package models_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	. "github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
)

func TestListFieldDefMarshalling(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	def := FieldDefinition{
		Required: true,
		Type: ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindString},
		},
	}
	bytes, err := json.Marshal(def)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	fmt.Printf("bytes are " + string(bytes))
	unmarshalled := FieldDefinition{}
	json.Unmarshal(bytes, &unmarshalled)

	if !reflect.DeepEqual(def, unmarshalled) {
		t.Errorf("field should be %v, but is %v", def, unmarshalled)
	}
}
