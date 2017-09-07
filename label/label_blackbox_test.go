package label_test

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	uuid "github.com/satori/go.uuid"
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
}
