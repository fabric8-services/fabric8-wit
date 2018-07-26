package rules

import (
	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
)

func TestSuiteActionFieldSet(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &ActionFieldSetSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

type ActionFieldSetSuite struct {
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

func (s *ActionFieldSetSuite) TestActionExecution() {
	fixture := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
	require.NotNil(s.T(), fixture)
	require.Len(s.T(), fixture.WorkItems, 2)

	s.T().Run("sideffects", func(t *testing.T) {
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		newVersion := createWICopy(fixture.WorkItems[0].ID, "open", []string{"bcid0", "bcid1"})
		contextChanges, err := fixture.WorkItems[0].ChangeSet(newVersion)
		require.Nil(s.T(), err)
		var action ActionFieldSet
		var convertChanges []convert.Change
		afterActionWI, convertChanges, err := action.OnChange(newVersion, contextChanges, "{ \"system.state\": \"updatedState\" }", &convertChanges)
		require.Nil(s.T(), err)
		require.Len(s.T(), convertChanges, 1)
		require.Equal(s.T(), "system.state", convertChanges[0].AttributeName)
		require.Equal(s.T(), "open", convertChanges[0].OldValue)
		require.Equal(s.T(), "updatedState", convertChanges[0].NewValue)
		require.Equal(s.T(), "updatedState", afterActionWI.(workitem.WorkItem).Fields["system.state"])
	})

	s.T().Run("unknown field", func(t *testing.T) {
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		newVersion := createWICopy(fixture.WorkItems[0].ID, "open", []string{"bcid0", "bcid1"})
		contextChanges, err := fixture.WorkItems[0].ChangeSet(newVersion)
		require.Nil(s.T(), err)
		var action ActionFieldSet
		var convertChanges []convert.Change
		afterActionWI, convertChanges, err := action.OnChange(newVersion, contextChanges, "{ \"system.notavailable\": \"updatedState\" }", &convertChanges)
		require.Nil(s.T(), err)
		require.Len(s.T(), convertChanges, 1)
		require.Equal(s.T(), "system.notavailable", convertChanges[0].AttributeName)
		require.Equal(s.T(), nil, convertChanges[0].OldValue)
		require.Equal(s.T(), "updatedState", convertChanges[0].NewValue)
		require.Equal(s.T(), "updatedState", afterActionWI.(workitem.WorkItem).Fields["system.notavailable"])
	})

	s.T().Run("non-json configuration", func(t *testing.T) {
		fixture.WorkItems[0].Fields["system.state"] = "new"
		fixture.WorkItems[0].Fields["system.boardcolumns"] = []string{"bcid0", "bcid1"}
		newVersion := createWICopy(fixture.WorkItems[0].ID, "open", []string{"bcid0", "bcid1"})
		contextChanges, err := fixture.WorkItems[0].ChangeSet(newVersion)
		require.Nil(s.T(), err)
		var action ActionFieldSet
		var convertChanges []convert.Change
		_, convertChanges, err = action.OnChange(newVersion, contextChanges, "someNonJSON", &convertChanges)
		require.NotNil(s.T(), err)
	})
}
