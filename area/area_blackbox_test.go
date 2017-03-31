package area_test

import (
	"strconv"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/path"
	"github.com/pkg/errors"

	localerror "github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestAreaRepository struct {
	gormtestsupport.DBTestSuite

	clean func()
}

func TestRunAreaRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestAreaRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (test *TestAreaRepository) SetupTest() {
	test.clean = cleaner.DeleteCreatedEntities(test.DB)
}

func (test *TestAreaRepository) TearDownTest() {
	test.clean()
}

func (test *TestAreaRepository) TestCreateAreaWithSameNameFail() {
	// given
	repo := area.NewAreaRepository(test.DB)
	name := "TestCreateAreaWithSameNameFail"
	newSpace := space.Space{
		Name: "Space 1 " + uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(test.T(), err)
	a := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}
	repo.Create(context.Background(), &a)
	require.NotEqual(test.T(), uuid.Nil, a.ID)
	require.False(test.T(), a.CreatedAt.After(time.Now()), "Area was not created, CreatedAt after Now()")
	assert.Equal(test.T(), name, a.Name)
	anotherAreaWithSameName := area.Area{
		Name:    a.Name,
		SpaceID: space.ID,
	}
	// when
	err = repo.Create(context.Background(), &anotherAreaWithSameName)
	// then
	require.NotNil(test.T(), err)
	// In case of unique constrain error, a BadParameterError is returned.
	_, ok := errors.Cause(err).(localerror.BadParameterError)
	assert.True(test.T(), ok)
}

func (test *TestAreaRepository) TestCreateArea() {
	// given
	repo := area.NewAreaRepository(test.DB)
	name := "TestCreateArea"
	newSpace := space.Space{
		Name: uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(test.T(), err)
	a := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}
	// when
	err = repo.Create(context.Background(), &a)
	// then
	require.Nil(test.T(), err)
	require.NotEqual(test.T(), uuid.Nil, a.ID)
	assert.True(test.T(), !a.CreatedAt.After(time.Now()), "Area was not created, CreatedAt after Now()?")
	assert.Equal(test.T(), name, a.Name)
}

func (test *TestAreaRepository) TestCreateChildArea() {
	// given
	repo := area.NewAreaRepository(test.DB)
	newSpace := space.Space{
		Name: uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(test.T(), err)
	name := "TestCreateChildArea"
	name2 := "TestCreateChildArea.1"
	i := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}
	err = repo.Create(context.Background(), &i)
	assert.Nil(test.T(), err)
	// ltree field doesnt accept "-" , so we will save them as "_"
	expectedPath := path.Path{i.ID}
	area2 := area.Area{
		Name:    name2,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	// when
	err = repo.Create(context.Background(), &area2)
	// then
	require.Nil(test.T(), err)
	actualArea, err := repo.Load(context.Background(), area2.ID)
	actualPath := actualArea.Path
	require.Nil(test.T(), err)
	require.NotNil(test.T(), actualArea)
	assert.Equal(test.T(), expectedPath, actualPath)
}

func (test *TestAreaRepository) TestGetAreaBySpaceIDAndNameAndPath() {
	t := test.T()

	resource.Require(t, resource.Database)

	repo := area.NewAreaRepository(test.DB)

	name := "space name " + uuid.NewV4().String()
	newSpace := space.Space{
		Name: name,
	}

	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(t, err)

	a := area.Area{
		Name:    name,
		SpaceID: space.ID,
		Path:    path.Path{},
	}
	err = repo.Create(context.Background(), &a)
	require.Nil(t, err)

	// So now we have a space and area with the same name.

	areaList, err := repo.Query(area.AreaFilterBySpaceID(space.ID), area.AreaFilterByPath(path.Path{}), area.AreaFilterByName(name))
	require.Nil(t, err)

	// there must be ONLY 1 result, because of the space,name,path unique constraint
	require.Len(t, areaList, 1)

	rootArea := areaList[0]
	assert.Equal(t, name, rootArea.Name)
	assert.Equal(t, space.ID, rootArea.SpaceID)
}

func (test *TestAreaRepository) TestListAreaBySpace() {
	// given
	repo := area.NewAreaRepository(test.DB)
	newSpace := space.Space{
		Name: uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space1, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(test.T(), err)

	var createdAreaIds []uuid.UUID
	for i := 0; i < 3; i++ {
		name := "Test Area #20" + strconv.Itoa(i)

		a := area.Area{
			Name:    name,
			SpaceID: space1.ID,
		}
		err := repo.Create(context.Background(), &a)
		require.Nil(test.T(), err)
		createdAreaIds = append(createdAreaIds, a.ID)
		test.T().Log(a.ID)
	}
	newSpace2 := space.Space{
		Name: uuid.NewV4().String(),
	}
	space2, err := repoSpace.Create(context.Background(), &newSpace2)
	require.Nil(test.T(), err)
	err = repo.Create(context.Background(), &area.Area{
		Name:    "Other Test area #20",
		SpaceID: space2.ID,
	})
	require.Nil(test.T(), err)
	// when
	its, err := repo.List(context.Background(), space1.ID)
	// then
	require.Nil(test.T(), err)
	require.Len(test.T(), its, 3)
	for i := 0; i < 3; i++ {
		assert.NotNil(test.T(), searchInAreaSlice(createdAreaIds[i], its))
	}
}

func searchInAreaSlice(searchKey uuid.UUID, areaList []area.Area) *area.Area {
	for i := 0; i < len(areaList); i++ {
		if searchKey == areaList[i].ID {
			return &areaList[i]
		}
	}
	return nil
}

func (test *TestAreaRepository) TestListChildrenOfParents() {
	// given
	resource.Require(test.T(), resource.Database)
	repo := area.NewAreaRepository(test.DB)
	name := "TestListChildrenOfParents"
	name2 := "TestListChildrenOfParents.1"
	name3 := "TestListChildrenOfParents.2"
	var createdAreaIDs []uuid.UUID
	newSpace := space.Space{
		Name: uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(test.T(), err)
	// *** Create Parent Area ***
	i := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}
	err = repo.Create(context.Background(), &i)
	require.Nil(test.T(), err)
	// *** Create 1st child area ***
	// ltree field doesnt accept "-" , so we will save them as "_"
	expectedPath := path.Path{i.ID}
	area2 := area.Area{
		Name:    name2,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area2)
	require.Nil(test.T(), err)
	createdAreaIDs = append(createdAreaIDs, area2.ID)
	actualArea, err := repo.Load(context.Background(), area2.ID)
	actualPath := actualArea.Path
	require.Nil(test.T(), err)
	assert.NotEqual(test.T(), uuid.Nil, area2.Path)
	assert.Equal(test.T(), expectedPath, actualPath) // check that path ( an ltree field ) was populated.
	// *** Create 2nd child area ***
	expectedPath = path.Path{i.ID}
	area3 := area.Area{
		Name:    name3,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area3)
	require.Nil(test.T(), err)
	createdAreaIDs = append(createdAreaIDs, area3.ID)
	actualArea, err = repo.Load(context.Background(), area3.ID)
	require.Nil(test.T(), err)
	actualPath = actualArea.Path
	assert.Equal(test.T(), expectedPath, actualPath)
	// *** Validate that there are 2 children
	childAreaList, err := repo.ListChildren(context.Background(), &i)
	require.Nil(test.T(), err)
	assert.Equal(test.T(), 2, len(childAreaList))
	for i := 0; i < len(createdAreaIDs); i++ {
		assert.NotNil(test.T(), createdAreaIDs[i], childAreaList[i].ID)
	}
}

func (test *TestAreaRepository) TestListImmediateChildrenOfGrandParents() {
	// given
	repo := area.NewAreaRepository(test.DB)
	name := "TestListImmediateChildrenOfGrandParents"
	name2 := "TestListImmediateChildrenOfGrandParents.1"
	name3 := "TestListImmediateChildrenOfGrandParents.1.3"
	newSpace := space.Space{
		Name: uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(test.T(), err)
	// *** Create Parent Area ***
	i := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}
	err = repo.Create(context.Background(), &i)
	assert.Nil(test.T(), err)
	// *** Create 'son' area ***
	expectedPath := path.Path{i.ID}
	area2 := area.Area{
		Name:    name2,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area2)
	require.Nil(test.T(), err)
	childAreaList, err := repo.ListChildren(context.Background(), &i)
	assert.Equal(test.T(), 1, len(childAreaList))
	require.Nil(test.T(), err)
	// *** Create 'grandson' area ***
	expectedPath = path.Path{i.ID, area2.ID}
	area4 := area.Area{
		Name:    name3,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area4)
	require.Nil(test.T(), err)
	// when
	childAreaList, err = repo.ListChildren(context.Background(), &i)
	// But , There is only 1 'son' .
	require.Nil(test.T(), err)
	assert.Equal(test.T(), 1, len(childAreaList))
	assert.Equal(test.T(), area2.ID, childAreaList[0].ID)
	// *** Confirm the grandson has no son
	childAreaList, err = repo.ListChildren(context.Background(), &area4)
	assert.Equal(test.T(), 0, len(childAreaList))
}

func (test *TestAreaRepository) TestListParentTree() {
	// given
	repo := area.NewAreaRepository(test.DB)
	name := "TestListParentTree"
	name2 := "TestListParentTree.1"
	newSpace := space.Space{
		Name: uuid.NewV4().String(),
	}
	repoSpace := space.NewRepository(test.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(test.T(), err)
	// *** Create Parent Area ***
	i := area.Area{
		Name:    name,
		SpaceID: newSpace.ID,
	}
	err = repo.Create(context.Background(), &i)
	assert.Nil(test.T(), err)
	// *** Create 'son' area ***
	expectedPath := path.Path{i.ID}
	area2 := area.Area{
		Name:    name2,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area2)
	require.Nil(test.T(), err)
	listOfCreatedID := []uuid.UUID{i.ID, area2.ID}
	// when
	listOfCreatedAreas, err := repo.LoadMultiple(context.Background(), listOfCreatedID)
	// then
	require.Nil(test.T(), err)
	assert.Equal(test.T(), 2, len(listOfCreatedAreas))
	for i := 0; i < 2; i++ {
		assert.NotNil(test.T(), searchInAreaSlice(listOfCreatedID[i], listOfCreatedAreas))
	}

}
