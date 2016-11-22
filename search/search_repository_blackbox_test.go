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
	"golang.org/x/net/context"
)

func TestRestricByType(t *testing.T) {
	resource.Require(t, resource.Database)
	undoScript := &gormsupport.DBScript{}
	defer undoScript.Run(search.DB)
	typeRepo := models.NewUndoableWorkItemTypeRepository(models.NewWorkItemTypeRepository(search.DB), undoScript)
	wiRepo := models.NewUndoableWorkItemRepository(models.NewWorkItemRepository(search.DB), undoScript)
	searchRepo := search.NewGormSearchRepository(search.DB)

	ctx := context.Background()
	res, count, err := searchRepo.SearchFullText(ctx, "TestRestrictByType", nil, nil)
	require.Nil(t, err)
	require.True(t, count == uint64(len(res))) // safety check for many, many instances of bogus search results.
	for _, wi := range res {
		wiRepo.Delete(ctx, wi.ID)
	}

	search.DB.Unscoped().Delete(&models.WorkItemType{Name: "base"})
	search.DB.Unscoped().Delete(&models.WorkItemType{Name: "sub1"})
	search.DB.Unscoped().Delete(&models.WorkItemType{Name: "sub two"})

	extended := models.SystemBug
	base, err := typeRepo.Create(ctx, &extended, "base", map[string]app.FieldDefinition{})
	require.NotNil(t, base)
	require.Nil(t, err)

	extended = "base"
	sub1, err := typeRepo.Create(ctx, &extended, "sub1", map[string]app.FieldDefinition{})
	require.NotNil(t, sub1)
	require.Nil(t, err)

	sub2, err := typeRepo.Create(ctx, &extended, "sub two", map[string]app.FieldDefinition{})
	require.NotNil(t, sub2)
	require.Nil(t, err)

	wi1, err := wiRepo.Create(ctx, "sub1", map[string]interface{}{
		models.SystemTitle: "Test TestRestrictByType",
		models.SystemState: "closed",
	}, account.TestIdentity.ID.String())
	require.NotNil(t, wi1)
	require.Nil(t, err)

	wi2, err := wiRepo.Create(ctx, "sub two", map[string]interface{}{
		models.SystemTitle: "Test TestRestrictByType 2",
		models.SystemState: "closed",
	}, account.TestIdentity.ID.String())
	require.NotNil(t, wi2)
	require.Nil(t, err)

	res, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType", nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), count)

	res, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:sub1", nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), count)
	if count == 1 {
		assert.Equal(t, wi1.ID, res[0].ID)
	}

	res, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:sub+two", nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), count)
	if count == 1 {
		assert.Equal(t, wi2.ID, res[0].ID)
	}

	res, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:base", nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), count)

	res, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:sub+two type:sub1", nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), count)

	res, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:base type:sub1", nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), count)

	res, count, err = searchRepo.SearchFullText(ctx, "TRBTgorxi type:base", nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), count)
}
