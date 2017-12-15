package area_test

import (
	"context"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/area"
	errs "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/path"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestAreaRepository struct {
	gormtestsupport.DBTestSuite
}

func TestRunAreaRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestAreaRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *TestAreaRepository) TestCreateAreaWithSameNameFail() {
	// given
	repo := area.NewAreaRepository(s.DB)
	name := "TestCreateAreaWithSameNameFail"

	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Areas(1, func(fxt *tf.TestFixture, idx int) error {
		fxt.Areas[idx].Name = name
		return nil
	}))

	anotherAreaWithSameName := area.Area{
		Name:    name,
		SpaceID: fxt.Spaces[0].ID,
	}
	// when
	err := repo.Create(context.Background(), &anotherAreaWithSameName)
	// then
	require.Error(s.T(), err)
	// In case of unique constrain error, a DataConflictError is returned.
	_, ok := errors.Cause(err).(errs.DataConflictError)
	assert.True(s.T(), ok)
}

func (s *TestAreaRepository) TestCreateArea() {
	// given
	repo := area.NewAreaRepository(s.DB)
	name := "TestCreateArea"
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
	a := area.Area{
		Name:    name,
		SpaceID: fxt.Spaces[0].ID,
	}
	// when
	err := repo.Create(context.Background(), &a)
	// then
	require.NoError(s.T(), err)
	require.NotEqual(s.T(), uuid.Nil, a.ID)
	assert.True(s.T(), !a.CreatedAt.After(time.Now()), "Area was not created, CreatedAt after Now()?")
	assert.Equal(s.T(), name, a.Name)
}

func (s *TestAreaRepository) TestExistsArea() {
	t := s.T()
	resource.Require(t, resource.Database)
	repo := area.NewAreaRepository(s.DB)

	t.Run("area exists", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.Areas(1))
		// when
		err := repo.CheckExists(context.Background(), fxt.Areas[0].ID.String())
		// then
		require.NoError(t, err)
	})

	t.Run("area doesn't exist", func(t *testing.T) {
		// when
		err := repo.CheckExists(context.Background(), uuid.NewV4().String())
		// then
		require.IsType(t, errs.NotFoundError{}, err)
	})
}

func (s *TestAreaRepository) TestCreateChildArea() {
	// given
	var expectedPath path.Path
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Areas(2, func(fxt *tf.TestFixture, idx int) error {
			a := fxt.Areas[idx]
			switch idx {
			case 0:
				a.Name = "TestCreateChildArea"
				expectedPath = path.Path{a.ID}
			case 1:
				a.Name = "TestCreateChildArea.1"
				a.Path = expectedPath
			}
			return nil
		}),
	)
	// then
	actualArea, err := area.NewAreaRepository(s.DB).Load(context.Background(), fxt.Areas[1].ID)
	actualPath := actualArea.Path
	require.NoError(s.T(), err)
	require.NotNil(s.T(), actualArea)
	assert.Equal(s.T(), expectedPath, actualPath)
}

func (s *TestAreaRepository) TestGetAreaBySpaceIDAndNameAndPath() {
	// given a space and area with the same name.
	name := "space name " + uuid.NewV4().String()
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Spaces(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.Spaces[idx].Name = name
			return nil
		}),
		tf.Areas(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.Areas[idx].Name = name
			fxt.Areas[idx].Path = path.Path{}
			return nil
		}),
	)
	// when
	repo := area.NewAreaRepository(s.DB)
	areaList, err := repo.Query(area.FilterBySpaceID(fxt.Spaces[0].ID), area.FilterByPath(path.Path{}), area.FilterByName(name))
	// then
	require.NoError(s.T(), err)
	// there must be ONLY 1 result, because of the space,name,path unique constraint
	require.Len(s.T(), areaList, 1)
	rootArea := areaList[0]
	assert.Equal(s.T(), name, rootArea.Name)
	assert.Equal(s.T(), fxt.Spaces[0].ID, rootArea.SpaceID)
}

func (s *TestAreaRepository) TestListAreaBySpace() {
	// given two spaces and four areas (3 in 1st space and 1 in 2nd space)
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Spaces(2),
		tf.Areas(4, func(fxt *tf.TestFixture, idx int) error {
			if idx == 3 {
				fxt.Areas[idx].SpaceID = fxt.Spaces[1].ID
			}
			return nil
		}),
	)
	createdAreaIds := []uuid.UUID{fxt.Areas[0].ID, fxt.Areas[1].ID, fxt.Areas[2].ID}
	// when
	repo := area.NewAreaRepository(s.DB)
	its, err := repo.List(context.Background(), fxt.Spaces[0].ID)
	// then
	require.NoError(s.T(), err)
	require.Len(s.T(), its, 3)
	for i := 0; i < 3; i++ {
		assert.NotNil(s.T(), searchInAreaSlice(createdAreaIds[i], its))
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

func (s *TestAreaRepository) TestListChildrenOfParents() {
	// given 3 areas, the last two being a child of the first one

	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Areas(3, func(fxt *tf.TestFixture, idx int) error {
			a := fxt.Areas[idx]
			switch idx {
			case 1:
				a.Path = path.Path{fxt.Areas[0].ID}
			case 2:
				a.Path = path.Path{fxt.Areas[0].ID}
			}
			return nil
		}),
	)

	// then
	repo := area.NewAreaRepository(s.DB)

	s.T().Run("test paths of child areas", func(t *testing.T) {
		expectedPath := path.Path{fxt.Areas[0].ID}

		actualArea, err := repo.Load(context.Background(), fxt.Areas[1].ID)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, fxt.Areas[1].Path)
		assert.Equal(t, expectedPath, actualArea.Path) // check that path ( an ltree field ) was populated.

		actualArea, err = repo.Load(context.Background(), fxt.Areas[2].ID)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, fxt.Areas[2].Path)
		assert.Equal(t, expectedPath, actualArea.Path) // check that path ( an ltree field ) was populated.
	})

	s.T().Run("check that we have two child areas", func(t *testing.T) {
		// given
		childIDs := map[uuid.UUID]struct{}{
			fxt.Areas[1].ID: {},
			fxt.Areas[2].ID: {},
		}
		// when
		childAreaList, err := repo.ListChildren(context.Background(), fxt.Areas[0])
		// then
		require.NoError(t, err)
		for _, child := range childAreaList {
			delete(childIDs, child.ID)
		}
		require.Empty(t, childIDs)
	})
}

func (s *TestAreaRepository) TestListImmediateChildrenOfGrandParents() {

	// given 3 generations of areas (grandparent, parent, and child)

	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.Areas(3, func(fxt *tf.TestFixture, idx int) error {
			a := fxt.Areas[idx]
			switch idx {
			case 1:
				a.Path = path.Path{fxt.Areas[0].ID}
			case 2:
				a.Path = path.Path{fxt.Areas[0].ID, fxt.Areas[1].ID}
			}
			return nil
		}),
	)

	repo := area.NewAreaRepository(s.DB)

	s.T().Run("children of grandparent", func(t *testing.T) {
		// when
		childAreaList, err := repo.ListChildren(context.Background(), fxt.Areas[0])
		// then
		require.NoError(t, err)
		require.Len(t, childAreaList, 1)
		require.Equal(t, fxt.Areas[1].ID, childAreaList[0].ID)
	})

	s.T().Run("children of parent", func(t *testing.T) {
		// when
		childAreaList, err := repo.ListChildren(context.Background(), fxt.Areas[1])
		// then
		require.NoError(t, err)
		require.Len(t, childAreaList, 1)
		require.Equal(t, fxt.Areas[2].ID, childAreaList[0].ID)
	})

	s.T().Run("children of child (none)", func(t *testing.T) {
		// when
		childAreaList, err := repo.ListChildren(context.Background(), fxt.Areas[2])
		// then
		require.NoError(t, err)
		require.Len(t, childAreaList, 0)
	})
}

func (s *TestAreaRepository) TestListParentTree() {

	// given 2 areas (one is the parent, the other the child)

	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Areas(2, func(fxt *tf.TestFixture, idx int) error {
		a := fxt.Areas[idx]
		switch idx {
		case 1:
			a.Path = path.Path{fxt.Areas[idx-1].ID}
		}
		return nil
	}))

	listOfCreatedID := []uuid.UUID{fxt.Areas[0].ID, fxt.Areas[1].ID}
	// when
	listOfCreatedAreas, err := area.NewAreaRepository(s.DB).LoadMultiple(context.Background(), listOfCreatedID)
	// then
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 2, len(listOfCreatedAreas))
	for i := 0; i < 2; i++ {
		assert.NotNil(s.T(), searchInAreaSlice(listOfCreatedID[i], listOfCreatedAreas))
	}
}
