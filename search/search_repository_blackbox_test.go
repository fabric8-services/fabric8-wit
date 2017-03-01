package search_test

import (
	"os"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/search"
	testsupport "github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

func TestRunSearchRepositoryBlackboxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchRepositoryBlackboxTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

type searchRepositoryBlackboxTest struct {
	gormsupport.DBTestSuite
	modifierID uuid.UUID
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
func (s *searchRepositoryBlackboxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if _, c := os.LookupEnv(resource.Database); c {
		if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
}

func (s *searchRepositoryBlackboxTest) SetupTest() {
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "jdoe", "test")
	require.Nil(s.T(), err)
	s.modifierID = testIdentity.ID
}

func (s *searchRepositoryBlackboxTest) TestRestrictByType() {
	// given
	undoScript := &gormsupport.DBScript{}
	defer undoScript.Run(s.DB)
	typeRepo := workitem.NewUndoableWorkItemTypeRepository(workitem.NewWorkItemTypeRepository(s.DB), undoScript)
	wiRepo := workitem.NewUndoableWorkItemRepository(workitem.NewWorkItemRepository(s.DB), undoScript)
	searchRepo := search.NewGormSearchRepository(s.DB)

	ctx := context.Background()
	res, count, err := searchRepo.SearchFullText(ctx, "TestRestrictByType", nil, nil)
	require.Nil(s.T(), err)
	require.True(s.T(), count == uint64(len(res))) // safety check for many, many instances of bogus search results.
	for _, wi := range res {
		wiRepo.Delete(ctx, wi.ID, s.modifierID)
	}

	s.DB.Unscoped().Delete(&workitem.WorkItemType{Name: "base"})
	s.DB.Unscoped().Delete(&workitem.WorkItemType{Name: "sub1"})
	s.DB.Unscoped().Delete(&workitem.WorkItemType{Name: "subtwo"})

	extended := workitem.SystemBug
	base, err := typeRepo.Create(ctx, &extended, "base", map[string]app.FieldDefinition{})
	require.NotNil(s.T(), base)
	require.Nil(s.T(), err)

	extended = "base"
	sub1, err := typeRepo.Create(ctx, &extended, "sub1", map[string]app.FieldDefinition{})
	require.NotNil(s.T(), sub1)
	require.Nil(s.T(), err)

	sub2, err := typeRepo.Create(ctx, &extended, "subtwo", map[string]app.FieldDefinition{})
	require.NotNil(s.T(), sub2)
	require.Nil(s.T(), err)

	wi1, err := wiRepo.Create(ctx, "sub1", map[string]interface{}{
		workitem.SystemTitle: "Test TestRestrictByType",
		workitem.SystemState: "closed",
	}, s.modifierID)
	require.NotNil(s.T(), wi1)
	require.Nil(s.T(), err)

	wi2, err := wiRepo.Create(ctx, "subtwo", map[string]interface{}{
		workitem.SystemTitle: "Test TestRestrictByType 2",
		workitem.SystemState: "closed",
	}, s.modifierID)
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

	res, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:subtwo", nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(1), count)
	if count == 1 {
		assert.Equal(s.T(), wi2.ID, res[0].ID)
	}

	_, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:base", nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)

	_, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:subtwo type:sub1", nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)

	_, count, err = searchRepo.SearchFullText(ctx, "TestRestrictByType type:base type:sub1", nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)

	_, count, err = searchRepo.SearchFullText(ctx, "TRBTgorxi type:base", nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), count)
}
