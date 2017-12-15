package iteration_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestIterationRepository struct {
	gormtestsupport.DBTestSuite
}

func TestRunIterationRepository(t *testing.T) {
	suite.Run(t, &TestIterationRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *TestIterationRepository) TestCreateIteration() {
	t := s.T()
	resource.Require(t, resource.Database)
	repo := iteration.NewIterationRepository(s.DB)

	t.Run("success - create iteration", func(t *testing.T) {
		start := time.Now()
		end := start.Add(time.Hour * (24 * 8 * 3))
		name := "Sprint #24"
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))

		i := iteration.Iteration{
			Name:    name,
			SpaceID: fxt.Spaces[0].ID,
			StartAt: &start,
			EndAt:   &end,
		}
		// when
		repo.Create(context.Background(), &i)
		// then
		if i.ID == uuid.Nil {
			t.Errorf("Iteration was not created, ID nil")
		}
		if i.CreatedAt.After(time.Now()) {
			t.Errorf("Iteration was not created, CreatedAt after Now()?")
		}
		assert.Equal(t, start, *i.StartAt)
		assert.Equal(t, end, *i.EndAt)
		assert.Equal(t, name, i.Name)
	})

	t.Run("success - create child", func(t *testing.T) {
		start := time.Now()
		end := start.Add(time.Hour * (24 * 8 * 3))
		name := "Sprint #24"
		name2 := "Sprint #24.1"
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))

		i := iteration.Iteration{
			Name:    name,
			SpaceID: fxt.Spaces[0].ID,
			StartAt: &start,
			EndAt:   &end,
		}
		// when
		repo.Create(context.Background(), &i)
		parentPath := append(i.Path, i.ID)
		require.NotNil(t, parentPath)
		i2 := iteration.Iteration{
			Name:    name2,
			SpaceID: fxt.Spaces[0].ID,
			StartAt: &start,
			EndAt:   &end,
			Path:    parentPath,
		}
		repo.Create(context.Background(), &i2)
		// then
		i2L, err := repo.Load(context.Background(), i2.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, i2.Path)
		i2.Path.Convert()
		expectedPath := i2.Path.Convert()
		require.NotNil(t, i2L)
		assert.Equal(t, expectedPath, i2L.Path.Convert())
	})

	t.Run("fail - same iteration name within a space", func(t *testing.T) {
		name := "Iteration name test"
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.Iterations(1, tf.SetIterationNames(name)),
		)

		i := *fxt.Iterations[0]
		// another iteration with same name within same sapce, should fail
		i2 := i
		i2.ID = uuid.Nil
		// when
		err := repo.Create(context.Background(), &i2)
		// then
		require.Error(t, err)
		assert.Equal(t, reflect.TypeOf(errors.DataConflictError{}), reflect.TypeOf(err))
	})

	t.Run("pass - same iteration name across different space", func(t *testing.T) {
		name := "Iteration name test"
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.Spaces(2),
			tf.Iterations(1, tf.SetIterationNames(name)),
		)
		space2 := *fxt.Spaces[1]
		// create iteration with same name in another space, should pass
		i2 := iteration.Iteration{
			Name:    name,
			SpaceID: space2.ID,
		}
		// when
		err := repo.Create(context.Background(), &i2)
		// then
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, i2.ID)
	})
}

func (s *TestIterationRepository) TestLoad() {
	t := s.T()
	resource.Require(t, resource.Database)

	repo := iteration.NewIterationRepository(s.DB)
	t.Run("load root iteration", func(t *testing.T) {
		start := time.Now()
		end := start.Add(time.Hour * (24 * 8 * 3))
		name := "Sprint #24"
		name2 := "Sprint #24.1"
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))

		i := iteration.Iteration{
			Name:    name,
			SpaceID: fxt.Spaces[0].ID,
			StartAt: &start,
			EndAt:   &end,
		}
		// when
		repo.Create(context.Background(), &i)

		parentPath := append(i.Path, i.ID)
		require.NotNil(t, parentPath)
		i2 := iteration.Iteration{
			Name:    name2,
			SpaceID: fxt.Spaces[0].ID,
			StartAt: &start,
			EndAt:   &end,
			Path:    parentPath,
		}
		repo.Create(context.Background(), &i2)
		// then
		res, err := repo.Root(context.Background(), fxt.Spaces[0].ID)
		require.NoError(t, err)
		assert.Equal(t, i.Name, res.Name)
		assert.Equal(t, i.ID, res.ID)
		expectedPath := i.Path.Convert()
		require.NotNil(t, res)
		assert.Equal(t, expectedPath, res.Path.Convert())
	})

	t.Run("list by space", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.Spaces(2),
			tf.Iterations(4, func(fxt *tf.TestFixture, idx int) error {
				if idx == 3 {
					itr := fxt.Iterations[idx]
					itr.SpaceID = fxt.Spaces[1].ID
				}
				return nil
			}))
		// when
		its, err := repo.List(context.Background(), fxt.Spaces[0].ID)
		// then
		require.NoError(t, err)
		assert.Len(t, its, 3)
		var mustHaveIDs = make(map[uuid.UUID]struct{}, 3)
		mustHaveIDs = map[uuid.UUID]struct{}{
			fxt.Iterations[0].ID: {},
			fxt.Iterations[1].ID: {},
			fxt.Iterations[2].ID: {},
		}
		for _, itr := range its {
			delete(mustHaveIDs, itr.ID)
		}
		require.Empty(t, mustHaveIDs)

		// when
		its, err = repo.List(context.Background(), fxt.Spaces[1].ID)
		// then
		require.NoError(t, err)
		assert.Len(t, its, 1)
		mustHaveIDs = make(map[uuid.UUID]struct{}, 1)
		mustHaveIDs = map[uuid.UUID]struct{}{
			fxt.Iterations[3].ID: {},
		}
		for _, itr := range its {
			delete(mustHaveIDs, itr.ID)
		}
		require.Empty(t, mustHaveIDs)
	})

	t.Run("success - load children for iteration", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.Iterations(3, func(fxt *tf.TestFixture, idx int) error {
				i := fxt.Iterations[idx]
				switch idx {
				case 0:
					i.Name = "Top level iteration"
				case 1:
					i.Name = "Level 1 iteration"
					i.MakeChildOf(*fxt.Iterations[idx-1])
				case 2:
					i.Name = "Level 2 iteration"
					i.MakeChildOf(*fxt.Iterations[idx-1])
				}
				return nil
			}),
		)
		i1 := *fxt.Iterations[0]
		i2 := *fxt.Iterations[1]
		i3 := *fxt.Iterations[2]

		// when
		// fetch all children of top level iteration
		childIterations1, err := repo.LoadChildren(context.Background(), i1.ID)
		// then
		require.NoError(t, err)
		require.Equal(t, 2, len(childIterations1))
		expectedChildIDs1 := []uuid.UUID{i2.ID, i3.ID}
		var actualChildIDs1 []uuid.UUID
		for _, child := range childIterations1 {
			actualChildIDs1 = append(actualChildIDs1, child.ID)
		}
		assert.Equal(t, expectedChildIDs1, actualChildIDs1)

		// when
		// fetch all children of level 1 iteration
		childIterations2, err := repo.LoadChildren(context.Background(), i2.ID)
		// then
		require.NoError(t, err)
		require.Equal(t, 1, len(childIterations2))
		expectedChildIDs2 := []uuid.UUID{i3.ID}
		var actualChildIDs2 []uuid.UUID
		for _, child := range childIterations2 {
			actualChildIDs2 = append(actualChildIDs2, child.ID)
		}
		assert.Equal(t, expectedChildIDs2, actualChildIDs2)

		// when
		// fetch all children of level 2 iteration
		childIterations3, err := repo.LoadChildren(context.Background(), i3.ID)
		// then
		require.NoError(t, err)
		require.Equal(t, 0, len(childIterations3))
	})

	t.Run("fail - load children for non-existing iteration", func(t *testing.T) {
		// try to fetch children of non-existing parent
		fakeParentId := uuid.NewV4()
		// when
		_, err := repo.LoadChildren(context.Background(), fakeParentId)
		// then
		require.Error(t, err)
		assert.Equal(t, reflect.TypeOf(errors.NotFoundError{}), reflect.TypeOf(err))
	})

}

func (s *TestIterationRepository) TestUpdate() {
	t := s.T()
	resource.Require(t, resource.Database)
	repo := iteration.NewIterationRepository(s.DB)

	t.Run("update iteration", func(t *testing.T) {
		start := time.Now()
		end := start.Add(time.Hour * (24 * 8 * 3))

		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.Iterations(1,
				tf.UserActive(false),
				func(fxt *tf.TestFixture, idx int) error {
					i := fxt.Iterations[idx]
					i.Name = "Sprint #24"
					i.StartAt = &start
					i.EndAt = &end
					return nil
				},
			),
		)

		i := *fxt.Iterations[0]

		desc := "Updated item"
		i.Description = &desc
		updatedName := "Sprint 25"
		i.Name = updatedName
		// update iteration with new values of Name and Desc
		updatedIteration, err := repo.Save(context.Background(), i)
		require.NoError(t, err)
		assert.Equal(t, updatedIteration.Name, updatedName)
		assert.Equal(t, *updatedIteration.Description, desc)

		changedStart := start.Add(time.Hour)
		i.StartAt = &changedStart
		changedEnd := start.Add(time.Hour * 2)
		i.EndAt = &changedEnd
		// update iteration with new values of StartAt, EndAt
		updatedIteration, err = repo.Save(context.Background(), i)
		require.NoError(t, err)
		assert.Equal(t, changedStart, *updatedIteration.StartAt)
		assert.Equal(t, changedEnd, *updatedIteration.EndAt)
	})
}

func (s *TestIterationRepository) TestExistsIteration() {
	t := s.T()
	resource.Require(t, resource.Database)
	repo := iteration.NewIterationRepository(s.DB)
	t.Run("iteration exists", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.Iterations(1))
		require.Nil(t, repo.CheckExists(context.Background(), fxt.Iterations[0].ID.String()))
	})
	t.Run("iteration doesn't exist", func(t *testing.T) {
		err := repo.CheckExists(context.Background(), uuid.NewV4().String())
		require.IsType(t, errors.NotFoundError{}, err)
	})
}

func (s *TestIterationRepository) TestIsActive() {
	t := s.T()
	t.Run("user active is true", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.Iterations(1, tf.UserActive(true)))
		require.True(t, fxt.Iterations[0].IsActive())
	})
	t.Run("start date is nil", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.Iterations(1, tf.UserActive(false)))
		require.False(t, fxt.Iterations[0].IsActive())
	})
	t.Run("end date is nil and current date is after start date", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.Iterations(1,
				tf.UserActive(false),
				func(fxt *tf.TestFixture, idx int) error {
					start := time.Now().Add(-1 * time.Hour) // start date was one hour ago
					fxt.Iterations[idx].StartAt = &start
					return nil
				},
			),
		)
		require.True(t, fxt.Iterations[0].IsActive())
	})
	t.Run("end date is nil and current date is before start date", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.Iterations(1,
				tf.UserActive(false),
				func(fxt *tf.TestFixture, idx int) error {
					start := time.Now().Add(1 * time.Hour) // start date is one hour ahead
					fxt.Iterations[idx].StartAt = &start
					return nil
				},
			),
		)
		require.False(t, fxt.Iterations[0].IsActive())
	})
}

func (s *TestIterationRepository) TestIsRoot() {
	t := s.T()
	resource.Require(t, resource.Database)
	t.Run("check IsRoot on root & other iterations", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.Iterations(2, tf.PlaceIterationUnderRootIteration()))
		spaceID := fxt.Spaces[0].ID
		require.True(t, fxt.Iterations[0].IsRoot(spaceID))
		require.False(t, fxt.Iterations[1].IsRoot(spaceID))
		fakeSpaceID := uuid.NewV4()
		require.False(t, fxt.Iterations[0].IsRoot(fakeSpaceID))
		require.False(t, fxt.Iterations[1].IsRoot(fakeSpaceID))
	})
}

func (s *TestIterationRepository) TestParent() {
	t := s.T()
	resource.Require(t, resource.Database)
	t.Run("check Parent() for root & intermediate & leaf iterations", func(t *testing.T) {
		// Fixture is now creating following hierarchy of iterations
		// root Iteration
		// |___________Iteration 1
		// |                |___________Iteration 2
		// |                                |___________Iteration 3
		// |___________Iteration 4
		//                     |___________Iteration 5
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.Iterations(6,
			func(fxt *tf.TestFixture, idx int) error {
				i := fxt.Iterations[idx]
				switch idx {
				case 1:
					i.MakeChildOf(*fxt.Iterations[0])
				case 2:
					i.MakeChildOf(*fxt.Iterations[1])
				case 3:
					i.MakeChildOf(*fxt.Iterations[2])
				case 4:
					i.MakeChildOf(*fxt.Iterations[0])
				case 5:
					i.MakeChildOf(*fxt.Iterations[4])
				}
				return nil
			}))
		rootID := fxt.Iterations[0].ID
		iteration1ID := fxt.Iterations[1].ID
		iteration2ID := fxt.Iterations[2].ID
		iteration4ID := fxt.Iterations[4].ID

		require.Equal(t, uuid.Nil, fxt.Iterations[0].Parent())
		require.Equal(t, rootID, fxt.Iterations[1].Parent())
		require.Equal(t, iteration1ID, fxt.Iterations[2].Parent())
		require.Equal(t, iteration2ID, fxt.Iterations[3].Parent())
		require.Equal(t, rootID, fxt.Iterations[4].Parent())
		require.Equal(t, iteration4ID, fxt.Iterations[5].Parent())
	})
}
