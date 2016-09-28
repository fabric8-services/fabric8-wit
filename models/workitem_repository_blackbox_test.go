package models_test

import (
	"context"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/models"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateLoadDeleteListWI(t *testing.T) {
	doWithTransaction(t, func(ts *models.GormTransactionSupport) {
		witRepo := models.NewWorkItemTypeRepository(ts)
		repo := models.NewWorkItemRepository(ts, witRepo)

		wit, _ := witRepo.Create(context.Background(), nil, "foo.bar", map[string]app.FieldDefinition{
			"foo": app.FieldDefinition{
				Required: true,
				Type:     &app.FieldType{Kind: string(models.KindFloat)},
			},
			"bar": app.FieldDefinition{
				Required: false,
				Type:     &app.FieldType{Kind: string(models.KindString)},
			},
		})

		// missing mandatory field
		wi, err := repo.Create(context.Background(), wit.Name, map[string]interface{}{
			"bar": "abcd",
		})
		assert.IsType(t, models.BadParameterError{}, err)
		assert.Nil(t, wi)

		// wrong type of parameter
		wi, err = repo.Create(context.Background(), wit.Name, map[string]interface{}{
			"foo": "abcd",
		})
		assert.IsType(t, models.BadParameterError{}, err)
		assert.Nil(t, wi)

		wi, err = repo.Create(context.Background(), wit.Name, map[string]interface{}{
			"foo": 3.298,
		})

		assert.Nil(t, err)
		assert.NotNil(t, wi)
		assert.Equal(t, 3.298, wi.Fields["foo"])

		wi2, err := repo.Load(context.Background(), wi.ID)
		assert.Nil(t, err)
		assert.NotNil(t, wi)
		assert.Equal(t, wi, wi2)

		uuid := uuid.NewV4().String()

		wi2.Fields["bar"] = uuid
		wi2, err = repo.Save(context.Background(), *wi2)
		assert.Nil(t, err)
		wi2, _ = repo.Load(context.Background(), wi2.ID)
		assert.Equal(t, uuid, wi2.Fields["bar"])

		e1 := criteria.Equals(criteria.Field("bar"), criteria.Literal(uuid))
		e2 := criteria.Equals(criteria.Field("ID"), criteria.Literal(wi2.ID))

		list, err := repo.List(context.Background(), criteria.And(e1, e2), nil, nil)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(list))

		assert.Equal(t, wi2, list[0])

		err = repo.Delete(context.Background(), wi.ID)
		assert.Nil(t, err)
		wi3, err := repo.Load(context.Background(), wi.ID)
		assert.NotNil(t, err)
		assert.Nil(t, wi3)
	})
}
