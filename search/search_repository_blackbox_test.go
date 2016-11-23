package search_test

import (
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type searchRepositoryBlackboxTest struct {
	gormsupport.DBTestSuite
}

func TestRunSearchRepositoryWhiteboxTest(t *testing.T) {
	suite.Run(t, &searchRepositoryBlackboxTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

func (s *searchRepositoryBlackboxTest) TestRestrictByType() {
	resource.Require(s.T(), resource.Database)
	undoScript := &gormsupport.DBScript{}
	defer undoScript.Run(s.DB)
	typeRepo := models.NewUndoableWorkItemTypeRepository(models.NewWorkItemTypeRepository(s.DB), undoScript)
	wiRepo := models.NewUndoableWorkItemRepository(models.NewWorkItemRepository(s.DB), undoScript)
	searchRepo := search.NewGormSearchRepository(s.DB)

	ctx := context.Background()
	res, count, err := searchRepo.SearchFullText(ctx, "TestRestrictByType", nil, nil)
	require.Nil(s.T(), err)
	require.True(s.T(), count == uint64(len(res))) // safety check for many, many instances of bogus search results.
	for _, wi := range res {
		wiRepo.Delete(ctx, wi.ID)
	}

	s.DB.Unscoped().Delete(&models.WorkItemType{Name: "base"})
	s.DB.Unscoped().Delete(&models.WorkItemType{Name: "sub1"})
	s.DB.Unscoped().Delete(&models.WorkItemType{Name: "sub two"})

	extended := models.SystemBug
	base, err := typeRepo.Create(ctx, &extended, "base", map[string]app.FieldDefinition{})
	require.NotNil(s.T(), base)
	require.Nil(s.T(), err)

	extended = "base"
	sub1, err := typeRepo.Create(ctx, &extended, "sub1", map[string]app.FieldDefinition{})
	require.NotNil(s.T(), sub1)
	require.Nil(s.T(), err)

	sub2, err := typeRepo.Create(ctx, &extended, "sub two", map[string]app.FieldDefinition{})
	require.NotNil(s.T(), sub2)
	require.Nil(s.T(), err)

	wi1, err := wiRepo.Create(ctx, "sub1", map[string]interface{}{
		models.SystemTitle: "Test TestRestrictByType",
		models.SystemState: "closed",
	}, account.TestIdentity.ID.String())
	require.NotNil(s.T(), wi1)
	require.Nil(s.T(), err)

	wi2, err := wiRepo.Create(ctx, "sub two", map[string]interface{}{
		models.SystemTitle: "Test TestRestrictByType 2",
		models.SystemState: "closed",
	}, account.TestIdentity.ID.String())
	require.NotNil(s.T(), wi2)
	require.Nil(s.T(), err)

	res, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType", nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)

	res, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:sub1", nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(1), count)
	if count == 1 {
		assert.Equal(s.T(), wi1.ID, res[0].ID)
	}

	res, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:sub+two", nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(1), count)
	if count == 1 {
		assert.Equal(s.T(), wi2.ID, res[0].ID)
	}

	res, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:base", nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)

	res, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:sub+two type:sub1", nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)

	res, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:base type:sub1", nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)

	res, count, err = searchRepo.SearchFullText(ctx, "TRBTgorxi type:base", nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), count)
}
