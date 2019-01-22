package actions

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSuiteAction(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &ActionSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

type ActionSuite struct {
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

func (s *ActionSuite) TestChangeSet() {
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))

	s.T().Run("different ID", func(t *testing.T) {
		_, err := fxt.WorkItems[0].ChangeSet(*fxt.WorkItems[1])
		require.Error(t, err)
	})

	s.T().Run("same instance", func(t *testing.T) {
		changes, err := fxt.WorkItems[0].ChangeSet(*fxt.WorkItems[0])
		require.NoError(t, err)
		require.Empty(t, changes)
	})

	s.T().Run("no changes, same column order", func(t *testing.T) {
		wiCopy := createWICopy(*fxt.WorkItems[0], workitem.SystemStateNew, []interface{}{"bcid0", "bcid1"})
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{"bcid0", "bcid1"}
		changes, err := fxt.WorkItems[0].ChangeSet(wiCopy)
		require.NoError(t, err)
		require.Empty(t, changes)
	})

	s.T().Run("no changes, mixed column order", func(t *testing.T) {
		wiCopy := createWICopy(*fxt.WorkItems[0], workitem.SystemStateNew, []interface{}{"bcid1", "bcid0"})
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{"bcid0", "bcid1"}
		changes, err := fxt.WorkItems[0].ChangeSet(wiCopy)
		require.NoError(t, err)
		require.Empty(t, changes)
	})

	s.T().Run("state changes", func(t *testing.T) {
		wiCopy := createWICopy(*fxt.WorkItems[0], workitem.SystemStateNew, []interface{}{"bcid0", "bcid1"})
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateOpen
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{"bcid0", "bcid1"}
		changes, err := fxt.WorkItems[0].ChangeSet(wiCopy)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		require.Equal(t, workitem.SystemState, changes[0].AttributeName)
		require.Equal(t, workitem.SystemStateOpen, changes[0].NewValue)
		require.Equal(t, workitem.SystemStateNew, changes[0].OldValue)
	})

	s.T().Run("column changes", func(t *testing.T) {
		wiCopy := createWICopy(*fxt.WorkItems[0], workitem.SystemStateNew, []interface{}{"bcid0"})
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{"bcid0", "bcid1"}
		changes, err := fxt.WorkItems[0].ChangeSet(wiCopy)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		require.Equal(t, workitem.SystemBoardcolumns, changes[0].AttributeName)
		require.Equal(t, wiCopy.Fields[workitem.SystemBoardcolumns], changes[0].OldValue)
		require.Equal(t, fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns], changes[0].NewValue)
	})

	s.T().Run("multiple changes", func(t *testing.T) {
		wiCopy := createWICopy(*fxt.WorkItems[0], workitem.SystemStateOpen, []interface{}{"bcid0"})
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{"bcid0", "bcid1"}
		changes, err := fxt.WorkItems[0].ChangeSet(wiCopy)
		require.NoError(t, err)
		require.Len(t, changes, 2)
		// we intentionally test the order here as the code under test needs
		// to be expanded later, supporting more changes and this is an
		// integrity test on the current impl.
		require.Equal(t, workitem.SystemState, changes[0].AttributeName)
		require.Equal(t, workitem.SystemStateNew, changes[0].NewValue)
		require.Equal(t, workitem.SystemStateOpen, changes[0].OldValue)
		require.Equal(t, workitem.SystemBoardcolumns, changes[1].AttributeName)
		require.Equal(t, fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns], changes[1].NewValue)
		require.Equal(t, wiCopy.Fields[workitem.SystemBoardcolumns], changes[1].OldValue)
	})

	s.T().Run("new instance", func(t *testing.T) {
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{}
		changes, err := fxt.WorkItems[0].ChangeSet(nil)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		require.Equal(t, workitem.SystemState, changes[0].AttributeName)
		require.Equal(t, workitem.SystemStateNew, changes[0].NewValue)
		require.Nil(t, changes[0].OldValue)
	})
}

func (s *ActionSuite) TestActionExecution() {
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
	userID := fxt.Identities[0].ID

	s.T().Run("by Old New", func(t *testing.T) {
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{"bcid0", "bcid1"}
		newVersion := createWICopy(*fxt.WorkItems[0], workitem.SystemStateOpen, []interface{}{"bcid0", "bcid1"})
		_, changes, err := ExecuteActionsByOldNew(s.Ctx, s.GormDB, userID, fxt.WorkItems[0], newVersion, map[string]string{
			"Nil": "{ noConfig: 'none' }",
		})
		require.NoError(t, err)
		require.Len(t, changes, 0)
	})

	s.T().Run("by ChangeSet", func(t *testing.T) {
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{"bcid0", "bcid1"}
		newVersion := createWICopy(*fxt.WorkItems[0], workitem.SystemStateOpen, []interface{}{"bcid0", "bcid1"})
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.NoError(t, err)
		afterActionWI, changes, err := ExecuteActionsByChangeset(s.Ctx, s.GormDB, userID, newVersion, contextChanges, map[string]string{
			"Nil": "{ noConfig: 'none' }",
		})
		require.NoError(t, err)
		require.Len(t, changes, 0)
		require.Equal(t, workitem.SystemStateOpen, afterActionWI.(workitem.WorkItem).Fields["system_state"])
	})

	s.T().Run("unknown rule", func(t *testing.T) {
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{"bcid0", "bcid1"}
		newVersion := createWICopy(*fxt.WorkItems[0], workitem.SystemStateOpen, []interface{}{"bcid0", "bcid1"})
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.NoError(t, err)
		_, _, err = ExecuteActionsByChangeset(s.Ctx, s.GormDB, userID, newVersion, contextChanges, map[string]string{
			"unknownRule": "{ noConfig: 'none' }",
		})
		require.NotNil(t, err)
	})

	s.T().Run("sideffects", func(t *testing.T) {
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateNew
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{"bcid0", "bcid1"}
		newVersion := createWICopy(*fxt.WorkItems[0], workitem.SystemStateOpen, []interface{}{"bcid0", "bcid1"})
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.NoError(t, err)
		// Intentionally not using a constant here!
		afterActionWI, changes, err := ExecuteActionsByChangeset(s.Ctx, s.GormDB, userID, newVersion, contextChanges, map[string]string{
			"FieldSet": "{ \"system_state\": \"resolved\" }",
		})
		require.NoError(t, err)
		require.Len(t, changes, 1)
		require.Equal(t, workitem.SystemStateResolved, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemState])
	})
}
