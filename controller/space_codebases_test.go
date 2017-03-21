package controller_test

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/context"

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

func (rest *TestSpaceCodebaseREST) TestSuccessCreateCodebase() {
	t := rest.T()

	resource.Require(t, resource.Database)

	repo := "https://github.com/almighty/almighty-core.git"

	var p *space.Space
	ci := createSpaceCodebase(repo)

	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Spaces()
		newSpace := &space.Space{
			Name: "Test 1" + uuid.NewV4().String(),
		}
		p, _ = repo.Create(context.Background(), newSpace)
		return nil
	})
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateSpaceCodebasesCreated(t, svc.Context, svc, ctrl, p.ID.String(), ci)
	require.NotNil(t, c.Data.ID)
	require.NotNil(t, c.Data.Relationships.Space)
	assert.Equal(t, p.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(t, repo, *c.Data.Attributes.URL)
}

func (rest *TestSpaceCodebaseREST) TestListCodebase() {
	t := rest.T()
	resource.Require(t, resource.Database)

	// Create a new space where we'll create 3 codebase
	var s *space.Space

	// Create another space where we'll create 1 codebase.
	var anotherSpace *space.Space

	application.Transactional(rest.db, func(app application.Application) error {
		var err error
		repo := app.Spaces()
		newSpace := &space.Space{
			Name: "Test Space 1" + uuid.NewV4().String(),
		}
		s, err = repo.Create(context.Background(), newSpace)
		require.Nil(t, err)

		newSpace = &space.Space{
			Name: "Another space" + uuid.NewV4().String(),
		}
		anotherSpace, err = repo.Create(context.Background(), newSpace)
		require.Nil(t, err)
		return nil
	})

	repo := "https://github.com/almighty/almighty-core.git"

	svc, ctrl := rest.SecuredController()
	spaceId := s.ID
	anotherSpaceId := anotherSpace.ID
	var createdSpacesUuids1 []uuid.UUID

	for i := 0; i < 3; i++ {
		repoURL := strings.Replace(repo, "core", "core"+strconv.Itoa(i), -1)
		spaceCodebaseContext := createSpaceCodebase(repoURL)
		_, c := test.CreateSpaceCodebasesCreated(t, svc.Context, svc, ctrl, spaceId.String(), spaceCodebaseContext)
		require.NotNil(t, c.Data.ID)
		require.NotNil(t, c.Data.Relationships.Space)
		createdSpacesUuids1 = append(createdSpacesUuids1, *c.Data.ID)
	}

	otherRepo := "https://github.com/fabric8io/fabric8-planner.git"
	anotherSpaceCodebaseContext := createSpaceCodebase(otherRepo)
	_, createdCodebase := test.CreateSpaceCodebasesCreated(t, svc.Context, svc, ctrl, anotherSpaceId.String(), anotherSpaceCodebaseContext)
	require.NotNil(t, createdCodebase)

	offset := "0"
	limit := 100

	svc, ctrl = rest.UnSecuredController()
	_, codebaseList := test.ListSpaceCodebasesOK(t, svc.Context, svc, ctrl, spaceId.String(), &limit, &offset)
	assert.Len(t, codebaseList.Data, 3)
	for i := 0; i < len(createdSpacesUuids1); i++ {
		assert.NotNil(t, searchInCodebaseSlice(createdSpacesUuids1[i], codebaseList))
	}

	_, anotherCodebaseList := test.ListSpaceCodebasesOK(t, svc.Context, svc, ctrl, anotherSpaceId.String(), &limit, &offset)
	require.Len(t, anotherCodebaseList.Data, 1)
	assert.Equal(t, anotherCodebaseList.Data[0].ID, createdCodebase.Data.ID)

}

func (rest *TestSpaceCodebaseREST) TestCreateCodebaseMissingSpace() {
	t := rest.T()
	resource.Require(t, resource.Database)

	ci := createSpaceCodebase("https://github.com/fabric8io/fabric8-planner.git")

	svc, ctrl := rest.SecuredController()
	test.CreateSpaceCodebasesNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), ci)
}

func (rest *TestSpaceCodebaseREST) TestFailCreateCodebaseNotAuthorized() {
	t := rest.T()
	resource.Require(t, resource.Database)

	ci := createSpaceCodebase("https://github.com/fabric8io/fabric8-planner.git")

	svc, ctrl := rest.UnSecuredController()
	test.CreateSpaceCodebasesUnauthorized(t, svc.Context, svc, ctrl, uuid.NewV4().String(), ci)
}

func (rest *TestSpaceCodebaseREST) TestFailListCodebaseByMissingSpace() {
	t := rest.T()
	resource.Require(t, resource.Database)

	offset := "0"
	limit := 100

	svc, ctrl := rest.UnSecuredController()
	test.ListSpaceCodebasesNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), &limit, &offset)
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
