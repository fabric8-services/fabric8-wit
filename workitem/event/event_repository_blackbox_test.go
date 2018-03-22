package event_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type eventRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	wiRepo       workitem.WorkItemRepository
	wiEventRepo  event.WorkItemEventRepository
	identityRepo account.IdentityRepository
}

func TestRunEventRepoBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &eventRepoBlackBoxTest{
		DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml"),
	})
}

func (s *eventRepoBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.wiRepo = workitem.NewWorkItemRepository(s.DB)
	s.wiEventRepo = event.NewWorkItemEventRepository(s.DB)
	s.identityRepo = account.NewIdentityRepository(s.DB)
}

func (s *eventRepoBlackBoxTest) TestList() {

	s.T().Run("empty event list", func(t *testing.T) {

		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))

		eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.Empty(t, eventList)
	})
	s.T().Run("event assignee - previous assignee nil", func(t *testing.T) {

		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))

		assignee := []string{fxt.Identities[0].ID.String()}

		fxt.WorkItems[0].Fields[workitem.SystemAssignees] = assignee
		wiNew, err := s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemAssignees].([]interface{}), 1)
		eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 1)
		assert.Equal(t, eventList[0].Name, "assigned")
		assert.Empty(t, eventList[0].PreviousAssignees)
		assert.Equal(t, fxt.Identities[0].Username, eventList[0].NewAssignees[0])
	})

	s.T().Run("event assignee - new assignee nil", func(t *testing.T) {

		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))

		assignee := []string{fxt.Identities[0].ID.String()}

		fxt.WorkItems[0].Fields[workitem.SystemAssignees] = assignee
		wiNew, err := s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemAssignees].([]interface{}), 1)
		eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 1)
		assert.Equal(t, eventList[0].Name, "assigned")
		assert.Empty(t, eventList[0].PreviousAssignees)
		assert.Equal(t, fxt.Identities[0].Username, eventList[0].NewAssignees[0])

		wiNew.Fields[workitem.SystemAssignees] = []string{}
		wiNew.Version = fxt.WorkItems[0].Version + 1
		wiNew, err = s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *wiNew, fxt.Identities[0].ID)
		//panic(wiNew.ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemAssignees].([]interface{}), 0)
		eventList, err = s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 2)
		assert.Equal(t, eventList[1].Name, "assigned")
		assert.Empty(t, eventList[1].NewAssignees)
	})
	s.T().Run("state change from new to open", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateResolved
		wiNew, err := s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Equal(t, workitem.SystemStateResolved, wiNew.Fields[workitem.SystemState])
		eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 1)
	})
}
