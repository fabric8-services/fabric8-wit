package iteration_test

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
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

	clean func()
}

func TestRunIterationRepository(t *testing.T) {
	suite.Run(t, &TestIterationRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *TestIterationRepository) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *TestIterationRepository) TearDownTest() {
	s.clean()
}

func (s *TestIterationRepository) TestCreateIteration() {
	userActive := false
	t := s.T()
	resource.Require(t, resource.Database)
	repo := iteration.NewIterationRepository(s.DB)
	start := time.Now()
	end := start.Add(time.Hour * (24 * 8 * 3))
	name := "Sprint #24"

	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))

	i := iteration.Iteration{
		Name:       name,
		SpaceID:    fxt.Spaces[0].ID,
		StartAt:    &start,
		EndAt:      &end,
		UserActive: &userActive,
	}

	repo.Create(context.Background(), &i)
	if i.ID == uuid.Nil {
		t.Errorf("Iteration was not created, ID nil")
	}

	if i.CreatedAt.After(time.Now()) {
		t.Errorf("Iteration was not created, CreatedAt after Now()?")
	}
	assert.Equal(t, start, *i.StartAt)
	assert.Equal(t, end, *i.EndAt)
	assert.Equal(t, name, i.Name)
}

func (s *TestIterationRepository) TestCreateChildIteration() {
	t := s.T()
	resource.Require(t, resource.Database)

	userActive := false
	repo := iteration.NewIterationRepository(s.DB)

	start := time.Now()
	end := start.Add(time.Hour * (24 * 8 * 3))
	name := "Sprint #24"
	name2 := "Sprint #24.1"

	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))

	i := iteration.Iteration{
		Name:       name,
		SpaceID:    fxt.Spaces[0].ID,
		StartAt:    &start,
		EndAt:      &end,
		UserActive: &userActive,
	}
	repo.Create(context.Background(), &i)

	parentPath := append(i.Path, i.ID)
	require.NotNil(t, parentPath)
	i2 := iteration.Iteration{
		Name:       name2,
		SpaceID:    fxt.Spaces[0].ID,
		StartAt:    &start,
		EndAt:      &end,
		Path:       parentPath,
		UserActive: &userActive,
	}
	repo.Create(context.Background(), &i2)

	i2L, err := repo.Load(context.Background(), i2.ID)
	require.Nil(t, err)
	assert.NotEmpty(t, i2.Path)
	i2.Path.Convert()
	expectedPath := i2.Path.Convert()
	require.NotNil(t, i2L)
	assert.Equal(t, expectedPath, i2L.Path.Convert())
}

func (s *TestIterationRepository) TestRootIteration() {
	t := s.T()
	resource.Require(t, resource.Database)

	repo := iteration.NewIterationRepository(s.DB)

	userActive := false
	start := time.Now()
	end := start.Add(time.Hour * (24 * 8 * 3))
	name := "Sprint #24"
	name2 := "Sprint #24.1"

	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))

	i := iteration.Iteration{
		Name:       name,
		SpaceID:    fxt.Spaces[0].ID,
		StartAt:    &start,
		EndAt:      &end,
		UserActive: &userActive,
	}
	repo.Create(context.Background(), &i)

	parentPath := append(i.Path, i.ID)
	require.NotNil(t, parentPath)
	i2 := iteration.Iteration{
		Name:       name2,
		SpaceID:    fxt.Spaces[0].ID,
		StartAt:    &start,
		EndAt:      &end,
		Path:       parentPath,
		UserActive: &userActive,
	}
	repo.Create(context.Background(), &i2)

	res, err := repo.Root(context.Background(), fxt.Spaces[0].ID)
	require.Nil(t, err)
	assert.Equal(t, i.Name, res.Name)
	assert.Equal(t, i.ID, res.ID)
	expectedPath := i.Path.Convert()
	require.NotNil(t, res)
	assert.Equal(t, expectedPath, res.Path.Convert())
}

func (s *TestIterationRepository) TestListIterationBySpace() {
	t := s.T()
	resource.Require(t, resource.Database)

	userActive := false
	repo := iteration.NewIterationRepository(s.DB)

	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(2))

	for i := 0; i < 3; i++ {
		start := time.Now()
		end := start.Add(time.Hour * (24 * 8 * 3))
		name := "Sprint #2" + strconv.Itoa(i)

		i := iteration.Iteration{
			Name:       name,
			SpaceID:    fxt.Spaces[0].ID,
			StartAt:    &start,
			EndAt:      &end,
			UserActive: &userActive,
		}
		e := repo.Create(context.Background(), &i)
		require.Nil(t, e)
	}
	// add iteration to the second space from our fixture
	e := repo.Create(context.Background(), &iteration.Iteration{
		Name:       "Other Spring #2",
		SpaceID:    fxt.Spaces[1].ID,
		UserActive: &userActive,
	})
	require.Nil(t, e)

	its, err := repo.List(context.Background(), fxt.Spaces[0].ID)
	assert.Nil(t, err)
	assert.Len(t, its, 3)
}

func (s *TestIterationRepository) TestUpdateIteration() {
	t := s.T()
	resource.Require(t, resource.Database)
	repo := iteration.NewIterationRepository(s.DB)

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
	require.Nil(t, err)
	assert.Equal(t, updatedIteration.Name, updatedName)
	assert.Equal(t, *updatedIteration.Description, desc)

	changedStart := start.Add(time.Hour)
	i.StartAt = &changedStart
	changedEnd := start.Add(time.Hour * 2)
	i.EndAt = &changedEnd
	// update iteration with new values of StartAt, EndAt
	updatedIteration, err = repo.Save(context.Background(), i)
	require.Nil(t, err)
	assert.Equal(t, changedStart, *updatedIteration.StartAt)
	assert.Equal(t, changedEnd, *updatedIteration.EndAt)
}

func (s *TestIterationRepository) TestCreateIterationSameNameFailsWithinSpace() {
	t := s.T()
	resource.Require(t, resource.Database)
	userActive := false
	repo := iteration.NewIterationRepository(s.DB)
	name := "Iteration name test"

	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Spaces(2),
		tf.Iterations(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.Iterations[idx].Name = name
			return nil
		}),
	)

	i := *fxt.Iterations[0]
	space1 := *fxt.Spaces[0]
	space2 := *fxt.Spaces[1]

	// another iteration with same name within same sapce, should fail
	i2 := iteration.Iteration{
		Name:       name,
		SpaceID:    space1.ID,
		UserActive: &userActive,
	}
	err := repo.Create(context.Background(), &i)
	require.NotNil(t, err)
	require.Equal(t, uuid.Nil, i2.ID)
	assert.Equal(t, reflect.TypeOf(errors.DataConflictError{}), reflect.TypeOf(err))

	// create iteration with same name in another space, should pass
	i3 := iteration.Iteration{
		Name:       name,
		SpaceID:    space2.ID,
		UserActive: &userActive,
	}
	err = repo.Create(context.Background(), &i3)
	require.Nil(t, err)
	require.NotEqual(t, uuid.Nil, i3.ID)
}

func (s *TestIterationRepository) TestLoadChildren() {
	t := s.T()
	resource.Require(t, resource.Database)
	repo := iteration.NewIterationRepository(s.DB)

	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Iterations(3, func(fxt *tf.TestFixture, idx int) error {
			i := fxt.Iterations[idx]
			switch idx {
			case 0:
				i.Name = "Top level iteration"
			case 1:
				i.Name = "Level 1 iteration"
				i.Path = append(fxt.Iterations[idx-1].Path, fxt.Iterations[idx-1].ID)
			case 2:
				i.Name = "Level 2 iteration"
				i.Path = append(fxt.Iterations[idx-1].Path, fxt.Iterations[idx-1].ID)
			}
			return nil
		}),
	)
	i1 := *fxt.Iterations[0]
	i2 := *fxt.Iterations[1]
	i3 := *fxt.Iterations[2]

	// fetch all children of top level iteraiton
	childIterations1, err := repo.LoadChildren(context.Background(), i1.ID)
	require.Nil(t, err)
	require.Equal(t, 2, len(childIterations1))
	expectedChildIDs1 := []uuid.UUID{i2.ID, i3.ID}
	var actualChildIDs1 []uuid.UUID
	for _, child := range childIterations1 {
		actualChildIDs1 = append(actualChildIDs1, child.ID)
	}
	assert.Equal(t, expectedChildIDs1, actualChildIDs1)

	// fetch all children of level 1 iteraiton
	childIterations2, err := repo.LoadChildren(context.Background(), i2.ID)
	require.Nil(t, err)
	require.Equal(t, 1, len(childIterations2))
	expectedChildIDs2 := []uuid.UUID{i3.ID}
	var actualChildIDs2 []uuid.UUID
	for _, child := range childIterations2 {
		actualChildIDs2 = append(actualChildIDs2, child.ID)
	}
	assert.Equal(t, expectedChildIDs2, actualChildIDs2)

	// fetch all children of level 2 iteraiton
	childIterations3, err := repo.LoadChildren(context.Background(), i3.ID)
	require.Nil(t, err)
	require.Equal(t, 0, len(childIterations3))

	// try to fetch children of non-exisitng parent
	fakeParentId := uuid.NewV4()
	_, err = repo.LoadChildren(context.Background(), fakeParentId)
	require.NotNil(t, err)
	assert.Equal(t, reflect.TypeOf(errors.NotFoundError{}), reflect.TypeOf(err))
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
