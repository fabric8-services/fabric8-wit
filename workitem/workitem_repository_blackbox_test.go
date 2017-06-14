package workitem_test

import (
	"fmt"
	"testing"

	"github.com/almighty/almighty-core/codebase"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"

	"context"

	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type workItemRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo      workitem.WorkItemRepository
	clean     func()
	creatorID uuid.UUID
	spaceID   uuid.UUID
	ctx       context.Context
}

func TestRunWorkTypeRepoBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemRepoBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)
}

func (s *workItemRepoBlackBoxTest) SetupTest() {
	s.repo = workitem.NewWorkItemRepository(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.spaceID = space.SystemSpace
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "jdoe", "test")
	require.Nil(s.T(), err)
	s.creatorID = testIdentity.ID
}

func (s *workItemRepoBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *workItemRepoBlackBoxTest) TestFailDeleteZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	// given
	_, err := s.repo.Create(
		s.ctx, s.spaceID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create work item")
	// when
	err = s.repo.Delete(s.ctx, s.spaceID, "0", s.creatorID)
	// then
	require.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
}

func (s *workItemRepoBlackBoxTest) TestFailSaveZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	// given
	wi, err := s.repo.Create(
		s.ctx, s.spaceID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi.ID = "0"
	_, err = s.repo.Save(s.ctx, s.spaceID, *wi, s.creatorID)
	// then
	assert.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
}

func (s *workItemRepoBlackBoxTest) TestFaiLoadZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	// given
	_, err := s.repo.Create(
		s.ctx, s.spaceID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	_, err = s.repo.Load(s.ctx, s.spaceID, "0")
	// then
	assert.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
}

func (s *workItemRepoBlackBoxTest) TestSaveAssignees() {
	// given
	wi, err := s.repo.Create(
		s.ctx, s.spaceID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:     "Title",
			workitem.SystemState:     workitem.SystemStateNew,
			workitem.SystemAssignees: []string{"A", "B"},
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.LoadByID(s.ctx, wi.ID)
	// then
	require.Nil(s.T(), err)
	assert.Equal(s.T(), "A", wi.Fields[workitem.SystemAssignees].([]interface{})[0])
}

func (s *workItemRepoBlackBoxTest) TestSaveForUnchangedCreatedDate() {
	// given
	wi, err := s.repo.Create(
		s.ctx, s.spaceID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.Load(s.ctx, s.spaceID, wi.ID)
	require.Nil(s.T(), err)
	wiNew, err := s.repo.Save(s.ctx, s.spaceID, *wi, s.creatorID)
	// then
	require.Nil(s.T(), err)
	assert.Equal(s.T(), wi.Fields[workitem.SystemCreatedAt], wiNew.Fields[workitem.SystemCreatedAt])
}

func (s *workItemRepoBlackBoxTest) TestCreateWorkItemWithDescriptionNoMarkup() {
	// given
	wi, err := s.repo.Create(
		s.ctx, s.spaceID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "Title",
			workitem.SystemDescription: rendering.NewMarkupContentFromLegacy("Description"),
			workitem.SystemState:       workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.Load(s.ctx, s.spaceID, wi.ID)
	// then
	require.Nil(s.T(), err)
	// workitem.WorkItem does not contain the markup associated with the description (yet)
	assert.Equal(s.T(), rendering.NewMarkupContentFromLegacy("Description"), wi.Fields[workitem.SystemDescription])
}

func (s *workItemRepoBlackBoxTest) TestExistsWorkItem() {
	// given
	wi, err := s.repo.Create(
		s.ctx, s.spaceID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "Title",
			workitem.SystemDescription: rendering.NewMarkupContentFromLegacy("Description"),
			workitem.SystemState:       workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	var exists bool
	exists, err = s.repo.Exists(s.ctx, s.spaceID, wi.ID)
	// then
	require.Nil(s.T(), err)
	require.True(s.T(), exists)
}

func (s *workItemRepoBlackBoxTest) TestNoExistsWorkItem() {
	// when
	var exists bool
	exists, err := s.repo.Exists(s.ctx, s.spaceID, "123112")
	// then
	require.Nil(s.T(), err)
	require.False(s.T(), exists)
}

func (s *workItemRepoBlackBoxTest) TestCreateWorkItemWithDescriptionMarkup() {
	// given
	wi, err := s.repo.Create(
		s.ctx,
		s.spaceID,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "Title",
			workitem.SystemDescription: rendering.NewMarkupContent("Description", rendering.SystemMarkupMarkdown),
			workitem.SystemState:       workitem.SystemStateNew,
		},
		s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.Load(s.ctx, s.spaceID, wi.ID)
	// then
	require.Nil(s.T(), err)
	// workitem.WorkItem does not contain the markup associated with the description (yet)
	assert.Equal(s.T(), rendering.NewMarkupContent("Description", rendering.SystemMarkupMarkdown), wi.Fields[workitem.SystemDescription])
}

// TestTypeChangeIsNotProhibitedOnDBLayer tests that you can change the type of
// a work item. NOTE: This functionality only works on the DB layer and is not
// exposed to REST.
func (s *workItemRepoBlackBoxTest) TestTypeChangeIsNotProhibitedOnDBLayer() {
	// Create at least 1 item to avoid RowsAffectedCheck
	// given
	wi, err := s.repo.Create(
		s.ctx, s.spaceID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creatorID)
	require.Nil(s.T(), err)
	// when
	wi.Type = workitem.SystemFeature
	newWi, err := s.repo.Save(s.ctx, s.spaceID, *wi, s.creatorID)
	// then
	require.Nil(s.T(), err)
	assert.True(s.T(), uuid.Equal(workitem.SystemFeature, newWi.Type))
}

// TestGetCountsPerIteration makes sure that the query being executed is correctly returning
// the counts of work items
func (s *workItemRepoBlackBoxTest) TestGetCountsPerIteration() {
	// create seed data
	// given
	spaceRepo := space.NewRepository(s.DB)
	spaceInstance := space.Space{
		Name: "Testing space" + uuid.NewV4().String(),
	}
	spaceRepo.Create(s.ctx, &spaceInstance)
	assert.NotEqual(s.T(), uuid.UUID{}, spaceInstance.ID)
	// when
	iterationRepo := iteration.NewIterationRepository(s.DB)
	iteration1 := iteration.Iteration{
		Name:    "Sprint 1",
		SpaceID: spaceInstance.ID,
	}
	err := iterationRepo.Create(s.ctx, &iteration1)
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
	err = iterationRepo.Create(s.ctx, &iteration2)
	// then
	require.Nil(s.T(), err)
	s.T().Log("iteration2 id = ", iteration2.ID)
	assert.NotEqual(s.T(), uuid.UUID{}, iteration2.ID)
	// given
	for i := 0; i < 3; i++ {
		_, err = s.repo.Create(
			s.ctx, spaceInstance.ID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: iteration1.ID.String(),
			}, s.creatorID)
		require.Nil(s.T(), err)
	}
	for i := 0; i < 2; i++ {
		_, err = s.repo.Create(
			s.ctx, spaceInstance.ID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("Closed issue #%d", i),
				workitem.SystemState:     workitem.SystemStateClosed,
				workitem.SystemIteration: iteration1.ID.String(),
			}, s.creatorID)
		require.Nil(s.T(), err)
	}
	// when
	countsMap, _ := s.repo.GetCountsPerIteration(s.ctx, spaceInstance.ID)
	// then
	require.Len(s.T(), countsMap, 1)
	require.Contains(s.T(), countsMap, iteration1.ID.String())
	assert.Equal(s.T(), 5, countsMap[iteration1.ID.String()].Total)
	assert.Equal(s.T(), 2, countsMap[iteration1.ID.String()].Closed)
}

func (s *workItemRepoBlackBoxTest) TestCodebaseAttributes() {
	// given
	title := "solution on global warming"
	branch := "earth-recycle-101"
	repo := "golang-project"
	file := "main.go"
	line := 200
	cbase := codebase.Content{
		Branch:     branch,
		Repository: repo,
		FileName:   file,
		LineNumber: line,
	}

	wi, err := s.repo.Create(
		s.ctx, space.SystemSpace, workitem.SystemPlannerItem,
		map[string]interface{}{
			workitem.SystemTitle:    title,
			workitem.SystemState:    workitem.SystemStateNew,
			workitem.SystemCodebase: cbase,
		}, s.creatorID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.Load(s.ctx, space.SystemSpace, wi.ID)
	// then
	require.Nil(s.T(), err)
	assert.Equal(s.T(), title, wi.Fields[workitem.SystemTitle].(string))
	require.NotNil(s.T(), wi.Fields[workitem.SystemCodebase])
	cb := wi.Fields[workitem.SystemCodebase].(codebase.Content)
	assert.Equal(s.T(), repo, cb.Repository)
	assert.Equal(s.T(), branch, cb.Branch)
	assert.Equal(s.T(), file, cb.FileName)
	assert.Equal(s.T(), line, cb.LineNumber)
}
