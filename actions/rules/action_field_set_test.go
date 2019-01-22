package rules

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/fabric8-services/fabric8-wit/actions/change"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
)

func TestSuiteActionFieldSet(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &ActionFieldSetSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

type ActionFieldSetSuite struct {
	gormtestsupport.DBTestSuite
}

func createWICopy(wi workitem.WorkItem, state string, boardcolumns []interface{}) workitem.WorkItem {
	var wiCopy workitem.WorkItem
	wiCopy.ID = wi.ID
	wiCopy.SpaceID = wi.SpaceID
	wiCopy.Type = wi.Type
	wiCopy.Number = wi.Number
	wiCopy.Fields = map[string]interface{}{}
	for k := range wi.Fields {
		wiCopy.Fields[k] = wi.Fields[k]
	}
	wiCopy.Fields[workitem.SystemState] = state
	wiCopy.Fields[workitem.SystemBoardcolumns] = boardcolumns
	return wiCopy
}

func (s *ActionFieldSetSuite) TestActionExecution() {
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
	require.NotNil(s.T(), fxt)
	require.Len(s.T(), fxt.WorkItems, 2)

	s.T().Run("sideffects", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{"bcid0", "bcid1"}
		newVersion := createWICopy(*fxt.WorkItems[0], workitem.SystemStateOpen, []interface{}{"bcid0", "bcid1"})
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.NoError(t, err)
		action := ActionFieldSet{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges change.Set
		// Not using constants here intentionally.
		afterActionWI, convertChanges, err := action.OnChange(newVersion, contextChanges, "{ \"system.state\": \"resolved\" }", &convertChanges)
		require.NoError(t, err)
		require.Len(t, convertChanges, 1)
		require.Equal(t, workitem.SystemState, convertChanges[0].AttributeName)
		require.Equal(t, workitem.SystemStateOpen, convertChanges[0].OldValue)
		require.Equal(t, workitem.SystemStateResolved, convertChanges[0].NewValue)
		require.Equal(t, workitem.SystemStateResolved, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemState])
	})

	s.T().Run("stacking", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{"bcid0", "bcid1"}
		newVersion := createWICopy(*fxt.WorkItems[0], workitem.SystemStateOpen, []interface{}{"bcid0", "bcid1"})
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.NoError(t, err)
		action := ActionFieldSet{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges change.Set
		// Not using constants here intentionally.
		afterActionWI, convertChanges, err := action.OnChange(newVersion, contextChanges, "{ \"system.state\": \"resolved\" }", &convertChanges)
		require.NoError(t, err)
		require.Len(t, convertChanges, 1)
		require.Equal(t, workitem.SystemState, convertChanges[0].AttributeName)
		require.Equal(t, workitem.SystemStateOpen, convertChanges[0].OldValue)
		require.Equal(t, workitem.SystemStateResolved, convertChanges[0].NewValue)
		require.Equal(t, workitem.SystemStateResolved, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemState])
		// doing another change, the convertChange needs to stack.
		afterActionWI, convertChanges, err = action.OnChange(afterActionWI, change.Set{}, "{ \"system.state\": \"new\" }", &convertChanges)
		require.NoError(t, err)
		require.Len(t, convertChanges, 2)
		require.Equal(t, workitem.SystemState, convertChanges[0].AttributeName)
		require.Equal(t, workitem.SystemStateOpen, convertChanges[0].OldValue)
		require.Equal(t, workitem.SystemStateResolved, convertChanges[0].NewValue)
		require.Equal(t, workitem.SystemState, convertChanges[1].AttributeName)
		require.Equal(t, workitem.SystemStateResolved, convertChanges[1].OldValue)
		require.Equal(t, workitem.SystemStateNew, convertChanges[1].NewValue)
		require.Equal(t, workitem.SystemStateNew, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemState])
	})

	s.T().Run("unknown field", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{"bcid0", "bcid1"}
		newVersion := createWICopy(*fxt.WorkItems[0], workitem.SystemStateOpen, []interface{}{"bcid0", "bcid1"})
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.NoError(t, err)
		action := ActionFieldSet{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges change.Set
		_, _, err = action.OnChange(newVersion, contextChanges, "{ \"system.notavailable\": \"updatedState\" }", &convertChanges)
		require.NotNil(t, err)
	})

	s.T().Run("non-json configuration", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{"bcid0", "bcid1"}
		newVersion := createWICopy(*fxt.WorkItems[0], workitem.SystemStateOpen, []interface{}{"bcid0", "bcid1"})
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.NoError(t, err)
		action := ActionFieldSet{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges change.Set
		_, convertChanges, err = action.OnChange(newVersion, contextChanges, "someNonJSON", &convertChanges)
		require.NotNil(t, err)
	})
}
