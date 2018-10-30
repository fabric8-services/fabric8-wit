package controller

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertTypeFromModel(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// given

	//------------------------------
	// Work item type in model space
	//------------------------------

	descFoo := "Description of 'foo'"
	id := uuid.NewV4()
	createdAt := time.Now().Add(-1 * time.Hour).UTC()
	updatedAt := time.Now().UTC()
	a := workitem.WorkItemType{
		ID: id,
		Lifecycle: gormsupport.Lifecycle{
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:        "foo",
		Description: &descFoo,
		Version:     42,
		Path:        "something",
		Fields: map[string]workitem.FieldDefinition{
			"aListType": {
				Label:       "some list type",
				Description: "description for 'some list type'",
				Type: workitem.EnumType{
					BaseType:   workitem.SimpleType{Kind: workitem.KindString},
					SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
					Values:     []interface{}{"open", "done", "closed"},
				},
				Required: true,
			},
		},
		SpaceTemplateID: spacetemplate.SystemLegacyTemplateID,
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
	reqLong := &http.Request{Host: "api.service.domain.org"}
	spaceTemplateID := spacetemplate.SystemLegacyTemplateID
	spaceTemplateSelfURL := rest.AbsoluteURL(reqLong, app.SpaceTemplateHref(spaceTemplateID.String()))
	version := 42
	expected := app.WorkItemTypeSingle{
		Data: &app.WorkItemTypeData{
			ID:   &id,
			Type: "workitemtypes",
			Attributes: &app.WorkItemTypeAttributes{
				Name:        "foo",
				Description: &descFoo,
				Version:     &version,
				CreatedAt:   &createdAt,
				UpdatedAt:   &updatedAt,
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
				SpaceTemplate: app.NewSpaceTemplateRelation(spaceTemplateID, spaceTemplateSelfURL),
			},
		},
	}
	// when
	result := ConvertWorkItemTypeFromModel(reqLong, &a)
	// then
	require.NotNil(t, result.ID)
	assert.True(t, uuid.Equal(*expected.Data.ID, *result.ID))
	assert.Equal(t, expected.Data.Attributes.Version, result.Attributes.Version)
	assert.Equal(t, expected.Data.Attributes.CreatedAt, result.Attributes.CreatedAt)
	assert.Equal(t, expected.Data.Attributes.UpdatedAt, result.Attributes.UpdatedAt)
	assert.Equal(t, expected.Data.Attributes.Name, result.Attributes.Name)
	require.NotNil(t, result.Attributes.Description)
	assert.Equal(t, *expected.Data.Attributes.Description, *result.Attributes.Description)
	assert.Len(t, result.Attributes.Fields, len(expected.Data.Attributes.Fields))
	assert.Equal(t, expected.Data.Attributes.Fields, result.Attributes.Fields)
}

func TestConvertTypeFromModelFieldNames(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	//------------------------------
	// Work item type in model space
	//------------------------------

	id := uuid.NewV4()
	createdAt := time.Now().Add(-1 * time.Hour).UTC()
	updatedAt := time.Now().UTC()
	a := workitem.WorkItemType{
		ID: id,
		Lifecycle: gormsupport.Lifecycle{
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name: "foo",
		Fields: map[string]workitem.FieldDefinition{
			"system_title": {
				Type: workitem.SimpleType{
					Kind: workitem.KindString,
				},
			},
		},
	}

	reqLong := &http.Request{Host: "api.service.domain.org"}
	//----------------------------
	// Work item type in app space
	//----------------------------
	expected := app.WorkItemTypeSingle{
		Data: &app.WorkItemTypeData{
			ID:   &id,
			Type: "workitemtypes",
			Attributes: &app.WorkItemTypeAttributes{
				Name:      "foo",
				CreatedAt: &createdAt,
				UpdatedAt: &updatedAt,
				Fields: map[string]*app.FieldDefinition{
					"system.title": {
						Type: &app.FieldType{
							Kind: "string",
						},
					},
					"system_title": {
						Type: &app.FieldType{
							Kind: "string",
						},
					},
				},
			},
		},
	}
	// when
	result := ConvertWorkItemTypeFromModel(reqLong, &a)
	require.NotNil(t, result.ID)
	assert.True(t, uuid.Equal(*expected.Data.ID, *result.ID))
	assert.Equal(t, expected.Data.Attributes.CreatedAt, result.Attributes.CreatedAt)
	assert.Equal(t, expected.Data.Attributes.UpdatedAt, result.Attributes.UpdatedAt)
	assert.Equal(t, expected.Data.Attributes.Name, result.Attributes.Name)
	assert.Len(t, result.Attributes.Fields, len(expected.Data.Attributes.Fields))
	assert.Equal(t, expected.Data.Attributes.Fields, result.Attributes.Fields)
}

func TestConvertFieldTypes(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	types := []workitem.FieldType{
		workitem.SimpleType{Kind: workitem.KindInteger},
		workitem.ListType{
			SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
			ComponentType: workitem.SimpleType{Kind: workitem.KindString},
		},
		workitem.EnumType{
			SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
			BaseType:   workitem.SimpleType{Kind: workitem.KindString},
			Values:     []interface{}{"foo", "bar"},
		},
	}

	for _, theType := range types {
		t.Logf("testing type %v", theType)
		if err := testConvertFieldType(theType); err != nil {
			t.Error(err.Error())
		}
	}
}

func testConvertFieldType(original workitem.FieldType) error {
	converted := ConvertFieldTypeFromModel(original)
	reconverted, _ := ConvertFieldTypeToModel(converted)
	if !reflect.DeepEqual(original, reconverted) {
		return fmt.Errorf("reconverted should be %v, but is %v", original, reconverted)
	}
	return nil
}

func TestConvertFieldTypeToModel(t *testing.T) {
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
	_, err := ConvertFieldTypeToModel(app.FieldType{Kind: "DefinitivelyNotAType"})
	assert.NotNil(t, err)
}
