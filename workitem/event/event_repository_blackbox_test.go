package event_test

import (
	"strings"
	"testing"

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
	wiRepo      workitem.WorkItemRepository
	wiEventRepo event.Repository
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
	s.wiEventRepo = event.NewEventRepository(s.DB)
}

func (s *eventRepoBlackBoxTest) TestList() {

	s.T().Run("empty event list", func(t *testing.T) {

		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))

		eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.Empty(t, eventList)
	})

	s.T().Run("event assignee", func(t *testing.T) {

		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1), tf.Identities(2))

		assignee := []string{fxt.Identities[0].ID.String()}

		fxt.WorkItems[0].Fields[workitem.SystemAssignees] = assignee
		wiNew, err := s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemAssignees].([]interface{}), 1)
		eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 1)
		assert.Equal(t, eventList[0].Name, event.Assignees)
		assert.Empty(t, eventList[0].Old)
		assert.Equal(t, fxt.Identities[0].ID.String(), strings.Split(eventList[0].New, ",")[0])

		assignee = []string{fxt.Identities[1].ID.String()}
		wiNew.Fields[workitem.SystemAssignees] = assignee
		wiNew.Version = fxt.WorkItems[0].Version + 1
		wiNew, err = s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *wiNew, fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemAssignees].([]interface{}), 1)
		eventList, err = s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 2)
		assert.Equal(t, eventList[1].Name, event.Assignees)
		assert.NotEmpty(t, eventList[1].Old)
		assert.NotEmpty(t, eventList[1].New)
		assert.Equal(t, fxt.Identities[0].ID.String(), strings.Split(eventList[0].New, ",")[0])
		assert.Equal(t, fxt.Identities[1].ID.String(), strings.Split(eventList[1].New, ",")[0])
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
		assert.Equal(t, eventList[0].Name, event.Assignees)
		assert.Empty(t, eventList[0].Old)
		assert.Equal(t, fxt.Identities[0].ID.String(), strings.Split(eventList[0].New, ",")[0])
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
		assert.Equal(t, eventList[0].Name, event.Assignees)
		assert.Empty(t, eventList[0].Old)
		assert.Equal(t, fxt.Identities[0].ID.String(), strings.Split(eventList[0].New, ",")[0])

		wiNew.Fields[workitem.SystemAssignees] = []string{}
		wiNew.Version = fxt.WorkItems[0].Version + 1
		wiNew, err = s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *wiNew, fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemAssignees].([]interface{}), 0)
		eventList, err = s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 2)
		assert.Equal(t, eventList[1].Name, event.Assignees)
		assert.Empty(t, eventList[1].New)
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
		assert.Equal(t, eventList[0].Name, event.State)
		assert.Equal(t, workitem.SystemStateResolved, eventList[0].New)
	})
	s.T().Run("event label", func(t *testing.T) {

		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))

		label := []string{"label1"}

		fxt.WorkItems[0].Fields[workitem.SystemLabels] = label
		wiNew, err := s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemLabels].([]interface{}), 1)
		eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 1)
		assert.Equal(t, eventList[0].Name, event.Labels)
		assert.Empty(t, eventList[0].Old)
		assert.Equal(t, "label1", strings.Split(eventList[0].New, ",")[0])

		label = []string{"label2"}
		wiNew.Fields[workitem.SystemLabels] = label
		wiNew.Version = fxt.WorkItems[0].Version + 1
		wiNew, err = s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *wiNew, fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemLabels].([]interface{}), 1)
		eventList, err = s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 2)
		assert.Equal(t, eventList[1].Name, event.Labels)
		assert.NotEmpty(t, eventList[1].Old)
		assert.NotEmpty(t, eventList[1].New)
		assert.Equal(t, "label1", strings.Split(eventList[0].New, ",")[0])
		assert.Equal(t, "label2", strings.Split(eventList[1].New, ",")[0])
	})

	s.T().Run("event label - previous label nil", func(t *testing.T) {

		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))

		label := []string{"label1"}

		fxt.WorkItems[0].Fields[workitem.SystemLabels] = label
		wiNew, err := s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemLabels].([]interface{}), 1)
		eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 1)
		assert.Equal(t, eventList[0].Name, event.Labels)
		assert.Empty(t, eventList[0].Old)
	})

	s.T().Run("event label - new label nil", func(t *testing.T) {

		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))

		label := []string{"label1"}

		fxt.WorkItems[0].Fields[workitem.SystemLabels] = label
		wiNew, err := s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemLabels].([]interface{}), 1)
		eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 1)
		assert.Equal(t, eventList[0].Name, event.Labels)
		assert.Empty(t, eventList[0].Old)

		wiNew.Fields[workitem.SystemLabels] = []string{}
		wiNew.Version = fxt.WorkItems[0].Version + 1
		wiNew, err = s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *wiNew, fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemLabels].([]interface{}), 0)
		eventList, err = s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 2)
		assert.Equal(t, eventList[1].Name, event.Labels)
		assert.Empty(t, eventList[1].New)
	})

	s.T().Run("iteration changed", func(t *testing.T) {

		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1), tf.Iterations(2))
		fxt.WorkItems[0].Fields[workitem.SystemIteration] = fxt.Iterations[0].ID.String()
		wiNew, err := s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		require.NoError(t, err)
		eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 1)
		assert.Equal(t, eventList[0].Name, event.Iteration)
		assert.Empty(t, eventList[0].Old)

		wiNew.Fields[workitem.SystemIteration] = fxt.Iterations[1].ID.String()
		wiNew.Version = fxt.WorkItems[0].Version + 1
		wiNew, err = s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *wiNew, fxt.Identities[0].ID)
		require.NoError(t, err)
		eventList, err = s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.Len(t, eventList, 2)
		assert.Equal(t, eventList[1].Name, event.Iteration)
	})

	s.T().Run("multiple events", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1))
		label := []string{"label1"}
		fxt.WorkItems[0].Fields[workitem.SystemLabels] = label
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateResolved
		_, err := s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		require.NoError(t, err)
		eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 2)
		assert.Equal(t, event.State, eventList[0].Name)
		assert.Equal(t, "new", eventList[0].Old)
		assert.Equal(t, event.Labels, eventList[1].Name)
		assert.Empty(t, eventList[1].Old)
		assert.Equal(t, eventList[1].ID, eventList[0].ID)
	})
}
