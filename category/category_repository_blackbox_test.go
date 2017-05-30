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
	uuid "github.com/satori/go.uuid"
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

// TestCreateLoadValidCategory tests create and load valid category
func (test *categoryRepoBlackBoxTest) TestCreateLoadValidCategory() {
	category1 := category.Category{
		Name: "Backlog",
	}

	test.T().Run("create and load (valid)", func(t *testing.T) {
		result1, err := test.repo.Create(test.ctx, &category1) // Create
		require.Nil(test.T(), err)
		require.NotNil(test.T(), result1)
		require.NotNil(test.T(), result1.ID)
		require.NotNil(test.T(), result1.Name)

		result2, err := test.repo.LoadCategoryFromDB(test.ctx, result1.ID) // Load
		require.Nil(test.T(), err)
		require.NotNil(test.T(), result2)
		require.NotNil(test.T(), result2.ID)

		assert.Equal(test.T(), result1.ID, result2.ID)
		assert.Equal(test.T(), result1.Name, result2.Name)
	})
}

// TestCreateLoadInvalidCategory tests create and load invalid category
func (test *categoryRepoBlackBoxTest) TestCreateLoadInvalidCategory() {
	category1 := category.Category{
		Name: "Issues1",
	}
	category2 := category.Category{
		Name: "Issues2",
	}
	test.T().Run("create and load (invalid)", func(t *testing.T) {
		result1, err := test.repo.Create(test.ctx, &category1) // Create
		require.Nil(test.T(), err)
		require.NotNil(test.T(), result1)
		require.NotNil(test.T(), result1.ID)
		require.NotNil(test.T(), result1.Name)

		result2, err := test.repo.Create(test.ctx, &category2) // Create
		require.Nil(test.T(), err)
		require.NotNil(test.T(), result2)
		require.NotNil(test.T(), result2.ID)
		require.NotNil(test.T(), result2.Name)

		result3, err := test.repo.LoadCategoryFromDB(test.ctx, result2.ID) // Load
		require.Nil(test.T(), err)
		require.NotNil(test.T(), result3)
		require.NotNil(test.T(), result3.ID)
		assert.Equal(test.T(), result2.ID, result3.ID)
		assert.Equal(test.T(), result2.Name, result3.Name)

		assert.NotEqual(test.T(), result1.ID, result3.ID)
		assert.NotEqual(test.T(), result1.Name, result3.Name)
	})
}

// TestCategoryNotFoundError tests create category and check NotFoundError is Nil
func (test *categoryRepoBlackBoxTest) TestCategoryNotFoundError() {
	category1 := category.Category{
		Name: "Backlog1",
	}

	test.T().Run("create and check NotFoundError", func(t *testing.T) {

		// loads a random category by ID which is not created and tests it should return NotFoundError.
		randomID := uuid.FromStringOrNil("e42d0f80-9b1f-4715-b616-1fd931ce73cd")
		result1, err := test.repo.LoadCategoryFromDB(test.ctx, randomID) // Load
		require.NotNil(test.T(), err)
		require.Nil(test.T(), result1)
		_, ok := errors.Cause(err).(errs.NotFoundError)

		// creates a category
		result2, err := test.repo.Create(test.ctx, &category1) // Create
		require.Nil(test.T(), err)
		require.NotNil(test.T(), result2)
		require.NotNil(test.T(), result2.ID)
		require.NotNil(test.T(), result2.Name)

		// Loads the created category and tests that NotFound error is Nil
		result3, err := test.repo.LoadCategoryFromDB(test.ctx, category1.ID) // Load
		require.Nil(test.T(), err)
		require.NotNil(test.T(), result3)
		require.NotNil(test.T(), result3.ID)

		assert.Equal(test.T(), result2.ID, result3.ID)
		assert.Equal(test.T(), result2.Name, result3.Name)
		assert.True(test.T(), ok)
	})
}

// TestCreateCategoryWithSameNameFail tests that we cannot create another category with same name
// This tests unique name violation
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

// TestDoNotCreateCategoryWithMissingName tests that we cannot create a category without a name
func (test *categoryRepoBlackBoxTest) TestDoNotCreateCategoryWithMissingName() {
	category1 := category.Category{}

	test.T().Run("empty category name violation", func(t *testing.T) {
		cat1, err := test.repo.Create(test.ctx, &category1)
		require.NotNil(test.T(), err)
		require.Nil(test.T(), cat1)
	})
}

// TestListCategories lists the categories and checks the total count of categories list
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

		require.Nil(t, err)
		require.Condition(t, func() bool { return len(resultCategories) >= 4 }, "expected at least 4 categories") // 2 category names are hard-coded + 2 category names are created in this test = total 4 categories

		assert.Equal(t, category1.Name, resultCategories[0].Name)
		assert.Equal(t, category2.Name, resultCategories[1].Name)
	})
}
