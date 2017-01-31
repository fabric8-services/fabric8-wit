package iteration_test

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"strconv"

	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/resource"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestIterationRepository struct {
	gormsupport.DBTestSuite

	clean func()
}

func TestRunIterationRepository(t *testing.T) {
	suite.Run(t, &TestIterationRepository{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
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

	i := iteration.Iteration{
		Name:    name,
		SpaceID: uuid.NewV4(),
		StartAt: &start,
		EndAt:   &end,
	}

	repo.Create(context.Background(), &i)
	if i.ID == uuid.Nil {
		t.Errorf("Comment was not created, ID nil")
	}

	if i.CreatedAt.After(time.Now()) {
		t.Errorf("Comment was not created, CreatedAt after Now()?")
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

	i := iteration.Iteration{
		Name:    name,
		SpaceID: uuid.NewV4(),
		StartAt: &start,
		EndAt:   &end,
	}
	repo.Create(context.Background(), &i)

	i2 := iteration.Iteration{
		Name:     name2,
		SpaceID:  uuid.NewV4(),
		StartAt:  &start,
		EndAt:    &end,
		ParentID: i.ID,
	}
	repo.Create(context.Background(), &i2)

	i2L, err := repo.Load(context.Background(), i2.ID)
	require.Nil(t, err)
	assert.NotEqual(t, uuid.Nil, i2.ParentID)
	assert.Equal(t, i2.ParentID, i2L.ParentID)
}

func (test *TestIterationRepository) TestListIterationBySpace() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := iteration.NewIterationRepository(test.DB)

	spaceID := uuid.NewV4()

	for i := 0; i < 3; i++ {
		start := time.Now()
		end := start.Add(time.Hour * (24 * 8 * 3))
		name := "Sprint #2" + strconv.Itoa(i)

		i := iteration.Iteration{
			Name:    name,
			SpaceID: spaceID,
			StartAt: &start,
			EndAt:   &end,
		}
		repo.Create(context.Background(), &i)
	}
	repo.Create(context.Background(), &iteration.Iteration{
		Name:    "Other Spring #2",
		SpaceID: uuid.NewV4(),
	})

	its, err := repo.List(context.Background(), spaceID)
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

	i := iteration.Iteration{
		Name:    name,
		SpaceID: uuid.NewV4(),
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
