package models_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
)

func TestListFieldDefMarshalling(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	def := models.FieldDefinition{
		Required: true,
		Type: models.ListType{
			SimpleType:    models.SimpleType{Kind: models.KindList},
			ComponentType: models.SimpleType{Kind: models.KindString},
		},
	}
	bytes, err := json.Marshal(def)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	fmt.Printf("bytes are " + string(bytes))
	unmarshalled := models.FieldDefinition{}
	json.Unmarshal(bytes, &unmarshalled)

	if !reflect.DeepEqual(def, unmarshalled) {
		t.Errorf("field should be %v, but is %v", def, unmarshalled)
	}
}
