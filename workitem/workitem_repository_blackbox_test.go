package workitem_test

import (
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type workItemRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo    workitem.WorkItemRepository
	creator account.Identity
	space   space.Space
}

func TestRunWorkItemRepoBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *workItemRepoBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = workitem.NewWorkItemRepository(s.DB)
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
	s.space = *testFxt.Spaces[0]
	s.creator = *testFxt.Identities[0]
}

func (s *workItemRepoBlackBoxTest) TestFailSaveNilNumber() {
	// Create at least 1 item to avoid RowsEffectedCheck
	// given
	wi, err := s.repo.Create(
		s.Ctx, s.space.ID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creator.ID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi.Number = 0
	_, err = s.repo.Save(s.Ctx, s.space.ID, *wi, s.creator.ID)
	// then
	assert.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
}

func (s *workItemRepoBlackBoxTest) TestFailLoadNilID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	// given
	_, err := s.repo.Create(
		s.Ctx, s.space.ID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creator.ID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	_, err = s.repo.LoadByID(s.Ctx, uuid.Nil)
	// then
	assert.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
}

func (s *workItemRepoBlackBoxTest) TestSaveAssignees() {
	// given
	wi, err := s.repo.Create(
		s.Ctx, s.space.ID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:     "Title",
			workitem.SystemState:     workitem.SystemStateNew,
			workitem.SystemAssignees: []string{"A", "B"},
		}, s.creator.ID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.LoadByID(s.Ctx, wi.ID)
	// then
	require.Nil(s.T(), err)
	assert.Equal(s.T(), "A", wi.Fields[workitem.SystemAssignees].([]interface{})[0])
}

func (s *workItemRepoBlackBoxTest) TestSaveForUnchangedCreatedDate() {
	// given
	wi, err := s.repo.Create(
		s.Ctx, s.space.ID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creator.ID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.LoadByID(s.Ctx, wi.ID)
	require.Nil(s.T(), err)
	wiNew, err := s.repo.Save(s.Ctx, s.space.ID, *wi, s.creator.ID)
	// then
	require.Nil(s.T(), err)
	assert.Equal(s.T(), wi.Fields[workitem.SystemCreatedAt], wiNew.Fields[workitem.SystemCreatedAt])
}

func (s *workItemRepoBlackBoxTest) TestCreateWorkItemWithDescriptionNoMarkup() {
	// given
	wi, err := s.repo.Create(
		s.Ctx, s.space.ID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "Title",
			workitem.SystemDescription: rendering.NewMarkupContentFromLegacy("Description"),
			workitem.SystemState:       workitem.SystemStateNew,
		}, s.creator.ID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.LoadByID(s.Ctx, wi.ID)
	// then
	require.Nil(s.T(), err)
	// workitem.WorkItem does not contain the markup associated with the description (yet)
	assert.Equal(s.T(), rendering.NewMarkupContentFromLegacy("Description"), wi.Fields[workitem.SystemDescription])
}

func (s *workItemRepoBlackBoxTest) TestExistsWorkItem() {
	t := s.T()
	resource.Require(t, resource.Database)

	t.Run("work item exists", func(t *testing.T) {
		// given
		wi, err := s.repo.Create(
			s.Ctx, s.space.ID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:       "Title",
				workitem.SystemDescription: rendering.NewMarkupContentFromLegacy("Description"),
				workitem.SystemState:       workitem.SystemStateNew,
			}, s.creator.ID)
		require.Nil(s.T(), err, "Could not create workitem")
		// when
		err = s.repo.CheckExists(s.Ctx, wi.ID.String())
		// then
		require.Nil(t, err)
	})

	t.Run("work item doesn't exist", func(t *testing.T) {
		t.Parallel()
		// when
		err := s.repo.CheckExists(s.Ctx, "00000000-0000-0000-0000-000000000000")
		// then

		require.IsType(t, errors.NotFoundError{}, err)
	})

}

func (s *workItemRepoBlackBoxTest) TestCreateWorkItemWithDescriptionMarkup() {
	// given
	wi, err := s.repo.Create(
		s.Ctx,
		s.space.ID,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "Title",
			workitem.SystemDescription: rendering.NewMarkupContent("Description", rendering.SystemMarkupMarkdown),
			workitem.SystemState:       workitem.SystemStateNew,
		},
		s.creator.ID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.LoadByID(s.Ctx, wi.ID)
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
		s.Ctx, s.space.ID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creator.ID)
	require.Nil(s.T(), err)
	// when
	wi.Type = workitem.SystemFeature
	newWi, err := s.repo.Save(s.Ctx, s.space.ID, *wi, s.creator.ID)
	// then
	require.Nil(s.T(), err)
	assert.True(s.T(), uuid.Equal(workitem.SystemFeature, newWi.Type))
}

// TestGetCountsPerIteration makes sure that the query being executed is correctly returning
// the counts of work items
func (s *workItemRepoBlackBoxTest) TestGetCountsPerIteration() {
	// given
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Iterations(2), tf.WorkItems(5, func(fxt *tf.TestFixture, idx int) error {
		wi := fxt.WorkItems[idx]
		wi.Fields[workitem.SystemIteration] = fxt.Iterations[0].ID.String()
		if idx < 3 {
			wi.Fields[workitem.SystemTitle] = fmt.Sprintf("New issue #%d", idx)
			wi.Fields[workitem.SystemState] = workitem.SystemStateNew
		} else if idx >= 3 {
			wi.Fields[workitem.SystemTitle] = fmt.Sprintf("Closed issue #%d", idx-3)
			wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
		}
		return nil
	}))

	// when
	countsMap, _ := s.repo.GetCountsPerIteration(s.Ctx, testFxt.Spaces[0].ID)
	// then
	require.Len(s.T(), countsMap, 2)
	require.Contains(s.T(), countsMap, testFxt.Iterations[0].ID.String())
	assert.Equal(s.T(), 5, countsMap[testFxt.Iterations[0].ID.String()].Total)
	assert.Equal(s.T(), 2, countsMap[testFxt.Iterations[0].ID.String()].Closed)
	require.Contains(s.T(), countsMap, testFxt.Iterations[1].ID.String())
	assert.Equal(s.T(), 0, countsMap[testFxt.Iterations[1].ID.String()].Total)
	assert.Equal(s.T(), 0, countsMap[testFxt.Iterations[1].ID.String()].Closed)
}

func (s *workItemRepoBlackBoxTest) TestCodebaseAttributes() {
	// given
	title := "solution on global warming"
	branch := "earth-recycle-101"
	repo := "https://github.com/pranavgore09/go-tutorial.git"
	file := "main.go"
	line := 200
	cbase := codebase.Content{
		Branch:     branch,
		Repository: repo,
		FileName:   file,
		LineNumber: line,
	}

	wi, err := s.repo.Create(
		s.Ctx, space.SystemSpace, workitem.SystemPlannerItem,
		map[string]interface{}{
			workitem.SystemTitle:    title,
			workitem.SystemState:    workitem.SystemStateNew,
			workitem.SystemCodebase: cbase,
		}, s.creator.ID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wi, err = s.repo.LoadByID(s.Ctx, wi.ID)
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

func (s *workItemRepoBlackBoxTest) TestCodebaseInvalidRepo() {
	// given
	title := "solution on global warming"
	branch := "earth-recycle-101"
	repo := "https://non-github.com/pranavgore09/go-tutorial"
	file := "main.go"
	line := 200
	cbase := codebase.Content{
		Branch:     branch,
		Repository: repo,
		FileName:   file,
		LineNumber: line,
	}

	_, err := s.repo.Create(
		s.Ctx, space.SystemSpace, workitem.SystemPlannerItem,
		map[string]interface{}{
			workitem.SystemTitle:    title,
			workitem.SystemState:    workitem.SystemStateNew,
			workitem.SystemCodebase: cbase,
		}, s.creator.ID)
	require.NotNil(s.T(), err, "Should not create workitem")
}

func (s *workItemRepoBlackBoxTest) TestLookupIDByNamedSpaceAndNumberOK() {
	// given
	wi, err := s.repo.Create(
		s.Ctx, s.space.ID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creator.ID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	wiID, spaceID, err := s.repo.LookupIDByNamedSpaceAndNumber(s.Ctx, s.creator.Username, s.space.Name, wi.Number)
	// then
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wiID)
	assert.Equal(s.T(), wi.ID, *wiID)
	// TODO(xcoulon) can be removed once PR for #1452 is merged
	require.NotNil(s.T(), spaceID)
	assert.Equal(s.T(), wi.SpaceID, *spaceID)
}

func (s *workItemRepoBlackBoxTest) TestLookupIDByNamedSpaceAndNumberNotFound() {
	// given
	wi, err := s.repo.Create(
		s.Ctx, s.space.ID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.creator.ID)
	require.Nil(s.T(), err, "Could not create workitem")
	// when
	_, _, err = s.repo.LookupIDByNamedSpaceAndNumber(s.Ctx, "foo", s.space.Name, wi.Number)
	// then
	require.NotNil(s.T(), err)
	assert.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
}

func (s *workItemRepoBlackBoxTest) TestLookupIDByNamedSpaceAndNumberStaleSpace() {
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1), tf.WorkItems(20, func(testf *tf.TestFixture, idx int) error {
		testf.WorkItems[idx].Type = workitem.SystemBug
		testf.WorkItems[idx].Fields[workitem.SystemTitle] = fmt.Sprintf("title %d", idx)
		testf.WorkItems[idx].Fields[workitem.SystemState] = workitem.SystemStateNew
		testf.WorkItems[idx].Fields[workitem.SystemCreator] = testf.Identities[0].ID.String()
		return nil
	}))
	sp := *testFxt.Spaces[0]
	wi := *testFxt.WorkItems[0]
	in := *testFxt.Identities[0]
	wiID, spaceID, err := s.repo.LookupIDByNamedSpaceAndNumber(s.ctx, in.Username, sp.Name, wi.Number)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wiID)
	assert.Equal(s.T(), wi.ID, *wiID)
	require.NotNil(s.T(), spaceID)
	assert.Equal(s.T(), wi.SpaceID, *spaceID)

	// delete above space
	spaceRepo := space.NewRepository(s.DB)
	err = spaceRepo.Delete(s.ctx, sp.ID)
	require.Nil(s.T(), err)

	testFxt2 := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1, func(testf *tf.TestFixture, idx int) error {
		testf.Spaces[0].Name = sp.Name
		testf.Spaces[0].OwnerId = in.ID
		return nil
	}), tf.WorkItems(20, func(testf *tf.TestFixture, idx int) error {
		testf.WorkItems[idx].Type = workitem.SystemBug
		testf.WorkItems[idx].Fields[workitem.SystemTitle] = fmt.Sprintf("title %d", idx)
		testf.WorkItems[idx].Fields[workitem.SystemState] = workitem.SystemStateNew
		testf.WorkItems[idx].Fields[workitem.SystemCreator] = testf.Identities[0].ID.String()
		return nil
	}))
	sp2 := *testFxt2.Spaces[0]
	wi2 := *testFxt2.WorkItems[0]
	wiID2, spaceID2, err := s.repo.LookupIDByNamedSpaceAndNumber(s.ctx, in.Username, sp2.Name, wi2.Number)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wiID2)
	assert.Equal(s.T(), wi2.ID, *wiID2)
	require.NotNil(s.T(), spaceID2)
	assert.Equal(s.T(), wi2.SpaceID, *spaceID2)
}
