package actions

import (
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"

	uuid "github.com/satori/go.uuid"
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

func createWICopy(ID uuid.UUID, state string, boardcolumns []string) workitem.WorkItem {
	var wiCopy workitem.WorkItem
	wiCopy.ID = ID
	fields := map[string]interface{}{
		"system.state":        state,
		"system.boardcolumns": boardcolumns,
	}
	wiCopy.Fields = fields
	return wiCopy
}

func (s *ActionSuite) TestChangeSet() {
	fixture := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
	require.NotNil(s.T(), fixture)
	require.Len(s.T(), fixture.WorkItems, 2)

	s.T().Run("different ID", func(t *testing.T) {
		_, err := fixture.WorkItems[0].ChangeSet(*fixture.WorkItems[1])
		require.NotNil(t, err)
	})

	s.T().Run("same instance", func(t *testing.T) {
		changes, err := fixture.WorkItems[0].ChangeSet(*fixture.WorkItems[0])
		require.Nil(t, err)
		require.Empty(t, changes)
	})

	s.T().Run("no changes, same column order", func(t *testing.T) {
		wiCopy := createWICopy(fixture.WorkItems[0].ID, "new", []string{"bcid0", "bcid1"})
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		changes, err := fixture.WorkItems[0].ChangeSet(wiCopy)
		require.Nil(t, err)
		require.Empty(t, changes)
	})

	s.T().Run("no changes, mixed column order", func(t *testing.T) {
		wiCopy := createWICopy(fixture.WorkItems[0].ID, "new", []string{"bcid1", "bcid0"})
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		changes, err := fixture.WorkItems[0].ChangeSet(wiCopy)
		require.Nil(t, err)
		require.Empty(t, changes)
	})

	s.T().Run("state changes", func(t *testing.T) {
		wiCopy := createWICopy(fixture.WorkItems[0].ID, "new", []string{"bcid0", "bcid1"})
		fixture.WorkItems[0].Fields["system.state"] = "open"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		changes, err := fixture.WorkItems[0].ChangeSet(wiCopy)
		require.Nil(t, err)
		require.Len(t, changes, 1)
		require.Equal(t, "system.state", changes[0].AttributeName)
		require.Equal(t, "open", changes[0].NewValue)
		require.Equal(t, "new", changes[0].OldValue)
	})

	s.T().Run("column changes", func(t *testing.T) {
		wiCopy := createWICopy(fixture.WorkItems[0].ID, "new", []string{"bcid0"})
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		changes, err := fixture.WorkItems[0].ChangeSet(wiCopy)
		require.Nil(t, err)
		require.Len(t, changes, 1)
		fmt.Println(changes[0].OldValue)
		fmt.Println(changes[0].NewValue)
		require.Equal(t, "system.boardcolumns", changes[0].AttributeName)
		require.Equal(t, wiCopy.Fields["system.boardcolumns"], changes[0].OldValue)
		require.Equal(t, fixture.WorkItems[0].Fields["system.boardcolumns"], changes[0].NewValue)
	})

	s.T().Run("multiple changes", func(t *testing.T) {
		wiCopy := createWICopy(fixture.WorkItems[0].ID, "open", []string{"bcid0"})
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		changes, err := fixture.WorkItems[0].ChangeSet(wiCopy)
		require.Nil(t, err)
		require.Len(t, changes, 2)
		// we intentionally test the order here as the code under test needs
		// to be expanded later, supporting more changes and this is an
		// integrity test on the current impl.
		require.Equal(t, "system.state", changes[0].AttributeName)
		require.Equal(t, "new", changes[0].NewValue)
		require.Equal(t, "open", changes[0].OldValue)
		require.Equal(t, "system.boardcolumns", changes[1].AttributeName)
		require.Equal(t, fixture.WorkItems[0].Fields["system.boardcolumns"], changes[1].NewValue)
		require.Equal(t, wiCopy.Fields["system.boardcolumns"], changes[1].OldValue)
	})
}

func (s *ActionSuite) TestActionExecution() {
	fixture := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
	require.NotNil(s.T(), fixture)
	require.Len(s.T(), fixture.WorkItems, 2)
	userID := fixture.Identities[0].ID

	s.T().Run("by Old New", func(t *testing.T) {
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		newVersion := createWICopy(fixture.WorkItems[0].ID, "open", []string{"bcid0", "bcid1"})
		afterActionWI, changes, err := ExecuteActionsByOldNew(s.Ctx, s.GormDB, userID, fixture.WorkItems[0], newVersion, map[string]string{
			"nilRule": "{ noConfig: 'none' }",
		})
		require.Nil(t, err)
		require.Len(t, changes, 1)
		require.Equal(t, afterActionWI.(workitem.WorkItem).Fields["system.state"], "open")
	})

	s.T().Run("by ChangeSet", func(t *testing.T) {
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		newVersion := createWICopy(fixture.WorkItems[0].ID, "open", []string{"bcid0", "bcid1"})
		contextChanges, err := fixture.WorkItems[0].ChangeSet(newVersion)
		require.Nil(t, err)
		afterActionWI, changes, err := ExecuteActionsByChangeset(s.Ctx, s.GormDB, userID, newVersion, contextChanges, map[string]string{
			"nilRule": "{ noConfig: 'none' }",
		})
		require.Nil(t, err)
		require.Len(t, changes, 1)
		require.Equal(t, afterActionWI.(workitem.WorkItem).Fields["system.state"], "open")
	})

	s.T().Run("unknown rule", func(t *testing.T) {
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		newVersion := createWICopy(fixture.WorkItems[0].ID, "open", []string{"bcid0", "bcid1"})
		contextChanges, err := fixture.WorkItems[0].ChangeSet(newVersion)
		require.Nil(t, err)
		_, _, err = ExecuteActionsByChangeset(s.Ctx, s.GormDB, userID, newVersion, contextChanges, map[string]string{
			"unknownRule": "{ noConfig: 'none' }",
		})
		require.NotNil(t, err)
	})

	s.T().Run("sideffects", func(t *testing.T) {
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		newVersion := createWICopy(fixture.WorkItems[0].ID, "open", []string{"bcid0", "bcid1"})
		contextChanges, err := fixture.WorkItems[0].ChangeSet(newVersion)
		require.Nil(t, err)
		afterActionWI, changes, err := ExecuteActionsByChangeset(s.Ctx, s.GormDB, userID, newVersion, contextChanges, map[string]string{
			"FieldSetRule": "{ system.state: 'updatedState' }",
		})
		require.Nil(t, err)
		require.Len(t, changes, 1)
		require.Equal(t, "updatedState", afterActionWI.(workitem.WorkItem).Fields["system.state"])
	})
}
