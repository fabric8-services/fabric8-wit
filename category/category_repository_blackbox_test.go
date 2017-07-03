package category_test

import (
	"context"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/category"
	errs "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
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

func (test *categoryRepoBlackBoxTest) SetupSuite() {
	test.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	test.DBTestSuite.PopulateDBTestSuite(ctx)
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

		result2, err := test.repo.LoadCategory(test.ctx, result1.ID) // Load
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

		result3, err := test.repo.LoadCategory(test.ctx, result2.ID) // Load
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
		result1, err := test.repo.LoadCategory(test.ctx, randomID) // Load
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
		result3, err := test.repo.LoadCategory(test.ctx, category1.ID) // Load
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
		require.False(test.T(), cat1.CreatedAt.After(time.Now()), "category was not created, CreatedAt after Now()")

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
		// given
		category1Payload := category.Category{
			Name: "Category1",
		}
		cat1, err := test.repo.Create(test.ctx, &category1Payload)
		require.Nil(t, err)
		category2Payload := category.Category{
			Name: "Category2",
		}
		cat2, err := test.repo.Create(test.ctx, &category2Payload)
		require.Nil(t, err)

		// when
		resultCategories, err := test.repo.List(test.ctx)

		// then
		require.Nil(t, err)
		require.Condition(t, func() bool { return len(resultCategories) >= 4 }, "expected at least 4 categories (2 from populate common types + 2 from this test)")
		for _, id := range []uuid.UUID{cat1.ID, cat2.ID} {
			found := false
			for _, cat := range resultCategories {
				if cat.ID == id {
					found = true
				}
			}
			assert.True(t, found, "failed to find ID %s in list of categories", id)
		}
	})
}

// TestListCategories creates category and checks if it exists and checks that category should not exist when not created
func (test *categoryRepoBlackBoxTest) TestExistsCategories() {
	t := test.T()
	resource.Require(t, resource.Database)

	t.Run("category exists", func(t *testing.T) {
		// given
		category1Payload := category.Category{
			Name: "Category1",
		}
		cat1, err := test.repo.Create(test.ctx, &category1Payload)
		require.Nil(t, err)
		require.NotNil(test.T(), cat1)
		require.NotNil(test.T(), cat1.ID)

		// when
		exists, err1 := test.repo.Exists(context.Background(), cat1.ID.String())
		// then
		require.Nil(t, err1)
		assert.True(t, exists)
	})

	t.Run("category doesn't exist", func(t *testing.T) {
		// when
		exists, err := test.repo.Exists(context.Background(), uuid.NewV4().String())
		// then
		require.IsType(t, errs.NotFoundError{}, err)
		assert.False(t, exists)
	})
}
