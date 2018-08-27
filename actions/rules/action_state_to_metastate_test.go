package rules

import (
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/actions/change"
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
	gormtestsupport.DBTestSuite
}

func ArrayEquals(a []interface{}, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func (s *ActionStateToMetastateSuite) TestContainsElement() {
	s.T().Run("contains an element", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment())
		action := ActionStateToMetaState{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		a := []interface{}{0, 1, 2, 3, 2}
		require.True(t, action.contains(a, 2))
		require.True(t, action.contains(a, 3))
		require.True(t, action.contains(a, 0))
	})
	s.T().Run("not contains an element", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment())
		action := ActionStateToMetaState{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		a := []interface{}{0, 1, 2, 3, 2}
		require.False(t, action.contains(a, 4))
		require.False(t, action.contains(a, nil))
		require.False(t, action.contains(a, "foo"))
	})
}

func (s *ActionStateToMetastateSuite) TestRemoveElement() {
	s.T().Run("removing an existing element", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment())
		action := ActionStateToMetaState{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		a := []interface{}{0, 1, 2, 3, 2}
		a = action.removeElement(a, 1)
		expected := []interface{}{0, 2, 3, 2}
		require.Len(t, a, 4)
		require.True(t, ArrayEquals(expected, a))
		a = action.removeElement(a, 3)
		expected = []interface{}{0, 2, 2}
		require.Len(t, a, 3)
		require.True(t, ArrayEquals(expected, a))
	})
	s.T().Run("removing a non-existing element", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment())
		action := ActionStateToMetaState{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		a := []interface{}{0, 1, 2, 3, 2}
		a = action.removeElement(a, 4)
		require.Len(t, a, 5)
		expected := []interface{}{0, 1, 2, 3, 2}
		require.True(t, ArrayEquals(expected, a))
	})
	s.T().Run("removing a duplicate element", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment())
		action := ActionStateToMetaState{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		a := []interface{}{0, 1, 2, 3, 2}
		a = action.removeElement(a, 2)
		expected := []interface{}{0, 1, 3}
		require.Len(t, a, 3)
		require.True(t, ArrayEquals(expected, a))
	})
}

func (s *ActionStateToMetastateSuite) TestDifference() {
	s.T().Run("finding differences", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment())
		action := ActionStateToMetaState{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		a := []interface{}{0, 1, 2, 3, 2}
		b := []interface{}{2, 3, 5}
		added, removed := action.difference(a, b)
		require.Len(t, added, 1)
		require.Len(t, removed, 2)
		// wasting plenty more memory here
		var expectedAdded []interface{}
		expectedAdded = append(expectedAdded, 5)
		var expectedRemoved []interface{}
		expectedRemoved = append(expectedRemoved, 0)
		expectedRemoved = append(expectedRemoved, 1)
		require.True(t, ArrayEquals(added, expectedAdded))
		require.True(t, ArrayEquals(removed, expectedRemoved))
	})
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
			fxt.WorkItems[idx].Fields[workitem.SystemState] = workitem.SystemStateNew
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
	require.Equal(s.T(), workitem.SystemStateNew, fxt.WorkItems[0].Fields[workitem.SystemState])
	require.Equal(s.T(), fxt.WorkItemBoards[0].Columns[0].ID.String(), fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns].([]interface{})[0])

	s.T().Run("updating the state for an existing work item", func(t *testing.T) {
		fxt := fxtFn(t)
		// set the state to "in progress" and create a changeset.
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateInProgress
		contextChanges := change.Set{
			{
				AttributeName: workitem.SystemState,
				OldValue:      workitem.SystemStateNew,
				NewValue:      workitem.SystemStateInProgress,
			},
		}
		// run the test.
		action := ActionStateToMetaState{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges change.Set
		// note: the rule does not use the explicit configuration, but reads from the template.
		afterActionWI, convertChanges, err := action.OnChange(*fxt.WorkItems[0], contextChanges, "", &convertChanges)
		require.NoError(t, err)
		require.Len(t, convertChanges, 2)
		// check metastate validity.
		require.Equal(t, workitem.SystemMetaState, convertChanges[0].AttributeName)
		require.Equal(t, "mNew", convertChanges[0].OldValue)
		require.Equal(t, "mInprogress", convertChanges[0].NewValue)
		require.Equal(t, "mInprogress", afterActionWI.(workitem.WorkItem).Fields[workitem.SystemMetaState])
		// check column validity.
		require.Equal(t, workitem.SystemBoardcolumns, convertChanges[1].AttributeName)
		require.Len(t, convertChanges[1].OldValue, 1)
		require.Equal(t, convertChanges[1].OldValue.([]interface{})[0], fxt.WorkItemBoards[0].Columns[0].ID.String())
		require.Equal(t, convertChanges[1].NewValue.([]interface{})[0], fxt.WorkItemBoards[0].Columns[1].ID.String())
		require.Equal(t, "mInprogress", afterActionWI.(workitem.WorkItem).Fields[workitem.SystemMetaState])
		require.Len(t, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemBoardcolumns].([]interface{}), 1)
		require.Equal(t, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemBoardcolumns].([]interface{})[0], fxt.WorkItemBoards[0].Columns[1].ID.String())
	})

	s.T().Run("updating the state for a vanilla work item", func(t *testing.T) {
		fxt := fxtFn(t)
		// this should be a vanilla work item, where metastate and boardcolumns is nil
		delete(fxt.WorkItems[0].Fields, workitem.SystemMetaState)
		delete(fxt.WorkItems[0].Fields, workitem.SystemBoardcolumns)
		// set the state to "in progress" and create a changeset.
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateInProgress
		contextChanges := change.Set{
			{
				AttributeName: workitem.SystemState,
				OldValue:      nil,
				NewValue:      workitem.SystemStateInProgress,
			},
		}
		// run the test.
		action := ActionStateToMetaState{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges change.Set
		// note: the rule does not use the explicit configuration, but reads from the template.
		afterActionWI, convertChanges, err := action.OnChange(*fxt.WorkItems[0], contextChanges, "", &convertChanges)
		require.NoError(t, err)
		require.Len(t, convertChanges, 2)
		// check metastate validity.
		require.Equal(t, workitem.SystemMetaState, convertChanges[0].AttributeName)
		require.Nil(t, convertChanges[0].OldValue)
		require.Equal(t, "mInprogress", convertChanges[0].NewValue)
		require.Equal(t, "mInprogress", afterActionWI.(workitem.WorkItem).Fields[workitem.SystemMetaState])
		// check column validity.
		require.Equal(t, workitem.SystemBoardcolumns, convertChanges[1].AttributeName)
		require.Empty(t, convertChanges[1].OldValue)
		require.Equal(t, fxt.WorkItemBoards[0].Columns[1].ID.String(), convertChanges[1].NewValue.([]interface{})[0])
		require.Equal(t, "mInprogress", afterActionWI.(workitem.WorkItem).Fields[workitem.SystemMetaState])
		require.Len(t, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemBoardcolumns].([]interface{}), 1)
		require.Equal(t, fxt.WorkItemBoards[0].Columns[1].ID.String(), afterActionWI.(workitem.WorkItem).Fields[workitem.SystemBoardcolumns].([]interface{})[0])
	})

	s.T().Run("updating multiple attributes and state", func(t *testing.T) {
		fxt := fxtFn(t)
		// this should be a vanilla work item, where metastate and boardcolumns is nil
		delete(fxt.WorkItems[0].Fields, workitem.SystemMetaState)
		delete(fxt.WorkItems[0].Fields, workitem.SystemBoardcolumns)
		// set the state to "in progress" and create a changeset.
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateInProgress
		fxt.WorkItems[0].Fields[workitem.SystemTitle] = "Updated Title"
		contextChanges := change.Set{
			{
				AttributeName: workitem.SystemTitle,
				OldValue:      nil,
				NewValue:      "Updated Title",
			},
			{
				AttributeName: workitem.SystemState,
				OldValue:      nil,
				NewValue:      workitem.SystemStateInProgress,
			},
		}
		// run the test.
		action := ActionStateToMetaState{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges change.Set
		// note: the rule does not use the explicit configuration, but reads from the template.
		afterActionWI, convertChanges, err := action.OnChange(*fxt.WorkItems[0], contextChanges, "", &convertChanges)
		require.NoError(t, err)
		require.Len(t, convertChanges, 2)
		// check metastate validity.
		require.Equal(t, workitem.SystemMetaState, convertChanges[0].AttributeName)
		require.Nil(t, convertChanges[0].OldValue)
		require.Equal(t, "mInprogress", convertChanges[0].NewValue)
		require.Equal(t, "mInprogress", afterActionWI.(workitem.WorkItem).Fields[workitem.SystemMetaState])
		// check column validity.
		require.Equal(t, workitem.SystemBoardcolumns, convertChanges[1].AttributeName)
		require.Empty(t, convertChanges[1].OldValue)
		require.Equal(t, fxt.WorkItemBoards[0].Columns[1].ID.String(), convertChanges[1].NewValue.([]interface{})[0])
		require.Equal(t, "mInprogress", afterActionWI.(workitem.WorkItem).Fields[workitem.SystemMetaState])
		require.Len(t, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemBoardcolumns].([]interface{}), 1)
		require.Equal(t, fxt.WorkItemBoards[0].Columns[1].ID.String(), afterActionWI.(workitem.WorkItem).Fields[workitem.SystemBoardcolumns].([]interface{})[0])
	})

	s.T().Run("updating the state for a work item with multiple metastate mappings on columns", func(t *testing.T) {
		fxt := fxtFn(t)
		// this should be a vanilla work item, where metastate and boardcolumns is nil
		delete(fxt.WorkItems[0].Fields, workitem.SystemMetaState)
		delete(fxt.WorkItems[0].Fields, workitem.SystemBoardcolumns)
		// set the state to "resolved" and create a changeset.
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateResolved
		contextChanges := change.Set{
			{
				AttributeName: workitem.SystemState,
				OldValue:      nil,
				NewValue:      workitem.SystemStateResolved,
			},
		}
		// run the test.
		action := ActionStateToMetaState{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges change.Set
		// note: the rule does not use the explicit configuration, but reads from the template.
		afterActionWI, convertChanges, err := action.OnChange(*fxt.WorkItems[0], contextChanges, "", &convertChanges)
		require.NoError(t, err)
		require.Len(t, convertChanges, 2)
		// check metastate validity.
		require.Equal(t, workitem.SystemMetaState, convertChanges[0].AttributeName)
		require.Nil(t, convertChanges[0].OldValue)
		require.Equal(t, "mResolved", convertChanges[0].NewValue)
		require.Equal(t, "mResolved", afterActionWI.(workitem.WorkItem).Fields[workitem.SystemMetaState])
		// check column validity. For the resolved state, two columns are matching,
		// but only the first one (column 2) should be used and available in the WI.
		require.Equal(t, workitem.SystemBoardcolumns, convertChanges[1].AttributeName)
		require.Empty(t, convertChanges[1].OldValue)
		require.Equal(t, convertChanges[1].NewValue.([]interface{})[0], fxt.WorkItemBoards[0].Columns[2].ID.String())
		require.Equal(t, "mResolved", afterActionWI.(workitem.WorkItem).Fields[workitem.SystemMetaState])
		require.Len(t, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemBoardcolumns].([]interface{}), 1)
		require.Equal(t, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemBoardcolumns].([]interface{})[0], fxt.WorkItemBoards[0].Columns[2].ID.String())
	})

	s.T().Run("updating the columns for an existing work item", func(t *testing.T) {
		fxt := fxtFn(t)
		// set the column to the "in progress" column and create a changeset.
		contextChanges := change.Set{
			{
				AttributeName: workitem.SystemBoardcolumns,
				OldValue:      fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns],
				NewValue:      []interface{}{fxt.WorkItemBoards[0].Columns[1].ID.String()},
			},
		}
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{fxt.WorkItemBoards[0].Columns[1].ID.String()}
		// run the test.
		action := ActionStateToMetaState{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges change.Set
		// note: the rule does not use the explicit configuration, but reads from the template.
		afterActionWI, convertChanges, err := action.OnChange(*fxt.WorkItems[0], contextChanges, "", &convertChanges)
		require.NoError(t, err)
		require.Len(t, convertChanges, 2)
		// check metastate validity.
		require.Equal(t, workitem.SystemMetaState, convertChanges[0].AttributeName)
		require.Equal(t, "mNew", convertChanges[0].OldValue)
		require.Equal(t, "mInprogress", convertChanges[0].NewValue)
		require.Equal(t, "mInprogress", afterActionWI.(workitem.WorkItem).Fields[workitem.SystemMetaState])
		// check state validity.
		require.Equal(t, workitem.SystemState, convertChanges[1].AttributeName)
		require.Equal(t, workitem.SystemStateNew, convertChanges[1].OldValue)
		require.Equal(t, workitem.SystemStateInProgress, convertChanges[1].NewValue)
		require.Equal(t, workitem.SystemStateInProgress, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemState])
	})

	s.T().Run("updating the state and columns for an existing work item", func(t *testing.T) {
		fxt := fxtFn(t)
		contextChanges := change.Set{
			{
				AttributeName: workitem.SystemState,
				OldValue:      fxt.WorkItems[0].Fields[workitem.SystemState],
				NewValue:      workitem.SystemStateResolved,
			},
			{
				AttributeName: workitem.SystemBoardcolumns,
				OldValue:      fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns],
				NewValue:      []interface{}{fxt.WorkItemBoards[0].Columns[2].ID.String()},
			},
		}
		// set the state
		fxt.WorkItems[0].Fields[workitem.SystemState] = workitem.SystemStateResolved
		// set the column to the "resolved" column and create a changeset.
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{fxt.WorkItemBoards[0].Columns[2].ID.String()}
		// run the test.
		action := ActionStateToMetaState{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges change.Set
		// note: the rule does not use the explicit configuration, but reads from the template.
		afterActionWI, convertChanges, err := action.OnChange(*fxt.WorkItems[0], contextChanges, "", &convertChanges)
		require.NoError(t, err)
		require.Len(t, convertChanges, 2)
		// check metastate validity.
		require.Equal(t, workitem.SystemMetaState, convertChanges[0].AttributeName)
		require.Equal(t, "mNew", convertChanges[0].OldValue)
		require.Equal(t, "mResolved", convertChanges[0].NewValue)
		require.Equal(t, "mResolved", afterActionWI.(workitem.WorkItem).Fields[workitem.SystemMetaState])
		// check column validity. For the resolved state, two columns are matching,
		// but only the first one (column 2) should be used and available in the WI.
		require.Equal(t, workitem.SystemBoardcolumns, convertChanges[1].AttributeName)
		require.Equal(t, []interface{}{fxt.WorkItemBoards[0].Columns[2].ID.String()}, convertChanges[1].OldValue)
		require.Equal(t, convertChanges[1].NewValue.([]interface{})[0], fxt.WorkItemBoards[0].Columns[2].ID.String())
		require.Equal(t, "mResolved", afterActionWI.(workitem.WorkItem).Fields[workitem.SystemMetaState])
		require.Len(t, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemBoardcolumns].([]interface{}), 1)
		require.Equal(t, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemBoardcolumns].([]interface{})[0], fxt.WorkItemBoards[0].Columns[2].ID.String())
		// check state validity.
		require.Equal(t, workitem.SystemStateResolved, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemState])
	})

	s.T().Run("updating the columns for a vanilla work item", func(t *testing.T) {
		fxt := fxtFn(t)
		// set the column to the "in progress" column and create a changeset.
		contextChanges := change.Set{
			{
				AttributeName: workitem.SystemBoardcolumns,
				OldValue:      nil,
				NewValue:      []interface{}{fxt.WorkItemBoards[0].Columns[1].ID.String()},
			},
		}
		delete(fxt.WorkItems[0].Fields, workitem.SystemMetaState)
		fxt.WorkItems[0].Fields[workitem.SystemBoardcolumns] = []interface{}{fxt.WorkItemBoards[0].Columns[1].ID.String()}
		// run the test.
		action := ActionStateToMetaState{
			Db:     s.GormDB,
			Ctx:    s.Ctx,
			UserID: &fxt.Identities[0].ID,
		}
		var convertChanges change.Set
		// note: the rule does not use the explicit configuration, but reads from the template.
		afterActionWI, convertChanges, err := action.OnChange(*fxt.WorkItems[0], contextChanges, "", &convertChanges)
		require.NoError(t, err)
		require.Len(t, convertChanges, 2)
		// check metastate validity.
		require.Equal(t, workitem.SystemMetaState, convertChanges[0].AttributeName)
		require.Nil(t, convertChanges[0].OldValue)
		require.Equal(t, "mInprogress", convertChanges[0].NewValue)
		require.Equal(t, "mInprogress", afterActionWI.(workitem.WorkItem).Fields[workitem.SystemMetaState])
		// check state validity.
		require.Equal(t, workitem.SystemState, convertChanges[1].AttributeName)
		require.Equal(t, workitem.SystemStateNew, convertChanges[1].OldValue)
		require.Equal(t, workitem.SystemStateInProgress, convertChanges[1].NewValue)
		require.Equal(t, workitem.SystemStateInProgress, afterActionWI.(workitem.WorkItem).Fields[workitem.SystemState])
	})

}
