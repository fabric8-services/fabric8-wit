package label_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	errs "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestLabelRepository struct {
	gormtestsupport.DBTestSuite
	testIdentity account.Identity
	clean        func()
}

func TestRunLabelRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestLabelRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *TestLabelRepository) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "WorkItemSuite setup user", "test provider")
	require.Nil(s.T(), err)
	s.testIdentity = *testIdentity
}

func (s *TestLabelRepository) TearDownTest() {
	s.clean()
}

func (s *TestLabelRepository) TestCreateLabel() {
	repo := label.NewLabelRepository(s.DB)
	newSpace := space.Space{
		Name:    "Space 1 " + uuid.NewV4().String(),
		OwnerId: s.testIdentity.ID,
	}
	repoSpace := space.NewRepository(s.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(s.T(), err)
	name := "TestCreateLabel"
	l := label.Label{
		SpaceID: space.ID,
		Name:    name,
	}
	repo.Create(context.Background(), &l)
	require.NotEqual(s.T(), uuid.Nil, l.ID)
	require.Equal(s.T(), "#000000", l.TextColor)
	require.Equal(s.T(), "#FFFFFF", l.BackgroundColor)
	require.False(s.T(), l.CreatedAt.After(time.Now()), "Label was not created, CreatedAt after Now()")
}

func (s *TestLabelRepository) TestCreateLabelWithSameName() {
	repo := label.NewLabelRepository(s.DB)
	newSpace := space.Space{
		Name:    "Space 1 " + uuid.NewV4().String(),
		OwnerId: s.testIdentity.ID,
	}
	repoSpace := space.NewRepository(s.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(s.T(), err)
	name := "TestCreateLabel"
	l := label.Label{
		SpaceID: space.ID,
		Name:    name,
	}
	repo.Create(context.Background(), &l)
	require.NotEqual(s.T(), uuid.Nil, l.ID)
	require.Equal(s.T(), "#000000", l.TextColor)
	require.Equal(s.T(), "#FFFFFF", l.BackgroundColor)
	require.False(s.T(), l.CreatedAt.After(time.Now()), "Label was not created, CreatedAt after Now()")

	err = repo.Create(context.Background(), &l)
	require.NotNil(s.T(), err)
	_, ok := errors.Cause(err).(errs.DataConflictError)
	assert.True(s.T(), ok)
}

func (s *TestLabelRepository) TestCreateLabelWithWrongColorCode() {
	repo := label.NewLabelRepository(s.DB)
	newSpace := space.Space{
		Name:    "Space 1 " + uuid.NewV4().String(),
		OwnerId: s.testIdentity.ID,
	}
	repoSpace := space.NewRepository(s.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(s.T(), err)
	name := "TestCreateLabel"
	l := label.Label{
		SpaceID:   space.ID,
		Name:      name,
		TextColor: "#yyppww",
	}
	err = repo.Create(context.Background(), &l)
	require.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "labels_text_color_check")

	l2 := label.Label{
		SpaceID:         space.ID,
		Name:            name,
		BackgroundColor: "#yyppww",
	}
	err = repo.Create(context.Background(), &l2)
	require.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "labels_background_color_check")
}

func (s *TestLabelRepository) TestListLabelBySpace() {
	repo := label.NewLabelRepository(s.DB)
	newSpace := space.Space{
		Name:    "Space 1 " + uuid.NewV4().String(),
		OwnerId: s.testIdentity.ID,
	}
	repoSpace := space.NewRepository(s.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(s.T(), err)

	var labelIDs []uuid.UUID
	for i := 0; i < 3; i++ {
		name := "Test Label #" + strconv.Itoa(i)
		l := label.Label{
			SpaceID: space.ID,
			Name:    name,
		}
		err := repo.Create(context.Background(), &l)
		require.Nil(s.T(), err)
		require.NotEqual(s.T(), uuid.Nil, l.ID)
		labelIDs = append(labelIDs, l.ID)
	}

	lbls, err := repo.List(context.Background(), space.ID)
	require.Nil(s.T(), err)
	require.Len(s.T(), lbls, 3)
	for i := 0; i < 3; i++ {
		assert.NotNil(s.T(), searchInLabelSlice(labelIDs[i], lbls))
	}
}

func searchInLabelSlice(searchKey uuid.UUID, labelList []label.Label) *label.Label {
	for i := 0; i < len(labelList); i++ {
		if searchKey == labelList[i].ID {
			return &labelList[i]
		}
	}
	return nil
}
