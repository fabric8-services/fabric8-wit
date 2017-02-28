package workitem_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/space"
	"github.com/almighty/almighty-core/test/resource"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type workItemRepoBlackBoxTest struct {
	gormsupport.DBTestSuite
	repo      workitem.WorkItemRepository
	clean     func()
	creatorID uuid.UUID
}

func TestRunWorkTypeRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &workItemRepoBlackBoxTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemRepoBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if _, c := os.LookupEnv(resource.Database); c != false {
		if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
}

func (s *workItemRepoBlackBoxTest) SetupTest() {
	s.repo = workitem.NewWorkItemRepository(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.creatorID = uuid.NewV4()
}

func (s *workItemRepoBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *workItemRepoBlackBoxTest) TestFailDeleteZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	// given
	_, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create work item")
	// when
	err = s.repo.Delete(context.Background(), "0")
	// then
	assert.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
}

func (s *workItemRepoBlackBoxTest) TestFailSaveZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	// given
	wi, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi.ID = "0"
	_, err = s.repo.Save(context.Background(), *wi)
	// then
	assert.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
}

func (s *workItemRepoBlackBoxTest) TestFaiLoadZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	// given
	_, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	_, err = s.repo.Load(context.Background(), "0")
	// then
	assert.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
}

func (s *workItemRepoBlackBoxTest) TestSaveAssignees() {
	// given
	wi, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:     "Title",
			workitem.SystemState:     workitem.SystemStateNew,
			workitem.SystemAssignees: []string{"A", "B"},
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.Load(context.Background(), wi.ID)
	// then
	require.Nil(s.T(), err)
	assert.Equal(s.T(), "A", wi.Fields[workitem.SystemAssignees].([]interface{})[0])
}

func (s *workItemRepoBlackBoxTest) TestSaveForUnchangedCreatedDate() {
	// given
	wi, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.Load(context.Background(), wi.ID)
	require.Nil(s.T(), err)
	wiNew, err := s.repo.Save(context.Background(), *wi)
	// then
	require.Nil(s.T(), err)
	assert.Equal(s.T(), wi.Fields[workitem.SystemCreatedAt], wiNew.Fields[workitem.SystemCreatedAt])
}

func (s *workItemRepoBlackBoxTest) TestCreateWorkItemWithDescriptionNoMarkup() {
	// given
	wi, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "Title",
			workitem.SystemDescription: rendering.NewMarkupContentFromLegacy("Description"),
			workitem.SystemState:       workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.Load(context.Background(), wi.ID)
	// then
	require.Nil(s.T(), err)
	// app.WorkItem does not contain the markup associated with the description (yet)
	assert.Equal(s.T(), rendering.NewMarkupContentFromLegacy("Description"), wi.Fields[workitem.SystemDescription])
}

func (s *workItemRepoBlackBoxTest) TestCreateWorkItemWithDescriptionMarkup() {
	// given
	wi, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "Title",
			workitem.SystemDescription: rendering.NewMarkupContent("Description", rendering.SystemMarkupMarkdown),
			workitem.SystemState:       workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.Load(context.Background(), wi.ID)
	// then
	require.Nil(s.T(), err)
	// app.WorkItem does not contain the markup associated with the description (yet)
	assert.Equal(s.T(), rendering.NewMarkupContent("Description", rendering.SystemMarkupMarkdown), wi.Fields[workitem.SystemDescription])
}

// TestTypeChangeIsNotProhibitedOnDBLayer tests that you can change the type of
// a work item. NOTE: This functionality only works on the DB layer and is not
// exposed to REST.
func (s *workItemRepoBlackBoxTest) TestTypeChangeIsNotProhibitedOnDBLayer() {
	// Create at least 1 item to avoid RowsAffectedCheck
	// given
	wi, err := s.repo.Create(
		context.Background(), "bug",
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err)
	// when
	wi.Type = "feature"
	newWi, err := s.repo.Save(context.Background(), *wi)
	// then
	require.Nil(s.T(), err)
	assert.Equal(s.T(), "feature", newWi.Type)
}

// TestGetCountsPerIteration makes sure that the query being executed is correctly returning
// the counts of work items
func (s *workItemRepoBlackBoxTest) TestGetCountsPerIteration() {
	// create seed data
	// given
	spaceRepo := space.NewRepository(s.DB)
	spaceInstance := space.Space{
		Name: "Testing space",
	}
	spaceRepo.Create(context.Background(), &spaceInstance)
	assert.NotEqual(s.T(), uuid.UUID{}, spaceInstance.ID)
	// when
	iterationRepo := iteration.NewIterationRepository(s.DB)
	iteration1 := iteration.Iteration{
		Name:    "Sprint 1",
		SpaceID: spaceInstance.ID,
	}
	err := iterationRepo.Create(context.Background(), &iteration1)
	// then
	require.Nil(s.T(), err)
	s.T().Log("iteration1 id = ", iteration1.ID)
	assert.NotEqual(s.T(), uuid.UUID{}, iteration1.ID)
	// given
	iteration2 := iteration.Iteration{
		Name:    "Sprint 2",
		SpaceID: spaceInstance.ID,
	}
	// when
	err = iterationRepo.Create(context.Background(), &iteration2)
	// then
	require.Nil(s.T(), err)
	s.T().Log("iteration2 id = ", iteration2.ID)
	assert.NotEqual(s.T(), uuid.UUID{}, iteration2.ID)
	// given
	for i := 0; i < 3; i++ {
		_, err = s.repo.Create(
			context.Background(), workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: iteration1.ID.String(),
			}, s.creatorID)
		require.Nil(s.T(), err)
	}
	for i := 0; i < 2; i++ {
		_, err = s.repo.Create(
			context.Background(), workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("Closed issue #%d", i),
				workitem.SystemState:     workitem.SystemStateClosed,
				workitem.SystemIteration: iteration1.ID.String(),
			}, s.creatorID)
		require.Nil(s.T(), err)
	}
	// when
	countsMap, _ := s.repo.GetCountsPerIteration(context.Background(), spaceInstance.ID)
	// then
	require.Len(s.T(), countsMap, 1)
	require.Contains(s.T(), countsMap, iteration1.ID.String())
	assert.Equal(s.T(), 5, countsMap[iteration1.ID.String()].Total)
	assert.Equal(s.T(), 2, countsMap[iteration1.ID.String()].Closed)
}
