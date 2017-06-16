package controller_test

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"

	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestSpaceCodebaseREST struct {
	gormtestsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunSpaceCodebaseREST(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		require.Nil(t, err)
	}
	suite.Run(t, &TestSpaceCodebaseREST{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (rest *TestSpaceCodebaseREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestSpaceCodebaseREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSpaceCodebaseREST) SecuredController() (*goa.Service, *SpaceCodebasesController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	//priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("SpaceCodebase-Service", almtoken.NewManager(pub), testsupport.TestIdentity)
	return svc, NewSpaceCodebasesController(svc, rest.db)
}

func (rest *TestSpaceCodebaseREST) UnSecuredController() (*goa.Service, *SpaceCodebasesController) {
	svc := goa.New("SpaceCodebase-Service")
	return svc, NewSpaceCodebasesController(svc, rest.db)
}

func (rest *TestSpaceCodebaseREST) TestCreateCodebaseCreated() {
	s := rest.createSpace(testsupport.TestIdentity.ID)
	ci := createSpaceCodebase("https://github.com/almighty/almighty-core.git")

	svc, ctrl := rest.SecuredController()
	_, c := test.CreateSpaceCodebasesCreated(rest.T(), svc.Context, svc, ctrl, s.ID, ci)
	require.NotNil(rest.T(), c.Data.ID)
	require.NotNil(rest.T(), c.Data.Relationships.Space)
	assert.Equal(rest.T(), s.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(rest.T(), "https://github.com/almighty/almighty-core.git", *c.Data.Attributes.URL)
}

func (rest *TestSpaceCodebaseREST) TestCreateCodebaseForbidden() {
	s := rest.createSpace(testsupport.TestIdentity2.ID)
	ci := createSpaceCodebase("https://github.com/almighty/almighty-core.git")

	svc, ctrl := rest.SecuredController()
	// Codebase creation is forbidden if the user is not the space owner
	test.CreateSpaceCodebasesForbidden(rest.T(), svc.Context, svc, ctrl, s.ID, ci)
}

func (rest *TestSpaceCodebaseREST) TestListCodebase() {
	t := rest.T()
	resource.Require(t, resource.Database)

	// Create a new space where we'll create 3 codebase
	s := rest.createSpace(testsupport.TestIdentity.ID)
	// Create another space where we'll create 1 codebase.
	anotherSpace := rest.createSpace(testsupport.TestIdentity.ID)

	repo := "https://github.com/almighty/almighty-core.git"

	svc, ctrl := rest.SecuredController()
	spaceId := s.ID
	anotherSpaceId := anotherSpace.ID
	var createdSpacesUuids1 []uuid.UUID

	for i := 0; i < 3; i++ {
		repoURL := strings.Replace(repo, "core", "core"+strconv.Itoa(i), -1)
		spaceCodebaseContext := createSpaceCodebase(repoURL)
		_, c := test.CreateSpaceCodebasesCreated(t, svc.Context, svc, ctrl, spaceId, spaceCodebaseContext)
		require.NotNil(t, c.Data.ID)
		require.NotNil(t, c.Data.Relationships.Space)
		createdSpacesUuids1 = append(createdSpacesUuids1, *c.Data.ID)
	}

	otherRepo := "https://github.com/fabric8io/fabric8-planner.git"
	anotherSpaceCodebaseContext := createSpaceCodebase(otherRepo)
	_, createdCodebase := test.CreateSpaceCodebasesCreated(t, svc.Context, svc, ctrl, anotherSpaceId, anotherSpaceCodebaseContext)
	require.NotNil(t, createdCodebase)

	offset := "0"
	limit := 100

	svc, ctrl = rest.UnSecuredController()
	_, codebaseList := test.ListSpaceCodebasesOK(t, svc.Context, svc, ctrl, spaceId, &limit, &offset)
	assert.Len(t, codebaseList.Data, 3)
	for i := 0; i < len(createdSpacesUuids1); i++ {
		assert.NotNil(t, searchInCodebaseSlice(createdSpacesUuids1[i], codebaseList))
	}

	_, anotherCodebaseList := test.ListSpaceCodebasesOK(t, svc.Context, svc, ctrl, anotherSpaceId, &limit, &offset)
	require.Len(t, anotherCodebaseList.Data, 1)
	assert.Equal(t, anotherCodebaseList.Data[0].ID, createdCodebase.Data.ID)

}

func (rest *TestSpaceCodebaseREST) TestCreateCodebaseMissingSpace() {
	t := rest.T()
	resource.Require(t, resource.Database)

	ci := createSpaceCodebase("https://github.com/fabric8io/fabric8-planner.git")

	svc, ctrl := rest.SecuredController()
	test.CreateSpaceCodebasesNotFound(t, svc.Context, svc, ctrl, uuid.NewV4(), ci)
}

func (rest *TestSpaceCodebaseREST) TestFailCreateCodebaseNotAuthorized() {
	t := rest.T()
	resource.Require(t, resource.Database)

	ci := createSpaceCodebase("https://github.com/fabric8io/fabric8-planner.git")

	svc, ctrl := rest.UnSecuredController()
	test.CreateSpaceCodebasesUnauthorized(t, svc.Context, svc, ctrl, uuid.NewV4(), ci)
}

func (rest *TestSpaceCodebaseREST) TestFailListCodebaseByMissingSpace() {
	t := rest.T()
	resource.Require(t, resource.Database)

	offset := "0"
	limit := 100

	svc, ctrl := rest.UnSecuredController()
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

func createSpaceCodebase(url string) *app.CreateSpaceCodebasesPayload {
	repoType := "git"
	return &app.CreateSpaceCodebasesPayload{
		Data: &app.Codebase{
			Type: APIStringTypeCodebase,
			Attributes: &app.CodebaseAttributes{
				Type: &repoType,
				URL:  &url,
			},
		},
	}
}

func (rest *TestSpaceCodebaseREST) createSpace(ownerID uuid.UUID) *space.Space {
	resource.Require(rest.T(), resource.Database)

	var s *space.Space
	var err error
	err = application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Spaces()
		newSpace := &space.Space{
			Name:    "TestSpaceCodebase " + uuid.NewV4().String(),
			OwnerId: ownerID,
		}
		s, err = repo.Create(context.Background(), newSpace)
		return err
	})
	require.Nil(rest.T(), err)
	return s
}
