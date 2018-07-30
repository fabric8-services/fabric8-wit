package rules

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/fabric8-services/fabric8-wit/convert"
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
	wiCopy.Fields["system.state"] = state
	wiCopy.Fields["system.boardcolumns"] = boardcolumns
	return wiCopy
}

func (s *ActionFieldSetSuite) TestActionExecution() {
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
	require.NotNil(s.T(), fxt)
	require.Len(s.T(), fxt.WorkItems, 2)

	s.T().Run("sideffects", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
		fxt.WorkItems[0].Fields["system.state"] = "new"
		fxt.WorkItems[0].Fields["system.boardcolumns"] = []interface{}{"bcid0", "bcid1"}
		newVersion := createWICopy(*fxt.WorkItems[0], "open", []interface{}{"bcid0", "bcid1"})
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.Nil(t, err)
		action := ActionFieldSet{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges []convert.Change
		afterActionWI, convertChanges, err := action.OnChange(newVersion, contextChanges, "{ \"system.state\": \"resolved\" }", &convertChanges)
		require.Nil(t, err)
		require.Len(t, convertChanges, 1)
		require.Equal(t, "system.state", convertChanges[0].AttributeName)
		require.Equal(t, "open", convertChanges[0].OldValue)
		require.Equal(t, "resolved", convertChanges[0].NewValue)
		require.Equal(t, "resolved", afterActionWI.(workitem.WorkItem).Fields["system.state"])
	})

	s.T().Run("unknown field", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
		fxt.WorkItems[0].Fields["system.state"] = "new"
		fxt.WorkItems[0].Fields["system.boardcolumns"] = []interface{}{"bcid0", "bcid1"}
		newVersion := createWICopy(*fxt.WorkItems[0], "open", []interface{}{"bcid0", "bcid1"})
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.Nil(t, err)
		action := ActionFieldSet{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges []convert.Change
		_, _, err = action.OnChange(newVersion, contextChanges, "{ \"system.notavailable\": \"updatedState\" }", &convertChanges)
		require.NotNil(t, err)
	})

	s.T().Run("non-json configuration", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2))
		fxt.WorkItems[0].Fields["system.state"] = "new"
		fxt.WorkItems[0].Fields["system.boardcolumns"] = []interface{}{"bcid0", "bcid1"}
		newVersion := createWICopy(*fxt.WorkItems[0], "open", []interface{}{"bcid0", "bcid1"})
		contextChanges, err := fxt.WorkItems[0].ChangeSet(newVersion)
		require.Nil(t, err)
		action := ActionFieldSet{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges []convert.Change
		_, convertChanges, err = action.OnChange(newVersion, contextChanges, "someNonJSON", &convertChanges)
		require.NotNil(t, err)
	})
}
