package actions

import (
	"fmt"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
)

func TestSuiteAction(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &ActionSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

type ActionSuite struct {
	suite.Suite
	gormtestsupport.DBTestSuite
}

func (s *ActionSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
}

func (s *ActionSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
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
		require.NotNil(s.T(), err)
	})

	s.T().Run("same instance", func(t *testing.T) {
		changes, err := fixture.WorkItems[0].ChangeSet(*fixture.WorkItems[0])
		require.Nil(s.T(), err)
		require.Empty(s.T(), changes)
	})

	s.T().Run("no changes, same column order", func(t *testing.T) {
		wiCopy := createWICopy(fixture.WorkItems[0].ID, "new", []string{"bcid0", "bcid1"})
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		changes, err := fixture.WorkItems[0].ChangeSet(wiCopy)
		require.Nil(s.T(), err)
		require.Empty(s.T(), changes)
	})

	s.T().Run("no changes, mixed column order", func(t *testing.T) {
		wiCopy := createWICopy(fixture.WorkItems[0].ID, "new", []string{"bcid1", "bcid0"})
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		changes, err := fixture.WorkItems[0].ChangeSet(wiCopy)
		require.Nil(s.T(), err)
		require.Empty(s.T(), changes)
	})

	s.T().Run("state changes", func(t *testing.T) {
		wiCopy := createWICopy(fixture.WorkItems[0].ID, "new", []string{"bcid0", "bcid1"})
		fixture.WorkItems[0].Fields["system.state"] = "open"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		changes, err := fixture.WorkItems[0].ChangeSet(wiCopy)
		require.Nil(s.T(), err)
		require.Len(s.T(), changes, 1)
		require.Equal(s.T(), changes[0].AttributeName, "system.state")
		require.Equal(s.T(), changes[0].NewValue, "open")
		require.Equal(s.T(), changes[0].OldValue, "new")
	})

	s.T().Run("column changes", func(t *testing.T) {
		wiCopy := createWICopy(fixture.WorkItems[0].ID, "new", []string{"bcid0"})
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		changes, err := fixture.WorkItems[0].ChangeSet(wiCopy)
		require.Nil(s.T(), err)
		require.Len(s.T(), changes, 1)
		fmt.Println(changes[0].OldValue)
		fmt.Println(changes[0].NewValue)
		require.Equal(s.T(), changes[0].AttributeName, "system.boardcolumns")
		require.Equal(s.T(), changes[0].OldValue, wiCopy.Fields["system.boardcolumns"])
		require.Equal(s.T(), changes[0].NewValue, fixture.WorkItems[0].Fields["system.boardcolumns"])
	})

	s.T().Run("multiple changes", func(t *testing.T) {
		wiCopy := createWICopy(fixture.WorkItems[0].ID, "open", []string{"bcid0"})
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		changes, err := fixture.WorkItems[0].ChangeSet(wiCopy)
		require.Nil(s.T(), err)
		require.Len(s.T(), changes, 2)
		// we intentionally test the order here as the code under test needs
		// to be expanded later, supporting more changes and this is an
		// integrity test on the current impl.
		require.Equal(s.T(), changes[0].AttributeName, "system.state")
		require.Equal(s.T(), changes[0].NewValue, "new")
		require.Equal(s.T(), changes[0].OldValue, "open")
		require.Equal(s.T(), changes[1].AttributeName, "system.boardcolumns")
		require.Equal(s.T(), changes[1].NewValue, fixture.WorkItems[0].Fields["system.boardcolumns"])
		require.Equal(s.T(), changes[1].OldValue, wiCopy.Fields["system.boardcolumns"])
	})
}

func (s *ActionSuite) TestActionExecution() {
	fixture := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
	require.NotNil(s.T(), fixture)
	require.Len(s.T(), fixture.WorkItems, 2)

	s.T().Run("by Old New", func(t *testing.T) {
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		newVersion := createWICopy(fixture.WorkItems[0].ID, "open", []string{"bcid0", "bcid1"})
		afterActionWI, changes, err := ExecuteActionsByOldNew(fixture.WorkItems[0], newVersion, map[string]string{
			"nilRule": "{ noConfig: 'none' }",
		})
		require.Nil(s.T(), err)
		require.Len(s.T(), changes, 1)
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.state"], "open")
	})

	s.T().Run("by ChangeSet", func(t *testing.T) {
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		newVersion := createWICopy(fixture.WorkItems[0].ID, "open", []string{"bcid0", "bcid1"})
		contextChanges, err := fixture.WorkItems[0].ChangeSet(newVersion)
		require.Nil(s.T(), err)
		afterActionWI, changes, err := ExecuteActionsByChangeset(newVersion, contextChanges, map[string]string{
			"nilRule": "{ noConfig: 'none' }",
		})
		require.Nil(s.T(), err)
		require.Len(s.T(), changes, 1)
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.state"], "open")
	})

	s.T().Run("unknown rule", func(t *testing.T) {
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		newVersion := createWICopy(fixture.WorkItems[0].ID, "open", []string{"bcid0", "bcid1"})
		contextChanges, err := fixture.WorkItems[0].ChangeSet(newVersion)
		require.Nil(s.T(), err)
		_, _, err = ExecuteActionsByChangeset(newVersion, contextChanges, map[string]string{
			"unknownRule": "{ noConfig: 'none' }",
		})
		require.NotNil(s.T(), err)
	})

	s.T().Run("sideffects", func(t *testing.T) {
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		newVersion := createWICopy(fixture.WorkItems[0].ID, "open", []string{"bcid0", "bcid1"})
		contextChanges, err := fixture.WorkItems[0].ChangeSet(newVersion)
		require.Nil(s.T(), err)
		afterActionWI, changes, err := ExecuteActionsByChangeset(newVersion, contextChanges, map[string]string{
			"FieldSetRule": "{ system.state: 'updatedState' }",
		})
		require.Nil(s.T(), err)
		require.Len(s.T(), changes, 1)
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.state"], "updatedState")
	})
}
