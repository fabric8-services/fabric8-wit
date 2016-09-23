package models

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/almighty/almighty-core/resource"
)

func TestConvertFieldTypes(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	types := []FieldType{
		SimpleType{Kind: KindInteger},
		ListType{SimpleType{Kind: KindList}, SimpleType{Kind: KindString}},
		EnumType{SimpleType{Kind: KindEnum}, SimpleType{Kind: KindString}, []interface{}{"foo", "bar"}},
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
