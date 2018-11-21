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
	suite.Run(t, &TestIterationRepository{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *TestIterationRepository) TestCreate() {
	t := s.T()
	resource.Require(t, resource.Database)
	repo := iteration.NewIterationRepository(s.DB)

	t.Run("success - create iteration", func(t *testing.T) {
		start := time.Now()
		end := start.Add(time.Hour * (24 * 8 * 3))
		name := "Sprint #24"
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(2))

		i := iteration.Iteration{
			Name:    name,
			SpaceID: fxt.Spaces[0].ID,
			StartAt: &start,
			EndAt:   &end,
		}
		// when
		err := repo.Create(context.Background(), &i)
		// then
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, i.ID, "iteration not created, ID is nil")
		require.False(t, i.CreatedAt.After(time.Now()), "iteration was not created, CreatedAt after Now()?")
		assert.Equal(t, start, *i.StartAt)
		assert.Equal(t, end, *i.EndAt)
		assert.Equal(t, name, i.Name)
		assert.Equal(t, 1, i.Number)
		t.Run("second iteration in space gets sequential number", func(t *testing.T) {
			i := iteration.Iteration{Name: "second iteration", SpaceID: fxt.Spaces[0].ID}
			err := repo.Create(context.Background(), &i)
			require.NoError(t, err)
			assert.Equal(t, 2, i.Number)
		})
		t.Run("first iteration in another space starts numbering at 1", func(t *testing.T) {
			i := iteration.Iteration{Name: "first iteration", SpaceID: fxt.Spaces[1].ID}
			err := repo.Create(context.Background(), &i)
			require.NoError(t, err)
			assert.Equal(t, 1, i.Number)
		})
	})

	t.Run("success - create child", func(t *testing.T) {
		start := time.Now()
		end := start.Add(time.Hour * (24 * 8 * 3))
		name := "Sprint #24"
		name2 := "Sprint #24.1"
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))

		i := iteration.Iteration{
			Name:    name,
			SpaceID: fxt.Spaces[0].ID,
			StartAt: &start,
			EndAt:   &end,
		}
		// when
		err := repo.Create(context.Background(), &i)
		require.NoError(t, err)
		parentPath := append(i.Path, i.ID)
		require.NotNil(t, parentPath)
		i2 := iteration.Iteration{
			Name:    name2,
			SpaceID: fxt.Spaces[0].ID,
			StartAt: &start,
			EndAt:   &end,
			Path:    parentPath,
		}
		err = repo.Create(context.Background(), &i2)
		// then
		require.NoError(t, err)
		i2L, err := repo.Load(context.Background(), i2.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, i2.Path)
		i2.Path.Convert()
		expectedPath := i2.Path.Convert()
		require.NotNil(t, i2L)
		assert.Equal(t, expectedPath, i2L.Path.Convert())
	})

	t.Run("fail - same iteration name within a space", func(t *testing.T) {
		t.Run("root iteration", func(t *testing.T) {
			name := "Iteration name test"
			// given
			fxt := tf.NewTestFixture(t, s.DB,
				tf.Iterations(1, tf.SetIterationNames(name)),
			)

			i := *fxt.Iterations[0]

			// another iteration with same name within same sapce, should fail
			i2 := iteration.Iteration{
				Name:    i.Name,
				SpaceID: i.SpaceID,
			}
			i2.MakeChildOf(i)

			i3 := iteration.Iteration{
				Name:    i.Name,
				SpaceID: i.SpaceID,
			}
			i3.MakeChildOf(*fxt.Iterations[0])

			// when
			err := repo.Create(context.Background(), &i2)

			// then
			require.NoError(t, err)

			// when
			err = repo.Create(context.Background(), &i3)

			// then
			require.Error(t, err)
			assert.Equal(t, reflect.TypeOf(errors.DataConflictError{}), reflect.TypeOf(err))
		})

		t.Run("sub iteration", func(t *testing.T) {
			// givend
			name := "Iteration name test"
			fxt := tf.NewTestFixture(t, s.DB,
				tf.Iterations(2, func(fxt *tf.TestFixture, idx int) error {
					fxt.Iterations[idx].Name = name
					if idx == 1 {
						fxt.Iterations[idx].MakeChildOf(*fxt.Iterations[idx-1])
					}
					return nil
				}),
			)

			// another iteration with same name within same space, should fail

			i3 := iteration.Iteration{
				Name:    fxt.Iterations[1].Name,
				SpaceID: fxt.Iterations[0].SpaceID,
			}
			i3.MakeChildOf(*fxt.Iterations[0])

			// when
			err := repo.Create(context.Background(), &i3)

			// then
			require.Error(t, err)
			assert.Equal(t, reflect.TypeOf(errors.DataConflictError{}), reflect.TypeOf(err))
		})
	})

	t.Run("pass - same iteration name across different space", func(t *testing.T) {
		name := "Iteration name test"
		// given
		fxt := tf.NewTestFixture(t, s.DB,
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
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))

		i := iteration.Iteration{
			Name:    name,
			SpaceID: fxt.Spaces[0].ID,
			StartAt: &start,
			EndAt:   &end,
		}
		// when
		err := repo.Create(context.Background(), &i)
		require.NoError(t, err)
		i2 := iteration.Iteration{
			Name:    name2,
			SpaceID: fxt.Spaces[0].ID,
			StartAt: &start,
			EndAt:   &end,
		}
		i2.MakeChildOf(i)
		err = repo.Create(context.Background(), &i2)
		// then
		require.NoError(t, err)
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
		fxt := tf.NewTestFixture(t, s.DB,
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
		fxt := tf.NewTestFixture(t, s.DB,
			tf.Iterations(3, func(fxt *tf.TestFixture, idx int) error {
				i := fxt.Iterations[idx]
				switch idx {
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
		i2 := *fxt.Iterations[1]
		i3 := *fxt.Iterations[2]

		// when
		// fetch all children of top level iteration
		childIterations1, err := repo.LoadChildren(context.Background(), fxt.Iterations[0].ID)
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

	t.Run("fail - doesn't exists", func(t *testing.T) {
		// Test fixture doesn't create root iteration and root area
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
		rootIteration, err := repo.Root(context.Background(), fxt.Spaces[0].ID)
		assert.EqualError(t, err, errors.NewNotFoundError("root iteration for space", fxt.Spaces[0].ID.String()).Error())
		assert.Nil(t, rootIteration)
	})

}

func (s *TestIterationRepository) TestUpdate() {
	t := s.T()
	resource.Require(t, resource.Database)
	repo := iteration.NewIterationRepository(s.DB)

	t.Run("update iteration", func(t *testing.T) {
		start := time.Now()
		end := start.Add(time.Hour * (24 * 8 * 3))

		fxt := tf.NewTestFixture(t, s.DB,
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
		fxt := tf.NewTestFixture(t, s.DB, tf.Iterations(1))
		require.Nil(t, repo.CheckExists(context.Background(), fxt.Iterations[0].ID))
	})
	t.Run("iteration doesn't exist", func(t *testing.T) {
		err := repo.CheckExists(context.Background(), uuid.NewV4())
		require.IsType(t, errors.NotFoundError{}, err)
	})
}

func (s *TestIterationRepository) TestIsActive() {
	t := s.T()
	t.Run("user active is true", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.Iterations(1, tf.UserActive(true)))
		require.True(t, fxt.Iterations[0].IsActive())
	})
	t.Run("start date is nil", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.Iterations(1, tf.UserActive(false)))
		require.False(t, fxt.Iterations[0].IsActive())
	})
	t.Run("end date is nil and current date is after start date", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB,
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
		fxt := tf.NewTestFixture(t, s.DB,
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
		fxt := tf.NewTestFixture(t, s.DB, tf.Iterations(2, tf.PlaceIterationUnderRootIteration()))
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
		fxt := tf.NewTestFixture(t, s.DB, tf.Iterations(6,
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
		rootID := fxt.Iterations[0].Path.Convert()
		iteration1ID := fxt.Iterations[1].Path.Convert()
		iteration2ID := fxt.Iterations[2].Path.Convert()
		iteration4ID := fxt.Iterations[4].Path.Convert()

		require.Equal(t, uuid.Nil, fxt.Iterations[0].Parent())
		require.Contains(t, fxt.Iterations[1].Path.Convert(), rootID)
		require.Contains(t, fxt.Iterations[2].Path.Convert(), iteration1ID)
		require.Contains(t, fxt.Iterations[3].Path.Convert(), iteration2ID)
		require.Contains(t, fxt.Iterations[4].Path.Convert(), rootID)
		require.Contains(t, fxt.Iterations[5].Path.Convert(), iteration4ID)
	})
}

func (s *TestIterationRepository) TestLoadMultiple() {
	// nothing should be returned by LoadMultiple when we pass empty slice
	s.T().Run("input empty slice", func(t *testing.T) {
		// keep few iterations in DB on purpose
		_ = tf.NewTestFixture(t, s.DB, tf.Iterations(2))
		emptyList := []uuid.UUID{}
		// when
		listLoadedIterations, err := iteration.NewIterationRepository(s.DB).LoadMultiple(context.Background(), emptyList)
		// then
		require.NoError(t, err)
		require.Empty(t, listLoadedIterations)
	})
}
