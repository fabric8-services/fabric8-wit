package search_test

import (
	"os"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
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
	clean      func()
	searchRepo *search.GormSearchRepository
	wiRepo     *workitem.GormWorkItemRepository
	witRepo    *workitem.GormWorkItemTypeRepository
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
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.witRepo = workitem.NewWorkItemTypeRepository(s.DB)
	s.wiRepo = workitem.NewWorkItemRepository(s.DB)
	s.searchRepo = search.NewGormSearchRepository(s.DB)
}

func (s *searchRepositoryBlackboxTest) TearDownTest() {
	s.clean()
}

func (s *searchRepositoryBlackboxTest) TestRestrictByType() {
	// given
	ctx := context.Background()
	res, count, err := s.searchRepo.SearchFullText(ctx, "TestRestrictByType", nil, nil)
	require.Nil(s.T(), err)
	require.True(s.T(), count == uint64(len(res))) // safety check for many, many instances of bogus search results.
	for _, wi := range res {
		s.wiRepo.Delete(ctx, wi.ID, s.modifierID)
	}

	extended := workitem.SystemBug
	base, err := s.witRepo.Create(ctx, nil, &extended, "base", nil, "fa-bomb", map[string]app.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), base)
	require.NotNil(s.T(), base.Data)
	require.NotNil(s.T(), base.Data.ID)

	sub1, err := s.witRepo.Create(ctx, nil, base.Data.ID, "sub1", nil, "fa-bomb", map[string]app.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), sub1)
	require.NotNil(s.T(), sub1.Data)
	require.NotNil(s.T(), sub1.Data.ID)

	sub2, err := s.witRepo.Create(ctx, nil, base.Data.ID, "subtwo", nil, "fa-bomb", map[string]app.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), sub2)
	require.NotNil(s.T(), sub2.Data)
	require.NotNil(s.T(), sub2.Data.ID)

	wi1, err := s.wiRepo.Create(ctx, *sub1.Data.ID, map[string]interface{}{
		workitem.SystemTitle: "Test TestRestrictByType",
		workitem.SystemState: "closed",
	}, s.modifierID)
	require.NotNil(s.T(), wi1)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wi1)

	wi2, err := s.wiRepo.Create(ctx, *sub2.Data.ID, map[string]interface{}{
		workitem.SystemTitle: "Test TestRestrictByType 2",
		workitem.SystemState: "closed",
	}, s.modifierID)
	require.NotNil(s.T(), wi2)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wi2)

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType", nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+sub1.Data.ID.String(), nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(1), count)
	if count == 1 {
		assert.Equal(s.T(), wi1.ID, res[0].ID)
	}

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+sub2.Data.ID.String(), nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(1), count)
	if count == 1 {
		assert.Equal(s.T(), wi2.ID, res[0].ID)
	}

	_, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+base.Data.ID.String(), nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)

	_, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+sub2.Data.ID.String()+" type:"+sub1.Data.ID.String(), nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)

	_, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+base.Data.ID.String()+" type:"+sub1.Data.ID.String(), nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)

	_, count, err = s.searchRepo.SearchFullText(ctx, "TRBTgorxi type:"+base.Data.ID.String(), nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), count)
}
