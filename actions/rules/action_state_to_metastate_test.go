package rules

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
)

func TestSuiteActionStateToMetastate(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &ActionStateToMetastateSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

type ActionStateToMetastateSuite struct {
	suite.Suite
	gormtestsupport.DBTestSuite
}

func (s *ActionStateToMetastateSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
}

func (s *ActionStateToMetastateSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
}

func (s *ActionStateToMetastateSuite) TestActionExecution() {
	// Note on the fixture: by default, the created board is attached to the type group
	// with the same index. The columns on each board are as follows:
	//   0 "New" "mNew"
	//   1 "In Progress" "mInprogress"
	//   2 "Resolved" "mResolved"
	//   3 "Approved" "mResolved"
	// All columns are set to the "BidirectionalStateToColumn" rule. The type has
	// the following states/metastates:
	//   "new"         "mNew"
	//   "open"        "mOpen"
	//   "in progress" "mInprogress"
	//   "resolved"    "mResolved"
	//   "closed"      "mClosed"
	fxtFn := func(t *testing.T) *tf.TestFixture {
		// this sets up a work item in state "new" with matching metastate "mNew" and the corresponding column 0 ("New")
		// this represents a single work item in an ordered "new" state.
		return tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItemBoards(1), tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemState] = "new"
			fxt.WorkItems[idx].Fields[workitem.SystemMetaState] = "mNew"
			fxt.WorkItems[idx].Fields[workitem.SystemBoardcolumns] = []string{
				fxt.WorkItemBoards[idx].Columns[0].ID.String(),
			}
			return nil
		}))
	}
	// test the fixture creation.
	fxt := fxtFn(s.T())
	require.NotNil(s.T(), fxt)
	require.Len(s.T(), fxt.WorkItems, 1)
	require.Equal(s.T(), "mNew", fxt.WorkItems[0].Fields[workitem.SystemMetaState])
	require.Equal(s.T(), "new", fxt.WorkItems[0].Fields[workitem.SystemState])
	require.Equal(s.T(), fxt.WorkItemBoards[0].Columns[0].ID.String(), fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns].([]interface{})[0])

	s.T().Run("updating the state for an existing work item", func(t *testing.T) {
		fxt := fxtFn(s.T())
		// set the state to "in progress" and create a changeset.
		fxt.WorkItems[0].Fields[workitem.SystemState] = "in progress"
		contextChanges := []convert.Change{
			convert.Change{
				AttributeName: workitem.SystemState,
				OldValue: "new",
				NewValue: "in progress",
			},
		}
		// run the test.
		action := ActionStateToMetaState {
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges []convert.Change
		// note: changing the state does not require a configuration.
		afterActionWI, convertChanges, err := action.OnChange(*fxt.WorkItems[0], contextChanges, "", &convertChanges)
		require.Nil(s.T(), err)
		require.Len(s.T(), convertChanges, 2)
		// check metastate validity.
		require.Equal(s.T(), convertChanges[0].AttributeName, "system.metastate")
		require.Equal(s.T(), convertChanges[0].OldValue, "mNew")
		require.Equal(s.T(), convertChanges[0].NewValue, "mInprogress")
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.metastate"], "mInprogress")
		// check column validity.
		require.Equal(s.T(), convertChanges[1].AttributeName, "system.boardcolumns")
		require.Len(s.T(), convertChanges[1].OldValue, 1)
		require.Equal(s.T(), convertChanges[1].OldValue.([]interface{})[0], fxt.WorkItemBoards[0].Columns[0].ID.String() )
		require.Equal(s.T(), convertChanges[1].NewValue.([]interface{})[0], fxt.WorkItemBoards[0].Columns[1].ID.String() )
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.metastate"], "mInprogress")
		require.Len(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.boardcolumns"].([]interface{}), 1)
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.boardcolumns"].([]interface{})[0], fxt.WorkItemBoards[0].Columns[1].ID.String())
	})

	s.T().Run("updating the state for a vanilla work item", func(t *testing.T) {
		fxt := fxtFn(s.T())
		// this should be a vanilla work item, where metastate and boardcolumns is nil
		delete(fxt.WorkItems[0].Fields, workitem.SystemMetaState)
		delete(fxt.WorkItems[0].Fields, workitem.SystemBoardcolumns)
		// set the state to "in progress" and create a changeset.
		fxt.WorkItems[0].Fields[workitem.SystemState] = "in progress"
		contextChanges := []convert.Change{
			convert.Change{
				AttributeName: workitem.SystemState,
				OldValue: nil,
				NewValue: "in progress",
			},
		}
		// run the test.
		action := ActionStateToMetaState {
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges []convert.Change
		// note: changing the state does not require a configuration.
		afterActionWI, convertChanges, err := action.OnChange(*fxt.WorkItems[0], contextChanges, "", &convertChanges)
		require.Nil(s.T(), err)
		require.Len(s.T(), convertChanges, 2)
		// check metastate validity.
		require.Equal(s.T(), convertChanges[0].AttributeName, "system.metastate")
		require.Nil(s.T(), convertChanges[0].OldValue)
		require.Equal(s.T(), convertChanges[0].NewValue, "mInprogress")
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.metastate"], "mInprogress")
		// check column validity.
		require.Equal(s.T(), convertChanges[1].AttributeName, "system.boardcolumns")
		require.Empty(s.T(), convertChanges[1].OldValue)
		require.Equal(s.T(), convertChanges[1].NewValue.([]interface{})[0], fxt.WorkItemBoards[0].Columns[1].ID.String() )
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.metastate"], "mInprogress")
		require.Len(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.boardcolumns"].([]interface{}), 1)
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.boardcolumns"].([]interface{})[0], fxt.WorkItemBoards[0].Columns[1].ID.String())
	})

	s.T().Run("updating the state for a work item with multiple metastate mappings on columns", func(t *testing.T) {
		fxt := fxtFn(s.T())
		// this should be a vanilla work item, where metastate and boardcolumns is nil
		delete(fxt.WorkItems[0].Fields, workitem.SystemMetaState)
		delete(fxt.WorkItems[0].Fields, workitem.SystemBoardcolumns)
		// set the state to "resolved" and create a changeset. 
		fxt.WorkItems[0].Fields[workitem.SystemState] = "resolved"
		contextChanges := []convert.Change{
			convert.Change{
				AttributeName: workitem.SystemState,
				OldValue: nil,
				NewValue: "resolved",
			},
		}
		// run the test.
		action := ActionStateToMetaState {
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges []convert.Change
		// note: changing the state does not require a configuration.
		afterActionWI, convertChanges, err := action.OnChange(*fxt.WorkItems[0], contextChanges, "", &convertChanges)
		require.Nil(s.T(), err)
		require.Len(s.T(), convertChanges, 2)
		// check metastate validity.
		require.Equal(s.T(), convertChanges[0].AttributeName, "system.metastate")
		require.Nil(s.T(), convertChanges[0].OldValue)
		require.Equal(s.T(), convertChanges[0].NewValue, "mResolved")
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.metastate"], "mResolved")
		// check column validity. For the resolved state, two columns are matching,
		// but only the first one (column 2) should be used and available in the WI.
		require.Equal(s.T(), convertChanges[1].AttributeName, "system.boardcolumns")
		require.Empty(s.T(), convertChanges[1].OldValue)
		require.Equal(s.T(), convertChanges[1].NewValue.([]interface{})[0], fxt.WorkItemBoards[0].Columns[2].ID.String() )
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.metastate"], "mResolved")
		require.Len(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.boardcolumns"].([]interface{}), 1)
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.boardcolumns"].([]interface{})[0], fxt.WorkItemBoards[0].Columns[2].ID.String())
	})

	s.T().Run("updating the columns for an existing work item", func(t *testing.T) {
		fxt := fxtFn(s.T())
		// set the column to the "in progress" column and create a changeset.
		contextChanges := []convert.Change{
			convert.Change{
				AttributeName: workitem.SystemBoardcolumns,
				OldValue: fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns],
				NewValue: []interface{}{ fxt.WorkItemBoards[0].Columns[1].ID.String() },
			},
		}
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{ fxt.WorkItemBoards[0].Columns[1].ID.String() }
		// run the test.
		action := ActionStateToMetaState {
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges []convert.Change
		// note: changing the state does not require a configuration.
		afterActionWI, convertChanges, err := action.OnChange(*fxt.WorkItems[0], contextChanges, "", &convertChanges)
		require.Nil(s.T(), err)
		require.Len(s.T(), convertChanges, 2)
		// check metastate validity.
		require.Equal(s.T(), convertChanges[0].AttributeName, "system.metastate")
		require.Equal(s.T(), convertChanges[0].OldValue, "mNew")
		require.Equal(s.T(), convertChanges[0].NewValue, "mInprogress")
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.metastate"], "mInprogress")
		// check state validity.
		require.Equal(s.T(), convertChanges[1].AttributeName, "system.state")
		require.Equal(s.T(), convertChanges[1].OldValue, "new")
		require.Equal(s.T(), convertChanges[1].NewValue, "in progress")
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.state"], "in progress")
	})

	s.T().Run("updating the columns for a vanilla work item", func(t *testing.T) {
		fxt := fxtFn(s.T())
		// set the column to the "in progress" column and create a changeset.
		contextChanges := []convert.Change{
			convert.Change{
				AttributeName: workitem.SystemBoardcolumns,
				OldValue: nil,
				NewValue: []interface{}{ fxt.WorkItemBoards[0].Columns[1].ID.String() },
			},
		}
		delete(fxt.WorkItems[0].Fields, workitem.SystemMetaState)
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{ fxt.WorkItemBoards[0].Columns[1].ID.String() }
		// run the test.
		action := ActionStateToMetaState {
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges []convert.Change
		// note: changing the state does not require a configuration.
		afterActionWI, convertChanges, err := action.OnChange(*fxt.WorkItems[0], contextChanges, "", &convertChanges)
		require.Nil(s.T(), err)
		require.Len(s.T(), convertChanges, 2)
		// check metastate validity.
		require.Equal(s.T(), convertChanges[0].AttributeName, "system.metastate")
		require.Nil(s.T(), convertChanges[0].OldValue)
		require.Equal(s.T(), convertChanges[0].NewValue, "mInprogress")
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.metastate"], "mInprogress")
		// check state validity.
		require.Equal(s.T(), convertChanges[1].AttributeName, "system.state")
		require.Equal(s.T(), convertChanges[1].OldValue, "new")
		require.Equal(s.T(), convertChanges[1].NewValue, "in progress")
		require.Equal(s.T(), afterActionWI.(workitem.WorkItem).Fields["system.state"], "in progress")
	})

}
