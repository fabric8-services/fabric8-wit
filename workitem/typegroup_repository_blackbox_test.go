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

type workItemTypeGroupRepoTest struct {
	gormtestsupport.DBTestSuite
	repo workitem.WorkItemTypeGroupRepository
}

func TestWorkItemTypeGroupRepository(t *testing.T) {
	suite.Run(t, &workItemTypeGroupRepoTest{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *workItemTypeGroupRepoTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = workitem.NewWorkItemTypeGroupRepository(s.DB)
}

func (s *workItemTypeGroupRepoTest) TestExists() {
	s.T().Run("group exists", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypeGroups(1))
		// when
		err := s.repo.CheckExists(s.Ctx, fxt.WorkItemTypeGroups[0].ID)
		// then
		require.NoError(s.T(), err)
	})

	s.T().Run("group doesn't exist", func(t *testing.T) {
		// given
		nonExistingWorkItemTypeGroupID := uuid.NewV4()
		// when
		err := s.repo.CheckExists(s.Ctx, nonExistingWorkItemTypeGroupID)
		// then
		require.IsType(t, errors.NotFoundError{}, err)
	})
}

func compareTypeGroups(t *testing.T, expected, actual workitem.WorkItemTypeGroup) {
	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.SpaceTemplateID, actual.SpaceTemplateID)
	assert.Equal(t, expected.Bucket, actual.Bucket)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Icon, actual.Icon)
	assert.Equal(t, expected.Position, actual.Position)
	assert.Equal(t, expected.TypeList, actual.TypeList)
}

func (s *workItemTypeGroupRepoTest) TestCreate() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemTypes(3))
	ID := uuid.NewV4()
	expected := workitem.WorkItemTypeGroup{
		ID:              ID,
		SpaceTemplateID: fxt.SpaceTemplates[0].ID,
		Bucket:          workitem.BucketIteration,
		Name:            "work item type group " + ID.String(),
		Icon:            "a world on the back of a turtle",
		Position:        42,
		TypeList: []uuid.UUID{
			fxt.WorkItemTypes[0].ID,
			fxt.WorkItemTypes[1].ID,
			fxt.WorkItemTypes[2].ID,
		},
	}

	s.T().Run("ok", func(t *testing.T) {
		actual, err := s.repo.Create(s.Ctx, expected)
		require.NoError(t, err)
		compareTypeGroups(t, expected, *actual)
		t.Run("load same work item and check it is the same", func(t *testing.T) {
			actual, err := s.repo.Load(s.Ctx, ID)
			require.NoError(t, err)
			compareTypeGroups(t, expected, *actual)
		})
	})
	s.T().Run("invalid", func(t *testing.T) {
		t.Run("unknown space template", func(t *testing.T) {
			g := expected
			g.SpaceTemplateID = uuid.NewV4()
			_, err := s.repo.Create(s.Ctx, g)
			require.Error(t, err)
		})
		t.Run("unknown work item type", func(t *testing.T) {
			g := expected
			g.TypeList = []uuid.UUID{uuid.NewV4()}
			_, err := s.repo.Create(s.Ctx, g)
			require.Error(t, err)
		})
		t.Run("empty type list", func(t *testing.T) {
			g := expected
			g.TypeList = []uuid.UUID{}
			_, err := s.repo.Create(s.Ctx, g)
			require.Error(t, err)
		})
	})
}

func (s *workItemTypeGroupRepoTest) TestLoad() {
	s.T().Run("group exists", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypeGroups(1))
		// when
		actual, err := s.repo.Load(s.Ctx, fxt.WorkItemTypeGroups[0].ID)
		require.NoError(t, err)
		compareTypeGroups(t, *fxt.WorkItemTypeGroups[0], *actual)
	})
	s.T().Run("group doesn't exist", func(t *testing.T) {
		// when
		_, err := s.repo.Load(s.Ctx, uuid.NewV4())
		// then
		require.Error(t, err)
	})
}

func (s *workItemTypeGroupRepoTest) TestList() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypeGroups(3))
		// when
		actual, err := s.repo.List(s.Ctx, fxt.SpaceTemplates[0].ID)
		// then
		require.NoError(t, err)
		require.Len(t, actual, len(fxt.WorkItemTypeGroups))
		for idx := range fxt.WorkItemTypeGroups {
			compareTypeGroups(t, *fxt.WorkItemTypeGroups[idx], *actual[idx])
		}
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

func TestWorkItemTypeGroup_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// given
	a := workitem.WorkItemTypeGroup{
		ID:              uuid.NewV4(),
		SpaceTemplateID: uuid.NewV4(),
		Name:            "foo",
		Bucket:          workitem.BucketRequirement,
		Position:        42,
		Icon:            "bar",
		TypeList: []uuid.UUID{
			uuid.NewV4(),
			uuid.NewV4(),
			uuid.NewV4(),
		},
	}
	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		b := a
		assert.True(t, a.Equal(b))
	})
	t.Run("types", func(t *testing.T) {
		t.Parallel()
		b := convert.DummyEqualer{}
		assert.False(t, a.Equal(b))
	})
	t.Run("Lifecycle", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
		assert.False(t, a.Equal(b))
	})
	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Name = "bar"
		assert.False(t, a.Equal(b))
	})
	t.Run("SpaceTemplateID", func(t *testing.T) {
		t.Parallel()
		b := a
		b.SpaceTemplateID = uuid.NewV4()
		assert.False(t, a.Equal(b))
	})
	t.Run("Bucket", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Bucket = workitem.BucketIteration
		assert.False(t, a.Equal(b))
	})
	t.Run("Icon", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Icon = "blabla"
		assert.False(t, a.Equal(b))
	})
	t.Run("TypeList", func(t *testing.T) {
		t.Parallel()
		b := a
		// different IDs
		b.TypeList = []uuid.UUID{uuid.NewV4(), uuid.NewV4(), uuid.NewV4()}
		assert.False(t, a.Equal(b))
		// different length
		b.TypeList = []uuid.UUID{uuid.NewV4(), uuid.NewV4()}
		assert.False(t, a.Equal(b))
	})
}
