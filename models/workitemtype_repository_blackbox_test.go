package models_test

import (
	"golang.org/x/net/context"

	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

func TestCreateLoadWIT(t *testing.T) {
	resource.Require(t, resource.Database)

	repo := models.NewWorkItemTypeRepository(db)
	db2 := db.Unscoped().Delete(models.WorkItemType{Name: "foo.bar"})

	if db2.Error != nil {
		t.Fatalf("Could not setup test %s", db2.Error.Error())
		return
	}

	wit, err := repo.Create(context.Background(), nil, "foo.bar", map[string]app.FieldDefinition{
		"foo": app.FieldDefinition{
			Required: true,
			Type:     &app.FieldType{Kind: string(models.KindFloat)},
		},
	})
	assert.Nil(t, err)
	assert.NotNil(t, wit)
	defer db.Unscoped().Delete(models.WorkItemType{Name: "foo.bar"})

	wit3, err := repo.Create(context.Background(), nil, "foo.bar", map[string]app.FieldDefinition{})
	assert.IsType(t, models.BadParameterError{}, err)
	assert.Nil(t, wit3)

	wit2, err := repo.Load(context.Background(), "foo.bar")
	assert.Nil(t, err)
	assert.NotNil(t, wit2)
	field := wit2.Fields["foo"]
	assert.NotNil(t, field)
	assert.Equal(t, string(models.KindFloat), field.Type.Kind)
	assert.Equal(t, true, field.Required)
	assert.Nil(t, field.Type.ComponentType)
	assert.Nil(t, field.Type.BaseType)
	assert.Nil(t, field.Type.Values)
}
