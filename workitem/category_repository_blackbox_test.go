package workitem_test

import (
	"context"
	"testing"

	"github.com/almighty/almighty-core/category"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type categoryRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo  category.CategoryRepository
	clean func()
	ctx   context.Context
}

func TestRunCategoryRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &categoryRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (test *categoryRepoBlackBoxTest) SetupTest() {
	test.repo = category.NewCategoryRepository(test.DB)
	test.clean = cleaner.DeleteCreatedEntities(test.DB)
}

func (test *categoryRepoBlackBoxTest) TearDownTest() {
	test.clean()
}

// TestCreateCategory tests that we can create a category
func (test *categoryRepoBlackBoxTest) TestCreateLoadCategory() {

	category := category.Category{
		Name: "planner.testIssues",
	}
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
}
