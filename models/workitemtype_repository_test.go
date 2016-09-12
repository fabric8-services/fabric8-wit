package models

import (
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

func TestCompatibleFields(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	a := FieldDefinition{
		Required: true,
		Type: ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{KindString},
		},
	}
	b := FieldDefinition{
		Required: true,
		Type: ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{KindString},
		},
	}
	assert.True(t, compatibleFields(a, b))
}

func TestConvertTypeFromModels(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	//------------------------------
	// Work item type in model space
	//------------------------------

	a := WorkItemType{
		Name:       "foo",
		Version:    42,
		ParentPath: "something",
		Fields: map[string]FieldDefinition{
			"aListType": FieldDefinition{
				Type: EnumType{
					BaseType:   SimpleType{KindString},
					SimpleType: SimpleType{KindEnum},
					Values:     []interface{}{"open", "done", "closed"},
				},
				Required: true,
			},
		},
	}

	//----------------------------
	// Work item type in app space
	//----------------------------

	// Create an enumeration of animal names
	typeStrings := []string{"open", "done", "closed"}

	// Convert string slice to slice of interface{} in O(n) time.
	typeEnum := make([]interface{}, len(typeStrings))
	for i := range typeStrings {
		typeEnum[i] = typeStrings[i]
	}

	// Create the type for "animal-type" field based on the enum above
	stString := "string"
	expected := app.WorkItemType{
		Name:    "foo",
		Version: 42,
		Fields: map[string]*app.FieldDefinition{
			"aListType": &app.FieldDefinition{
				Required: true,
				Type: &app.FieldType{
					BaseType:      &stString,
					Kind:          "enum",
					Values:        typeEnum,
				},
			},
		},
	}

	result := convertTypeFromModels(&a)

	assert.Equal(t, expected.Name, result.Name)
	assert.Equal(t, expected.Version, result.Version)
	assert.Len(t, result.Fields, len(expected.Fields))
	assert.Equal(t, expected.Fields, result.Fields)
}
