package workitem_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
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
	repo workitem.WorkItemRepository
}

func TestRunWorkItemRepoBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *workItemRepoBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = workitem.NewWorkItemRepository(s.DB)
}

func (s *workItemRepoBlackBoxTest) TestSave() {
	s.T().Run("fail - save nil number", func(t *testing.T) {
		// given at least 1 item to avoid RowsEffectedCheck
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))
		// when
		fxt.WorkItems[0].Number = 0
		_, err := s.repo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		// then
		assert.IsType(t, errors.NotFoundError{}, errs.Cause(err))
	})

	s.T().Run("ok - save for unchanged created date", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))
		oldDate, ok := fxt.WorkItems[0].Fields[workitem.SystemCreatedAt].(time.Time)
		require.True(t, ok, "failed to convert interface{} to time.Time")
		wiNew, err := s.repo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		newTime, ok := wiNew.Fields[workitem.SystemCreatedAt].(time.Time)
		require.True(t, ok, "failed to convert interface{} to time.Time")
		// then
		require.Nil(t, err)
		assert.Equal(t, oldDate.UTC(), newTime.UTC())
	})

	s.T().Run("change is not prohibited", func(t *testing.T) {
		// tests that you can change the type of a work item. NOTE: This
		// functionality only works on the DB layer and is not exposed to REST.
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1), tf.WorkItemTypes(2))
		// when
		fxt.WorkItems[0].Type = fxt.WorkItemTypes[1].ID
		newWi, err := s.repo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		// then
		require.Nil(s.T(), err)
		assert.Equal(s.T(), fxt.WorkItemTypes[1].ID, newWi.Type)
	})

}

func (s *workItemRepoBlackBoxTest) TestLoadID() {
	s.T().Run("fail - load nil ID", func(t *testing.T) {
		_, err := s.repo.LoadByID(s.Ctx, uuid.Nil)
		// then
		assert.IsType(t, errors.NotFoundError{}, errs.Cause(err))
	})
}

func (s *workItemRepoBlackBoxTest) TestCreate() {
	s.T().Run("ok - save assignees", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemAssignees] = []string{"A", "B"}
			return nil
		}))
		// when
		wi, err := s.repo.LoadByID(s.Ctx, fxt.WorkItems[0].ID)
		// then
		require.Nil(t, err)
		require.Len(t, wi.Fields[workitem.SystemAssignees].([]interface{}), 2)
		assert.Equal(t, "A", wi.Fields[workitem.SystemAssignees].([]interface{})[0])
		assert.Equal(t, "B", wi.Fields[workitem.SystemAssignees].([]interface{})[1])
	})

	s.T().Run("ok - create work item with description no markup", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemDescription] = rendering.NewMarkupContentFromLegacy("Description")
			return nil
		}))
		// when
		wi, err := s.repo.LoadByID(s.Ctx, fxt.WorkItems[0].ID)
		// then
		require.Nil(t, err)
		// workitem.WorkItem does not contain the markup associated with the description (yet)
		assert.Equal(t, rendering.NewMarkupContentFromLegacy("Description"), wi.Fields[workitem.SystemDescription])
	})

	s.T().Run("ok - work item with description markup", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemDescription] = rendering.NewMarkupContent("Description", rendering.SystemMarkupMarkdown)
			return nil
		}))
		// when
		wi, err := s.repo.LoadByID(s.Ctx, fxt.WorkItems[0].ID)
		// then
		require.Nil(t, err)
		// workitem.WorkItem does not contain the markup associated with the description (yet)
		assert.Equal(t, rendering.NewMarkupContent("Description", rendering.SystemMarkupMarkdown), wi.Fields[workitem.SystemDescription])
	})

	s.T().Run("ok - code base attributes", func(t *testing.T) {
		// given
		title := "solution on global warming"
		branch := "earth-recycle-101"
		repo := "https://github.com/pranavgore09/go-tutorial.git"
		file := "main.go"
		line := 200
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = title
			fxt.WorkItems[idx].Fields[workitem.SystemCodebase] = codebase.Content{
				Branch:     branch,
				Repository: repo,
				FileName:   file,
				LineNumber: line,
			}
			return nil
		}))
		// when
		wi, err := s.repo.LoadByID(s.Ctx, fxt.WorkItems[0].ID)
		// then
		require.Nil(t, err)
		assert.Equal(t, title, wi.Fields[workitem.SystemTitle].(string))
		require.NotNil(t, wi.Fields[workitem.SystemCodebase])
		cb := wi.Fields[workitem.SystemCodebase].(codebase.Content)
		assert.Equal(t, repo, cb.Repository)
		assert.Equal(t, branch, cb.Branch)
		assert.Equal(t, file, cb.FileName)
		assert.Equal(t, line, cb.LineNumber)
	})

	s.T().Run("fail - code base attributes: invalid repo", func(t *testing.T) {
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
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypes(1))
		_, err := s.repo.Create(
			s.Ctx, fxt.Spaces[0].ID, fxt.WorkItemTypes[0].ID,
			map[string]interface{}{
				workitem.SystemTitle:    title,
				workitem.SystemState:    workitem.SystemStateNew,
				workitem.SystemCodebase: cbase,
			}, fxt.Identities[0].ID)
		require.NotNil(t, err)
	})

}

func (s *workItemRepoBlackBoxTest) TestCheckExists() {
	s.T().Run("work item exists", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))
		// when
		err := s.repo.CheckExists(s.Ctx, fxt.WorkItems[0].ID.String())
		// then
		require.Nil(t, err)
	})

	s.T().Run("work item doesn't exist", func(t *testing.T) {
		// when
		err := s.repo.CheckExists(s.Ctx, uuid.NewV4().String())
		// then
		require.IsType(t, errors.NotFoundError{}, err)
	})
}

func (s *workItemRepoBlackBoxTest) TestGetCountsPerIteration() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		testFxt := tf.NewTestFixture(t, s.DB, tf.Iterations(2), tf.WorkItems(5, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			wi.Fields[workitem.SystemIteration] = fxt.Iterations[0].ID.String()
			if idx < 3 {
				wi.Fields[workitem.SystemState] = workitem.SystemStateNew
			} else if idx >= 3 {
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
			}
			return nil
		}))

		// when
		countsMap, _ := s.repo.GetCountsPerIteration(s.Ctx, testFxt.Spaces[0].ID)
		// then
		require.Len(t, countsMap, 2)
		require.Contains(t, countsMap, testFxt.Iterations[0].ID.String())
		assert.Equal(t, 5, countsMap[testFxt.Iterations[0].ID.String()].Total)
		assert.Equal(t, 2, countsMap[testFxt.Iterations[0].ID.String()].Closed)
		require.Contains(t, countsMap, testFxt.Iterations[1].ID.String())
		assert.Equal(t, 0, countsMap[testFxt.Iterations[1].ID.String()].Total)
		assert.Equal(t, 0, countsMap[testFxt.Iterations[1].ID.String()].Closed)
	})
}

func (s *workItemRepoBlackBoxTest) TestLookupIDByNamedSpaceAndNumber() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))
		// when
		wiID, spaceID, err := s.repo.LookupIDByNamedSpaceAndNumber(s.Ctx, fxt.Identities[0].Username, fxt.Spaces[0].Name, fxt.WorkItems[0].Number)
		// then
		require.Nil(t, err)
		require.NotNil(t, wiID)
		assert.Equal(t, fxt.WorkItems[0].ID, *wiID)
		// TODO(xcoulon) can be removed once PR for #1452 is merged
		require.NotNil(t, spaceID)
		assert.Equal(t, fxt.WorkItems[0].SpaceID, *spaceID)
	})

	s.T().Run("not found", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))
		// when
		_, _, err := s.repo.LookupIDByNamedSpaceAndNumber(s.Ctx, "foo", fxt.Spaces[0].Name, fxt.WorkItems[0].Number)
		// then
		require.NotNil(s.T(), err)
		assert.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
	})
}
