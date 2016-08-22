package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestListFieldDefMarshalling(t *testing.T) {
	def := FieldDefinition{
		Required: true,
		Type: ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{KindString},
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
