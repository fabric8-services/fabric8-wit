package models_test

import (
	"context"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

func TestCreateLoadWIT(t *testing.T) {
	doWithTransaction(t, func(ts *models.GormTransactionSupport) {
		repo := models.NewWorkItemTypeRepository(ts)
		wit, err := repo.Load(context.Background(), "foo.bar")
		assert.IsType(t, models.NotFoundError{}, err)
		assert.Nil(t, wit)
		wit, err = repo.Create(context.Background(), nil, "foo.bar", map[string]app.FieldDefinition{
			"foo": app.FieldDefinition{
				Required: true,
				Type:     &app.FieldType{Kind: string(models.KindFloat)},
			},
		})
		assert.Nil(t, err)
		assert.NotNil(t, wit)

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
	})
}

func doWithTransaction(t *testing.T, todo func(ts *models.GormTransactionSupport)) {
	resource.Require(t, resource.Database)
	ts := models.NewGormTransactionSupport(db)
	if err := ts.Begin(); err != nil {
		panic(err.Error())
	}
	defer ts.Rollback()
	todo(ts)
}
