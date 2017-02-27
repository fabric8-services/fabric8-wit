package workitem

import (
	"encoding/json"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/resource"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompatibleFields(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	a := FieldDefinition{
		Required: true,
		Type: ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindString},
		},
	}
	b := FieldDefinition{
		Required: true,
		Type: ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindString},
		},
	}
	assert.True(t, compatibleFields(a, b))
}

func TestConvertTypeFromModels(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	//------------------------------
	// Work item type in model space
	//------------------------------

	descFoo := "Description of 'foo'"
	id := uuid.NewV4()
	a := WorkItemType{
		ID:          id,
		Name:        "foo",
		Description: &descFoo,
		Version:     42,
		Path:        "something",
		Fields: map[string]FieldDefinition{
			"aListType": {
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
	expected := app.WorkItemTypeSingle{
		Data: &app.WorkItemTypeData{
			ID:   &id,
			Type: "workitemtypes",
			Attributes: &app.WorkItemTypeAttributes{
				Name:        "foo",
				Description: &descFoo,
				Version:     42,
				Fields: map[string]*app.FieldDefinition{
					"aListType": {
						Required: true,
						Type: &app.FieldType{
							BaseType: &stString,
							Kind:     "enum",
							Values:   typeEnum,
						},
					},
				},
			},
		},
	}

	result := convertTypeFromModels(&a)

	require.NotNil(t, result.ID)
	assert.True(t, uuid.Equal(*expected.Data.ID, *result.ID))
	assert.Equal(t, expected.Data.Attributes.Version, result.Attributes.Version)
	assert.Equal(t, expected.Data.Attributes.Name, result.Attributes.Name)
	require.NotNil(t, result.Attributes.Description)
	assert.Equal(t, *expected.Data.Attributes.Description, *result.Attributes.Description)
	assert.Len(t, result.Attributes.Fields, len(expected.Data.Attributes.Fields))
	assert.Equal(t, expected.Data.Attributes.Fields, result.Attributes.Fields)
}

func TestConvertAnyToKind(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	_, err := convertAnyToKind(1234)
	assert.NotNil(t, err)
}

func TestConvertStringToKind(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	_, err := convertStringToKind("DefinitivelyNotAKind")
	assert.NotNil(t, err)
}

func TestConvertFieldTypeToModels(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// Create an enumeration of animal names
	typeStrings := []string{"open", "done", "closed"}

	// Convert string slice to slice of interface{} in O(n) time.
	typeEnum := make([]interface{}, len(typeStrings))
	for i := range typeStrings {
		typeEnum[i] = typeStrings[i]
	}

	// Create the type for "animal-type" field based on the enum above
	stString := "string"

	_ = &app.FieldType{
		BaseType: &stString,
		Kind:     "DefinitivelyNotAType",
		Values:   typeEnum,
	}
	_, err := convertFieldTypeToModels(app.FieldType{Kind: "DefinitivelyNotAType"})
	assert.NotNil(t, err)
}

// TestTempConvertFieldsToModels is a temporary function to workaround the access
// issue from migrations.go  - hence keeping it in *_whitebox_test.go

func TestTempConvertFieldsToModels(t *testing.T) {

	resource.Require(t, resource.UnitTest)
	stString := "string"

	newFields := map[string]app.FieldDefinition{
		SystemTitle:        {Type: &app.FieldType{Kind: "string"}, Required: true},
		SystemDescription:  {Type: &app.FieldType{Kind: "string"}, Required: false},
		SystemCreator:      {Type: &app.FieldType{Kind: "user"}, Required: true},
		SystemRemoteItemID: {Type: &app.FieldType{Kind: "string"}, Required: false},
		SystemState: {
			Type: &app.FieldType{
				BaseType: &stString,
				Kind:     "enum",
				Values: []interface{}{
					SystemStateNew,
					SystemStateOpen,
					SystemStateInProgress,
					SystemStateResolved,
					SystemStateClosed,
				},
			},
			Required: true,
		},
	}

	expectedJSON := `{"system.creator":{"Required":true,"Type":{"Kind":"user"}},"system.description":{"Required":false,"Type":{"Kind":"string"}},"system.remote_item_id":{"Required":false,"Type":{"Kind":"string"}},"system.state":{"Required":true,"Type":{"Kind":"enum","BaseType":{"Kind":"string"},"Values":["new","open","in progress","resolved","closed"]}},"system.title":{"Required":true,"Type":{"Kind":"string"}}}`

	convertedFields, err := TEMPConvertFieldTypesToModel(newFields)
	jsonArray, err := json.Marshal(convertedFields)
	if err != nil {
		t.Fatal(err)
	}
	actualJSON := string(jsonArray[:])
	assert.Equal(t, expectedJSON, actualJSON)
}
