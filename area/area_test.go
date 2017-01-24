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
	repo.Create(context.Background(), &i)

	// ltree field doesnt accept "-" , so we will save them as "_"
	expectedPath := strings.Replace((i.ID).String(), "-", "_", -1)
	area2 := area.Area{
		Name:    name2,
		SpaceID: uuid.NewV4(),
		Path:    expectedPath,
	}
	repo.Create(context.Background(), &area2)

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
