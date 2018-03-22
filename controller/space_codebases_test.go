package controller_test

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"context"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SpaceCodebaseControllerTestSuite struct {
	gormtestsupport.DBTestSuite
	db *gormapplication.GormDB
}

func TestSpaceCodebaseController(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		require.NoError(t, err)
	}
	suite.Run(t, &SpaceCodebaseControllerTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (s *SpaceCodebaseControllerTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.db = gormapplication.NewGormDB(s.DB)
}

func (s *SpaceCodebaseControllerTestSuite) SecuredController() (*goa.Service, *SpaceCodebasesController) {
	svc := testsupport.ServiceAsUser("SpaceCodebase-Service", testsupport.TestIdentity)
	return svc, NewSpaceCodebasesController(svc, s.db)
}

func (s *SpaceCodebaseControllerTestSuite) UnSecuredController() (*goa.Service, *SpaceCodebasesController) {
	svc := goa.New("SpaceCodebase-Service")
	return svc, NewSpaceCodebasesController(svc, s.db)
}

func (s *SpaceCodebaseControllerTestSuite) TestCreateCodebaseCreated() {
	sp := s.createSpace(testsupport.TestIdentity.ID)
	stackId := "stackId"
	ci := createSpaceCodebase("https://github.com/fabric8-services/fabric8-wit.git", &stackId)

	svc, ctrl := s.SecuredController()
	_, c := test.CreateSpaceCodebasesCreated(s.T(), svc.Context, svc, ctrl, sp.ID, ci)
	require.NotNil(s.T(), c.Data.ID)
	require.NotNil(s.T(), c.Data.Relationships.Space)
	assert.Equal(s.T(), sp.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(s.T(), "https://github.com/fabric8-services/fabric8-wit.git", *c.Data.Attributes.URL)
	assert.Equal(s.T(), "stackId", *c.Data.Attributes.StackID)
}

func (s *SpaceCodebaseControllerTestSuite) TestCreateCodebaseWithNoStackIdCreated() {
	sp := s.createSpace(testsupport.TestIdentity.ID)
	ci := createSpaceCodebase("https://github.com/fabric8-services/fabric8-wit.git", nil)

	svc, ctrl := s.SecuredController()
	_, c := test.CreateSpaceCodebasesCreated(s.T(), svc.Context, svc, ctrl, sp.ID, ci)
	require.NotNil(s.T(), c.Data.ID)
	require.NotNil(s.T(), c.Data.Relationships.Space)
	assert.Equal(s.T(), sp.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(s.T(), "https://github.com/fabric8-services/fabric8-wit.git", *c.Data.Attributes.URL)
	assert.Nil(s.T(), c.Data.Attributes.StackID)
}

func (s *SpaceCodebaseControllerTestSuite) TestCreateCodebaseForbidden() {
	sp := s.createSpace(testsupport.TestIdentity2.ID)
	stackId := "stackId"
	ci := createSpaceCodebase("https://github.com/fabric8-services/fabric8-wit.git", &stackId)

	svc, ctrl := s.SecuredController()
	// Codebase creation is forbidden if the user is not the space owner
	test.CreateSpaceCodebasesForbidden(s.T(), svc.Context, svc, ctrl, sp.ID, ci)
}

func (s *SpaceCodebaseControllerTestSuite) TestListCodebase() {
	t := s.T()
	resource.Require(t, resource.Database)

	// Create a new space where we'll create 3 codebase
	sp := s.createSpace(testsupport.TestIdentity.ID)
	// Create another space where we'll create 1 codebase.
	anotherSpace := s.createSpace(testsupport.TestIdentity.ID)

	repo := "https://github.com/fabric8-services/fabric8-wit.git"

	svc, ctrl := s.SecuredController()
	spaceId := sp.ID
	anotherSpaceId := anotherSpace.ID
	var createdSpacesUuids1 []uuid.UUID

	for i := 0; i < 3; i++ {
		repoURL := strings.Replace(repo, "core", "core"+strconv.Itoa(i), -1)
		stackId := "stackId"
		spaceCodebaseContext := createSpaceCodebase(repoURL, &stackId)
		_, c := test.CreateSpaceCodebasesCreated(t, svc.Context, svc, ctrl, spaceId, spaceCodebaseContext)
		require.NotNil(t, c.Data.ID)
		require.NotNil(t, c.Data.Relationships.Space)
		createdSpacesUuids1 = append(createdSpacesUuids1, *c.Data.ID)
	}

	otherRepo := "https://github.com/fabric8io/fabric8-planner.git"
	stackId := "stackId"
	anotherSpaceCodebaseContext := createSpaceCodebase(otherRepo, &stackId)
	_, createdCodebase := test.CreateSpaceCodebasesCreated(t, svc.Context, svc, ctrl, anotherSpaceId, anotherSpaceCodebaseContext)
	require.NotNil(t, createdCodebase)

	offset := "0"
	limit := 100

	svc, ctrl = s.UnSecuredController()
	_, codebaseList := test.ListSpaceCodebasesOK(t, svc.Context, svc, ctrl, spaceId, &limit, &offset)
	assert.Len(t, codebaseList.Data, 3)
	for i := 0; i < len(createdSpacesUuids1); i++ {
		assert.NotNil(t, searchInCodebaseSlice(createdSpacesUuids1[i], codebaseList))
	}

	_, anotherCodebaseList := test.ListSpaceCodebasesOK(t, svc.Context, svc, ctrl, anotherSpaceId, &limit, &offset)
	require.Len(t, anotherCodebaseList.Data, 1)
	assert.Equal(t, anotherCodebaseList.Data[0].ID, createdCodebase.Data.ID)

}

func (s *SpaceCodebaseControllerTestSuite) TestCreateCodebaseMissingSpace() {
	t := s.T()
	resource.Require(t, resource.Database)
	stackId := "stackId"

	ci := createSpaceCodebase("https://github.com/fabric8io/fabric8-planner.git", &stackId)

	svc, ctrl := s.SecuredController()
	test.CreateSpaceCodebasesNotFound(t, svc.Context, svc, ctrl, uuid.NewV4(), ci)
}

func (s *SpaceCodebaseControllerTestSuite) TestFailCreateCodebaseNotAuthorized() {
	t := s.T()
	resource.Require(t, resource.Database)
	stackId := "stackId"

	ci := createSpaceCodebase("https://github.com/fabric8io/fabric8-planner.git", &stackId)

	svc, ctrl := s.UnSecuredController()
	test.CreateSpaceCodebasesUnauthorized(t, svc.Context, svc, ctrl, uuid.NewV4(), ci)
}

func (s *SpaceCodebaseControllerTestSuite) TestFailListCodebaseByMissingSpace() {
	t := s.T()
	resource.Require(t, resource.Database)

	offset := "0"
	limit := 100

	svc, ctrl := s.UnSecuredController()
	test.ListSpaceCodebasesNotFound(t, svc.Context, svc, ctrl, uuid.NewV4(), &limit, &offset)
}

func searchInCodebaseSlice(searchKey uuid.UUID, codebaseList *app.CodebaseList) *app.Codebase {
	for i := 0; i < len(codebaseList.Data); i++ {
		if searchKey == *codebaseList.Data[i].ID {
			return codebaseList.Data[i]
		}
	}
	return nil
}

func createSpaceCodebase(url string, stackId *string) *app.CreateSpaceCodebasesPayload {
	repoType := "git"
	return &app.CreateSpaceCodebasesPayload{
		Data: &app.Codebase{
			Type: APIStringTypeCodebase,
			Attributes: &app.CodebaseAttributes{
				Type:    &repoType,
				URL:     &url,
				StackID: stackId,
			},
		},
	}
}

func (s *SpaceCodebaseControllerTestSuite) createSpace(ownerID uuid.UUID) *space.Space {
	var sp *space.Space
	var err error
	err = application.Transactional(s.db, func(app application.Application) error {
		repo := app.Spaces()
		newSpace := &space.Space{
			Name:    "TestSpaceCodebase " + uuid.NewV4().String(),
			OwnerID: ownerID,
		}
		sp, err = repo.Create(context.Background(), newSpace)
		return err
	})
	require.NoError(s.T(), err)
	return sp
}
