package area_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestAreaRepository struct {
	gormsupport.DBTestSuite

	clean func()
}

func TestRunAreaRepository(t *testing.T) {
	suite.Run(t, &TestAreaRepository{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

func (test *TestAreaRepository) SetupTest() {
	test.clean = gormsupport.DeleteCreatedEntities(test.DB)
}

func (test *TestAreaRepository) TearDownTest() {
	test.clean()
}

func (test *TestAreaRepository) TestCreateArea() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := area.NewAreaRepository(test.DB)

	name := "Area 21"

	i := area.Area{
		Name:    name,
		SpaceID: uuid.NewV4(),
	}

	repo.Create(context.Background(), &i)
	if i.ID == uuid.Nil {
		t.Errorf("Area was not created, ID nil")
	}

	if i.CreatedAt.After(time.Now()) {
		t.Errorf("Area was not created, CreatedAt after Now()?")
	}

	assert.Equal(t, name, i.Name)
}

func (test *TestAreaRepository) TestCreateChildArea() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := area.NewAreaRepository(test.DB)

	name := "Area #24"
	name2 := "Area #24.1"

	i := area.Area{
		Name:    name,
		SpaceID: uuid.NewV4(),
	}
	err := repo.Create(context.Background(), &i)
	assert.Nil(t, err)

	// ltree field doesnt accept "-" , so we will save them as "_"
	expectedPath := strings.Replace((i.ID).String(), "-", "_", -1)
	area2 := area.Area{
		Name:    name2,
		SpaceID: uuid.NewV4(),
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area2)
	assert.Nil(t, err)

	actualArea, err := repo.Load(context.Background(), area2.ID)
	actualPath := actualArea.Path
	require.Nil(t, err)
	assert.NotEqual(t, uuid.Nil, area2.Path)
	assert.Equal(t, expectedPath, actualPath)

}

func (test *TestAreaRepository) TestListAreaBySpace() {
	t := test.T()
	resource.Require(t, resource.Database)

	repo := area.NewAreaRepository(test.DB)

	spaceID := uuid.NewV4()

	for i := 0; i < 3; i++ {
		name := "Test Area #20" + strconv.Itoa(i)

		i := area.Area{
			Name:    name,
			SpaceID: spaceID,
		}
		repo.Create(context.Background(), &i)
	}
	repo.Create(context.Background(), &area.Area{
		Name:    "Other Test area #20",
		SpaceID: uuid.NewV4(),
	})

	its, err := repo.List(context.Background(), spaceID)
	assert.Nil(t, err)
	assert.Len(t, its, 3)
}

func (test *TestAreaRepository) TestListChildrenOfParents() {
	t := test.T()
	resource.Require(t, resource.Database)
	test.DBTestSuite.DB = test.DBTestSuite.DB.Debug()
	repo := area.NewAreaRepository(test.DB)

	name := "Area #240"
	name2 := "Area #240.1"
	name3 := "Area #240.2"

	// *** Create Parent Area ***

	i := area.Area{
		Name:    name,
		SpaceID: uuid.NewV4(),
	}
	err := repo.Create(context.Background(), &i)
	assert.Nil(t, err)

	// *** Create 1st child area ***

	// ltree field doesnt accept "-" , so we will save them as "_"
	expectedPath := strings.Replace((i.ID).String(), "-", "_", -1)
	area2 := area.Area{
		Name:    name2,
		SpaceID: uuid.NewV4(),
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area2)
	assert.Nil(t, err)

	actualArea, err := repo.Load(context.Background(), area2.ID)
	actualPath := actualArea.Path
	require.Nil(t, err)
	assert.NotEqual(t, uuid.Nil, area2.Path)
	assert.Equal(t, expectedPath, actualPath) // check that path ( an ltree field ) was populated.

	// *** Create 2nd child area ***

	expectedPath = strings.Replace((i.ID).String(), "-", "_", -1)
	area3 := area.Area{
		Name:    name3,
		SpaceID: uuid.NewV4(),
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area3)
	require.Nil(t, err)
	actualArea, err = repo.Load(context.Background(), area3.ID)

	childAreaList, err := repo.ListChildren(context.Background(), i.ID)
	assert.Equal(t, 2, len(childAreaList))
	require.Nil(t, err)

}

func (test *TestAreaRepository) TestListImmediateChildrenOfGrandParents() {
	t := test.T()
	resource.Require(t, resource.Database)
	test.DBTestSuite.DB = test.DBTestSuite.DB.Debug()
	repo := area.NewAreaRepository(test.DB)

	name := "Area #240"
	name2 := "Area #240.1"
	name3 := "Area #240.1.3"

	// *** Create Parent Area ***

	i := area.Area{
		Name:    name,
		SpaceID: uuid.NewV4(),
	}
	err := repo.Create(context.Background(), &i)
	assert.Nil(t, err)

	// *** Create 'son' area ***

	expectedPath := strings.Replace((i.ID).String(), "-", "_", -1)
	area2 := area.Area{
		Name:    name2,
		SpaceID: uuid.NewV4(),
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area2)
	require.Nil(t, err)

	childAreaList, err := repo.ListChildren(context.Background(), i.ID)
	assert.Equal(t, 1, len(childAreaList))
	require.Nil(t, err)

	// *** Create 'grandson' area ***

	expectedPath = strings.Replace((i.ID).String()+"."+(area2.ID.String()), "-", "_", -1)
	area4 := area.Area{
		Name:    name3,
		SpaceID: uuid.NewV4(),
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area4)
	require.Nil(t, err)

	childAreaList, err = repo.ListChildren(context.Background(), i.ID)

	// But , There is only 1 'son' .

	assert.Equal(t, 1, len(childAreaList))
	require.Nil(t, err)
}
