package category_test

import (
	"context"
	"testing"
	"time"

	"github.com/almighty/almighty-core/category"
	errs "github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type categoryRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo  category.Repository
	clean func()
	ctx   context.Context
}

func TestRunCategoryRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &categoryRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (test *categoryRepoBlackBoxTest) SetupTest() {
	test.repo = category.NewRepository(test.DB)
	test.clean = cleaner.DeleteCreatedEntities(test.DB)
}

func (test *categoryRepoBlackBoxTest) TearDownTest() {
	test.clean()
}

// TestCreateCategory tests that we can create a category
func (test *categoryRepoBlackBoxTest) TestCreateLoadCategory() {

	category := category.Category{
		Name: "Backlog",
	}

	test.T().Run("create and load (valid)", func(t *testing.T) {
		category1, err := test.repo.Create(test.ctx, &category) // Create
		require.Nil(test.T(), err)
		require.NotNil(test.T(), category1)
		require.NotNil(test.T(), category1.ID)
		require.NotNil(test.T(), category1.Name)

		category2, err := test.repo.LoadCategoryFromDB(test.ctx, category1.ID) // Load
		require.Nil(test.T(), err)
		require.NotNil(test.T(), category2)
		require.NotNil(test.T(), category2.ID)

		assert.Equal(test.T(), category1.ID, category2.ID)
		assert.Equal(test.T(), category1.Name, category2.Name)
	})

}

func (test *categoryRepoBlackBoxTest) TestCreateCategoryWithSameNameFail() {
	category1 := category.Category{
		Name: "categorySameName",
	}
	category2 := category.Category{
		Name: "categorySameName",
	}

	test.T().Run("unique category name violation", func(t *testing.T) {
		cat1, err := test.repo.Create(test.ctx, &category1) // Create
		require.Nil(test.T(), err)
		require.NotNil(test.T(), cat1)
		require.NotNil(test.T(), cat1.ID)
		require.NotNil(test.T(), cat1.Name)
		require.False(test.T(), cat1.CreatedAt.After(time.Now()), "Category was not created, CreatedAt after Now()")

		_, err = test.repo.Create(test.ctx, &category2) // Create
		require.NotNil(test.T(), err)

		// In case of unique constraint error, a BadParameterError is returned
		_, ok := errors.Cause(err).(errs.BadParameterError)
		assert.True(test.T(), ok)

	})
}

func (test *categoryRepoBlackBoxTest) TestListCategories() {
	test.T().Run("list categories", func(t *testing.T) {
		category1 := category.Category{
			Name: "Category1",
		}
		test.repo.Create(test.ctx, &category1)
		category2 := category.Category{
			Name: "Category2",
		}
		test.repo.Create(test.ctx, &category2)

		resultCategories, err := test.repo.List(test.ctx)

		require.Nil(test.T(), err)
		require.Equal(test.T(), 4, len(resultCategories)) // 2 category names are hard-coded + 2 category names are created in this test = total 4 categories
		assert.Equal(test.T(), category1.Name, resultCategories[2].Name)
		assert.Equal(test.T(), category2.Name, resultCategories[3].Name)
	})
}

func (test *categoryRepoBlackBoxTest) TestDoNotCreateCategoryWithMissingName() {
	category1 := category.Category{}

	test.T().Run("empty category name violation", func(t *testing.T) {
		cat1, err := test.repo.Create(test.ctx, &category1)
		require.NotNil(test.T(), err)
		require.Nil(test.T(), cat1)
	})
}
