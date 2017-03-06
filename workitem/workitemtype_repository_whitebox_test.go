package workitem

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/space"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompatibleFields(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	a := FieldDefinition{
		Label:       "a",
		Description: "description for 'a'",
		Required:    true,
		Type: ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindString},
		},
	}
	b := FieldDefinition{
		Label:       "b",
		Description: "description for 'b'",
		Required:    true,
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
				Label:       "some list type",
				Description: "description for 'some list type'",
				Type: EnumType{
					BaseType:   SimpleType{KindString},
					SimpleType: SimpleType{KindEnum},
					Values:     []interface{}{"open", "done", "closed"},
				},
				Required: true,
			},
		},
		SpaceID: space.SystemSpace,
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
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	spaceSelfURL := rest.AbsoluteURL(reqLong, app.SpaceHref(space.SystemSpace.String()))
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
						Required:    true,
						Label:       "some list type",
						Description: "description for 'some list type'",
						Type: &app.FieldType{
							BaseType: &stString,
							Kind:     "enum",
							Values:   typeEnum,
						},
					},
				},
			},
			Relationships: &app.WorkItemTypeRelationships{
				Space: space.NewRelationSpaces(space.SystemSpace, spaceSelfURL),
			},
		},
	}

	result := convertTypeFromModels(reqLong, &a)

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
		SystemTitle:        {Type: &app.FieldType{Kind: "string"}, Required: true, Label: "l1", Description: "d1"},
		SystemDescription:  {Type: &app.FieldType{Kind: "string"}, Required: false, Label: "l2", Description: "d2"},
		SystemCreator:      {Type: &app.FieldType{Kind: "user"}, Required: true, Label: "l3", Description: "d3"},
		SystemRemoteItemID: {Type: &app.FieldType{Kind: "string"}, Required: false, Label: "l4", Description: "d4"},
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
			Required:    true,
			Label:       "l5",
			Description: "d5",
		},
	}

	expectedJSON := `{"system.creator":{"Required":true,"Label":"l3","Description":"d3","Type":{"Kind":"user"}},"system.description":{"Required":false,"Label":"l2","Description":"d2","Type":{"Kind":"string"}},"system.remote_item_id":{"Required":false,"Label":"l4","Description":"d4","Type":{"Kind":"string"}},"system.state":{"Required":true,"Label":"l5","Description":"d5","Type":{"Kind":"enum","BaseType":{"Kind":"string"},"Values":["new","open","in progress","resolved","closed"]}},"system.title":{"Required":true,"Label":"l1","Description":"d1","Type":{"Kind":"string"}}}`

	convertedFields, err := TEMPConvertFieldTypesToModel(newFields)
	jsonArray, err := json.Marshal(convertedFields)
	if err != nil {
		t.Fatal(err)
	}
	actualJSON := string(jsonArray[:])
	assert.Equal(t, expectedJSON, actualJSON)
}
