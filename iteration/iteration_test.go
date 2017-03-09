package iteration_test

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"strconv"

	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"

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

func (test *TestIterationRepository) SetupTest() {
	test.clean = cleaner.DeleteCreatedEntities(test.DB)
}

func (test *TestIterationRepository) TearDownTest() {
	test.clean()
}

func (test *TestIterationRepository) TestCreateIteration() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := iteration.NewIterationRepository(test.DB)

	start := time.Now()
	end := start.Add(time.Hour * (24 * 8 * 3))
	name := "Sprint #24"

	newSpace := space.Space{
		Name: "Space 1",
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	assert.Nil(t, err)

	i := iteration.Iteration{
		Name:    name,
		SpaceID: space.ID,
		StartAt: &start,
		EndAt:   &end,
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

func (test *TestIterationRepository) TestCreateChildIteration() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := iteration.NewIterationRepository(test.DB)

	start := time.Now()
	end := start.Add(time.Hour * (24 * 8 * 3))
	name := "Sprint #24"
	name2 := "Sprint #24.1"

	newSpace := space.Space{
		Name: "Space 1",
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	assert.Nil(t, err)

	i := iteration.Iteration{
		Name:    name,
		SpaceID: space.ID,
		StartAt: &start,
		EndAt:   &end,
	}
	repo.Create(context.Background(), &i)

	parentPath := append(i.Path, i.ID)
	require.NotNil(t, parentPath)
	i2 := iteration.Iteration{
		Name:    name2,
		SpaceID: space.ID,
		StartAt: &start,
		EndAt:   &end,
		Path:    parentPath,
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

func (test *TestIterationRepository) TestListIterationBySpace() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := iteration.NewIterationRepository(test.DB)

	newSpace := space.Space{
		Name: "Space 1",
	}
	repoSpace := space.NewRepository(test.DB)
	spaceInstance, err := repoSpace.Create(context.Background(), &newSpace)
	assert.Nil(t, err)

	for i := 0; i < 3; i++ {
		start := time.Now()
		end := start.Add(time.Hour * (24 * 8 * 3))
		name := "Sprint #2" + strconv.Itoa(i)

		i := iteration.Iteration{
			Name:    name,
			SpaceID: spaceInstance.ID,
			StartAt: &start,
			EndAt:   &end,
		}
		e := repo.Create(context.Background(), &i)
		require.Nil(t, e)
	}
	// create another space and add iteration to another space
	anotherSpace := space.Space{
		Name: "Space 2",
	}
	anotherSpaceCreated, err := repoSpace.Create(context.Background(), &anotherSpace)
	assert.Nil(t, err)
	e := repo.Create(context.Background(), &iteration.Iteration{
		Name:    "Other Spring #2",
		SpaceID: anotherSpaceCreated.ID,
	})
	require.Nil(t, e)

	its, err := repo.List(context.Background(), spaceInstance.ID)
	assert.Nil(t, err)
	assert.Len(t, its, 3)
}

func (test *TestIterationRepository) TestUpdateIteration() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := iteration.NewIterationRepository(test.DB)

	start := time.Now()
	end := start.Add(time.Hour * (24 * 8 * 3))
	name := "Sprint #24"

	newSpace := space.Space{
		Name: "Space 1",
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	assert.Nil(t, err)

	i := iteration.Iteration{
		Name:    name,
		SpaceID: space.ID,
		StartAt: &start,
		EndAt:   &end,
	}
	// creates an iteration
	repo.Create(context.Background(), &i)
	require.NotEqual(t, uuid.Nil, i.ID, "Iteration was not created, ID nil")

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
