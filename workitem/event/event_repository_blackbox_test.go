package event_test

import (
	"strconv"
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/rendering"
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
		DBTestSuite: gormtestsupport.NewDBTestSuite(),
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
		assert.Equal(t, eventList[0].Name, workitem.SystemAssignees)
		assert.Empty(t, eventList[0].Old)
		assert.Equal(t, fxt.Identities[0].ID.String(), eventList[0].New.([]interface{})[0])

		assignee = []string{fxt.Identities[1].ID.String()}
		wiNew.Fields[workitem.SystemAssignees] = assignee
		wiNew.Version = fxt.WorkItems[0].Version + 1
		wiNew, err = s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *wiNew, fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemAssignees].([]interface{}), 1)
		eventList, err = s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 2)
		assert.Equal(t, eventList[1].Name, workitem.SystemAssignees)
		assert.NotEmpty(t, eventList[1].Old)
		assert.NotEmpty(t, eventList[1].New)
		assert.Equal(t, fxt.Identities[0].ID.String(), eventList[0].New.([]interface{})[0])
		assert.Equal(t, fxt.Identities[1].ID.String(), eventList[1].New.([]interface{})[0])
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
		assert.Equal(t, eventList[0].Name, workitem.SystemAssignees)
		assert.Empty(t, eventList[0].Old)
		assert.Equal(t, fxt.Identities[0].ID.String(), eventList[0].New.([]interface{})[0])
	})

	s.T().Run("event description", func(t *testing.T) {
		oldDescription := rendering.NewMarkupContentFromLegacy("description1")
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemDescription] = oldDescription
			return nil
		}))
		newDescription := rendering.NewMarkupContentFromLegacy("description2")
		fxt.WorkItems[0].Fields[workitem.SystemDescription] = newDescription
		wiNew, err := s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
		require.NoError(t, err)
		eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NoError(t, err)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 1)
		require.Equal(t, "description1", eventList[0].Old)
		require.Equal(t, "description2", eventList[0].New)
		require.Equal(t, wiNew.Fields[workitem.SystemDescription], newDescription)
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
		assert.Equal(t, eventList[0].Name, workitem.SystemAssignees)
		assert.Empty(t, eventList[0].Old)
		assert.Equal(t, fxt.Identities[0].ID.String(), eventList[0].New.([]interface{})[0])

		wiNew.Fields[workitem.SystemAssignees] = []string{}
		wiNew.Version = fxt.WorkItems[0].Version + 1
		wiNew, err = s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *wiNew, fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemAssignees].([]interface{}), 0)
		eventList, err = s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 2)
		assert.Equal(t, eventList[1].Name, workitem.SystemAssignees)
		assert.Empty(t, eventList[1].New)
	})

	s.T().Run("event assignee - old assignee not nil & new assignee not nil", func(t *testing.T) {
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
		assert.Equal(t, eventList[0].Name, workitem.SystemAssignees)
		assert.Empty(t, eventList[0].Old)
		assert.Equal(t, fxt.Identities[0].ID.String(), eventList[0].New.([]interface{})[0])

		wiNew.Fields[workitem.SystemAssignees] = []string{fxt.Identities[1].ID.String()}
		wiNew.Version = fxt.WorkItems[0].Version + 1
		wiNew, err = s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *wiNew, fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemAssignees].([]interface{}), 1)
		eventList, err = s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 2)
		assert.Equal(t, eventList[1].Name, workitem.SystemAssignees)
		assert.Equal(t, fxt.Identities[1].ID.String(), eventList[1].New.([]interface{})[0])
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
		assert.Equal(t, eventList[0].Name, workitem.SystemState)
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
		assert.Equal(t, eventList[0].Name, workitem.SystemLabels)
		assert.Empty(t, eventList[0].Old)
		assert.Equal(t, "label1", eventList[0].New.([]interface{})[0])

		label = []string{"label2"}
		wiNew.Fields[workitem.SystemLabels] = label
		wiNew.Version = fxt.WorkItems[0].Version + 1
		wiNew, err = s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *wiNew, fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemLabels].([]interface{}), 1)
		eventList, err = s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 2)
		assert.Equal(t, eventList[1].Name, workitem.SystemLabels)
		assert.NotEmpty(t, eventList[1].Old)
		assert.NotEmpty(t, eventList[1].New)
		assert.Equal(t, "label1", eventList[0].New.([]interface{})[0])
		assert.Equal(t, "label2", eventList[1].New.([]interface{})[0])
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
		assert.Equal(t, eventList[0].Name, workitem.SystemLabels)
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
		assert.Equal(t, eventList[0].Name, workitem.SystemLabels)
		assert.Empty(t, eventList[0].Old)

		wiNew.Fields[workitem.SystemLabels] = []string{}
		wiNew.Version = fxt.WorkItems[0].Version + 1
		wiNew, err = s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *wiNew, fxt.Identities[0].ID)
		require.NoError(t, err)
		require.Len(t, wiNew.Fields[workitem.SystemLabels].([]interface{}), 0)
		eventList, err = s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList, 2)
		assert.Equal(t, eventList[1].Name, workitem.SystemLabels)
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
		assert.Equal(t, eventList[0].Name, workitem.SystemIteration)
		assert.Empty(t, eventList[0].Old)

		wiNew.Fields[workitem.SystemIteration] = fxt.Iterations[1].ID.String()
		wiNew.Version = fxt.WorkItems[0].Version + 1
		wiNew, err = s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *wiNew, fxt.Identities[0].ID)
		require.NoError(t, err)
		eventList, err = s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
		require.Len(t, eventList, 2)
		assert.Equal(t, eventList[1].Name, workitem.SystemIteration)
	})

	s.T().Run("Field with Kind", func(t *testing.T) {
		t.Run("Float", func(t *testing.T) {
			initialValue := 12.3
			updatedValue := 49.56
			fieldName := "foo"
			fxt := tf.NewTestFixture(t, s.DB,
				tf.WorkItemTypes(1, func(fxt *tf.TestFixture, idx int) error {
					fxt.WorkItemTypes[idx].Fields = map[string]workitem.FieldDefinition{
						fieldName: {
							Type: &workitem.SimpleType{Kind: workitem.KindFloat},
						},
					}
					return nil
				}),
				tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
					fxt.WorkItems[idx].Type = fxt.WorkItemTypes[0].ID
					fxt.WorkItems[idx].Fields[fieldName] = initialValue
					return nil
				}),
			)
			fxt.WorkItems[0].Fields[fieldName] = updatedValue
			_, err := s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
			require.NoError(t, err)
			eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
			require.Len(t, eventList, 1)
			assert.Equal(t, eventList[0].Name, fieldName)
			oldStr, _ := eventList[0].Old.(string)
			old, _ := strconv.ParseFloat(oldStr, 64)
			assert.Equal(t, old, initialValue)
			newStr, _ := eventList[0].New.(string)
			new, _ := strconv.ParseFloat(newStr, 64)
			assert.Equal(t, new, updatedValue)
		})

		t.Run("Int", func(t *testing.T) {
			initialValue := 12
			updatedValue := 49
			fieldName := "foo"
			fxt := tf.NewTestFixture(t, s.DB,
				tf.WorkItemTypes(1, func(fxt *tf.TestFixture, idx int) error {
					fxt.WorkItemTypes[idx].Fields = map[string]workitem.FieldDefinition{
						fieldName: {
							Type: &workitem.SimpleType{Kind: workitem.KindInteger},
						},
					}
					return nil
				}),
				tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
					fxt.WorkItems[idx].Type = fxt.WorkItemTypes[0].ID
					fxt.WorkItems[idx].Fields[fieldName] = initialValue
					return nil
				}),
			)
			fxt.WorkItems[0].Fields[fieldName] = updatedValue
			_, err := s.wiRepo.Save(s.Ctx, fxt.WorkItems[0].SpaceID, *fxt.WorkItems[0], fxt.Identities[0].ID)
			require.NoError(t, err)
			eventList, err := s.wiEventRepo.List(s.Ctx, fxt.WorkItems[0].ID)
			require.Len(t, eventList, 1)
			assert.Equal(t, eventList[0].Name, fieldName)
			oldStr, _ := eventList[0].Old.(string)
			old, _ := strconv.ParseInt(oldStr, 10, 0)
			assert.EqualValues(t, old, initialValue)
			newStr, _ := eventList[0].New.(string)
			new, _ := strconv.ParseInt(newStr, 10, 0)
			assert.EqualValues(t, new, updatedValue)
		})

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
		c := 0
		for _, k := range eventList {
			switch k.Name {
			case workitem.SystemState:
				c = c + 1
			case workitem.SystemLabels:
				c = c + 1
			}
		}
		assert.Equal(t, 2, c)
	})
}
