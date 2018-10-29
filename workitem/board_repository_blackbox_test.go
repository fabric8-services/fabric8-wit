package workitem_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type workItemBoardRepoTest struct {
	gormtestsupport.DBTestSuite
	repo workitem.BoardRepository
}

func TestWorkItemBoardRepository(t *testing.T) {
	suite.Run(t, &workItemBoardRepoTest{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *workItemBoardRepoTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = workitem.NewBoardRepository(s.DB)
}

func (s *workItemBoardRepoTest) TestExists() {
	s.T().Run("board exists", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemBoards(1))
		// when
		err := s.repo.CheckExists(s.Ctx, fxt.WorkItemBoards[0].ID)
		// then
		require.NoError(s.T(), err)
	})

	s.T().Run("board doesn't exist", func(t *testing.T) {
		// given
		nonExistingWorkItemBoardID := uuid.NewV4()
		// when
		err := s.repo.CheckExists(s.Ctx, nonExistingWorkItemBoardID)
		// then
		require.IsType(t, errors.NotFoundError{}, err)
	})
}

func (s *workItemBoardRepoTest) TestCreate() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment())
	ID := uuid.NewV4()
	expected := workitem.Board{
		ID:              ID,
		SpaceTemplateID: fxt.SpaceTemplates[0].ID,
		Name:            "Some Board Name",
		Description:     "Some Board Description ",
		ContextType:     "TypeLevelContext",
		Context:         uuid.NewV4().String(),
		Columns: []workitem.BoardColumn{
			{
				ID:                uuid.NewV4(),
				Name:              "New",
				Order:             0,
				TransRuleKey:      "updateStateFromColumnMove",
				TransRuleArgument: "{ 'metastate': 'mNew' }",
				BoardID:           ID,
			},
			{
				ID:                uuid.NewV4(),
				Name:              "Done",
				Order:             1,
				TransRuleKey:      "updateStateFromColumnMove",
				TransRuleArgument: "{ 'metastate': 'mDone' }",
				BoardID:           ID,
			},
		},
	}

	s.T().Run("ok", func(t *testing.T) {
		actual, err := s.repo.Create(s.Ctx, expected)
		require.NoError(t, err)
		require.False(t, expected.Equal(*actual))
		require.True(t, expected.EqualValue(*actual))
		require.True(t, expected.Columns[0].EqualValue(actual.Columns[0]))
		require.True(t, expected.Columns[1].EqualValue(actual.Columns[1]))
		t.Run("load same work item board and check it is the same", func(t *testing.T) {
			actual, err := s.repo.Load(s.Ctx, ID)
			require.NoError(t, err)
			require.True(t, expected.EqualValue(*actual))
			require.True(t, expected.Columns[0].EqualValue(actual.Columns[0]))
			require.True(t, expected.Columns[1].EqualValue(actual.Columns[1]))
		})
	})
	s.T().Run("invalid", func(t *testing.T) {
		t.Run("unknown space template", func(t *testing.T) {
			g := expected
			g.ID = uuid.NewV4()
			g.SpaceTemplateID = uuid.NewV4()
			_, err := s.repo.Create(s.Ctx, g)
			require.Contains(t, err.Error(), "work_item_boards_space_template_id_fkey")
		})
		t.Run("two boards with the same name are not allowed within the same space template", func(t *testing.T) {
			g := expected
			g.ID = uuid.NewV4()
			_, err := s.repo.Create(s.Ctx, g)
			require.Error(t, err)
			require.Contains(t, err.Error(), "work_item_board_name_space_template_id_unique")
		})
		t.Run("two columns with the same order in the same board", func(t *testing.T) {
			g := expected
			g.ID = uuid.NewV4()
			g.Name = uuid.NewV4().String()
			g.Columns[0].ID = uuid.NewV4()
			g.Columns[1].ID = uuid.NewV4()
			g.Columns[0].Order = 123
			g.Columns[1].Order = 123
			_, err := s.repo.Create(s.Ctx, g)
			require.Error(t, err)
			require.Contains(t, err.Error(), "work_item_board_id_order_unique")
		})
	})
}

func (s *workItemBoardRepoTest) TestLoad() {
	s.T().Run("board exists", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemBoards(1))
		// when
		actual, err := s.repo.Load(s.Ctx, fxt.WorkItemBoards[0].ID)
		require.NoError(t, err)
		require.True(t, fxt.WorkItemBoards[0].EqualValue(*actual))
	})
	s.T().Run("board doesn't exist", func(t *testing.T) {
		// when
		_, err := s.repo.Load(s.Ctx, uuid.NewV4())
		// then
		require.Error(t, err)
	})
}

func (s *workItemBoardRepoTest) TestList() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemBoards(3))
		// when
		actual, err := s.repo.List(s.Ctx, fxt.SpaceTemplates[0].ID)
		// then
		require.NoError(t, err)
		require.Len(t, actual, len(fxt.WorkItemBoards))
		toBeFound := map[uuid.UUID]struct{}{
			fxt.WorkItemBoards[0].ID: {},
			fxt.WorkItemBoards[1].ID: {},
			fxt.WorkItemBoards[2].ID: {},
		}
		for _, b := range actual {
			_, ok := toBeFound[b.ID]
			assert.True(t, ok, "found unexpected board (%+v)", b.ID)
			delete(toBeFound, b.ID)
		}
		require.Empty(t, toBeFound, "failed to find these boards: %+v", toBeFound)
	})
	s.T().Run("space template not found", func(t *testing.T) {
		// when
		groups, err := s.repo.List(s.Ctx, uuid.NewV4())
		// then
		require.Error(t, err)
		require.IsType(t, errors.NotFoundError{}, errs.Cause(err))
		require.Empty(t, groups)
	})
}

func TestWorkItemBoard_EqualAndEqualValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// given
	ID := uuid.NewV4()
	a := workitem.Board{
		ID:              ID,
		SpaceTemplateID: uuid.NewV4(),
		Name:            "Some Board Name",
		Description:     "Some Board Description ",
		ContextType:     "TypeLevelContext",
		Context:         uuid.NewV4().String(),
		Columns: []workitem.BoardColumn{
			{
				ID:                uuid.NewV4(),
				Name:              "New",
				Order:             0,
				TransRuleKey:      "updateStateFromColumnMove",
				TransRuleArgument: "{ 'metastate': 'mNew' }",
				BoardID:           ID,
			},
			{
				ID:                uuid.NewV4(),
				Name:              "Done",
				Order:             1,
				TransRuleKey:      "updateStateFromColumnMove",
				TransRuleArgument: "{ 'metastate': 'mDone' }",
				BoardID:           ID,
			},
		},
	}
	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		b := a
		require.True(t, a.Equal(b))
		require.True(t, a.EqualValue(b))
	})
	t.Run("types", func(t *testing.T) {
		t.Parallel()
		b := convert.DummyEqualer{}
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})
	t.Run("name", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Name = "bar"
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})
	t.Run("space template ID", func(t *testing.T) {
		t.Parallel()
		b := a
		b.SpaceTemplateID = uuid.NewV4()
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})
	t.Run("columns", func(t *testing.T) {
		t.Parallel()
		t.Run("different IDs", func(t *testing.T) {
			t.Parallel()
			b := a
			b.Columns = []workitem.BoardColumn{
				{
					ID:                uuid.NewV4(),
					Name:              "New",
					Order:             0,
					TransRuleKey:      "updateStateFromColumnMove",
					TransRuleArgument: "{ 'metastate': 'mNew' }",
					BoardID:           ID,
				},
				{
					ID:                uuid.NewV4(),
					Name:              "Done",
					Order:             1,
					TransRuleKey:      "updateStateFromColumnMove",
					TransRuleArgument: "{ 'metastate': 'mDone' }",
					BoardID:           ID,
				},
			}
			require.False(t, a.Equal(b))
			require.False(t, a.EqualValue(b))
		})
		t.Run("different length (shorter)", func(t *testing.T) {
			t.Parallel()
			b := a
			b.Columns = []workitem.BoardColumn{
				{
					ID:                uuid.NewV4(),
					Name:              "New",
					Order:             0,
					TransRuleKey:      "updateStateFromColumnMove",
					TransRuleArgument: "{ 'metastate': 'mNew' }",
					BoardID:           ID,
				},
			}
			require.False(t, a.Equal(b))
			require.False(t, a.EqualValue(b))
		})
		t.Run("different length (longer)", func(t *testing.T) {
			t.Parallel()
			b := a
			b.Columns = append(b.Columns, workitem.BoardColumn{
				ID:                uuid.NewV4(),
				Name:              "New 1",
				Order:             0,
				TransRuleKey:      "updateStateFromColumnMove",
				TransRuleArgument: "{ 'metastate': 'mNew' }",
				BoardID:           ID,
			})
			require.False(t, a.Equal(b))
			require.False(t, a.EqualValue(b))
		})
		t.Run("column lifecycle", func(t *testing.T) {
			t.Parallel()
			// given two identical boards
			col1 := workitem.BoardColumn{Lifecycle: gormsupport.Lifecycle{CreatedAt: time.Now()}}
			a := workitem.Board{Columns: []workitem.BoardColumn{col1}}
			b := workitem.Board{Columns: []workitem.BoardColumn{col1}}
			require.True(t, a.Equal(b))
			require.True(t, a.EqualValue(b))
			// when changing the creation date of a column
			col2 := col1
			col2.Lifecycle.CreatedAt = time.Now().Add(time.Hour)
			b.Columns[0] = col2
			// then expect the board comparison to fail
			require.False(t, a.Equal(b))
			require.True(t, a.EqualValue(b))
		})
	})
}

func TestWorkItemBoardColumn_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// given
	a := workitem.BoardColumn{
		ID:                uuid.NewV4(),
		Name:              "New",
		Order:             0,
		TransRuleKey:      "updateStateFromColumnMove",
		TransRuleArgument: "{ 'metastate': 'mNew' }",
		BoardID:           uuid.NewV4(),
	}
	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		b := a
		require.True(t, a.Equal(b))
	})
	t.Run("types", func(t *testing.T) {
		t.Parallel()
		b := convert.DummyEqualer{}
		require.False(t, a.Equal(b))
	})
	t.Run("name", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Name = "bar"
		require.False(t, a.Equal(b))
	})
	t.Run("id", func(t *testing.T) {
		t.Parallel()
		b := a
		b.ID = uuid.NewV4()
		require.False(t, a.Equal(b))
	})
	t.Run("order", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Order = 1234
		require.False(t, a.Equal(b))
	})
	t.Run("trans rule key", func(t *testing.T) {
		t.Parallel()
		b := a
		b.TransRuleKey = "foo"
		require.False(t, a.Equal(b))
	})
	t.Run("trans rule argument", func(t *testing.T) {
		t.Parallel()
		b := a
		b.TransRuleArgument = "bar"
		require.False(t, a.Equal(b))
	})
	t.Run("board ID", func(t *testing.T) {
		t.Parallel()
		b := a
		b.BoardID = uuid.NewV4()
		require.False(t, a.Equal(b))
	})
}
