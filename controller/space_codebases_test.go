package controller_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestSpaceCodebaseREST struct {
	gormtestsupport.DBTestSuite
}

func TestRunSpaceCodebaseREST(t *testing.T) {
	suite.Run(t, &TestSpaceCodebaseREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (rest *TestSpaceCodebaseREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
}

func (rest *TestSpaceCodebaseREST) SecuredControllerWithIdentity(idn *account.Identity) (*goa.Service, *SpaceCodebasesController) {
	svc := testsupport.ServiceAsUser("SpaceCodebase-Service", *idn)
	return svc, NewSpaceCodebasesController(svc, rest.GormDB)
}

func (rest *TestSpaceCodebaseREST) SecuredController() (*goa.Service, *SpaceCodebasesController) {
	svc := testsupport.ServiceAsUser("SpaceCodebase-Service", testsupport.TestIdentity)
	return svc, NewSpaceCodebasesController(svc, rest.GormDB)
}

func (rest *TestSpaceCodebaseREST) UnSecuredController() (*goa.Service, *SpaceCodebasesController) {
	svc := goa.New("SpaceCodebase-Service")
	return svc, NewSpaceCodebasesController(svc, rest.GormDB)
}

func (rest *TestSpaceCodebaseREST) TestCreateCodebaseCreated() {
	sp := rest.createSpace(testsupport.TestIdentity.ID)
	stackId := "stackId"
	ci := createSpaceCodebase("https://github.com/fabric8-services/fabric8-wit.git", &stackId)

	t := rest.T()
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateSpaceCodebasesCreated(t, svc.Context, svc, ctrl, sp.ID, ci)
	require.NotNil(t, c.Data.ID)
	require.NotNil(t, c.Data.Relationships.Space)
	assert.Equal(t, sp.ID.String(), *c.Data.Relationships.Space.Data.ID)
	require.NotNil(t, c.Data.Attributes.CveScan)
	assert.Equal(t, true, *c.Data.Attributes.CveScan)
	assert.Equal(t, "https://github.com/fabric8-services/fabric8-wit.git", *c.Data.Attributes.URL)
	assert.Equal(t, "stackId", *c.Data.Attributes.StackID)
}

func (rest *TestSpaceCodebaseREST) TestCreateCodebaseWithNoStackIdCreated() {
	sp := rest.createSpace(testsupport.TestIdentity.ID)
	ci := createSpaceCodebase("https://github.com/fabric8-services/fabric8-wit.git", nil)

	svc, ctrl := rest.SecuredController()
	_, c := test.CreateSpaceCodebasesCreated(rest.T(), svc.Context, svc, ctrl, sp.ID, ci)
	require.NotNil(rest.T(), c.Data.ID)
	require.NotNil(rest.T(), c.Data.Relationships.Space)
	assert.Equal(rest.T(), sp.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(rest.T(), "https://github.com/fabric8-services/fabric8-wit.git", *c.Data.Attributes.URL)
	assert.Nil(rest.T(), c.Data.Attributes.StackID)
}

func (rest *TestSpaceCodebaseREST) TestCreateCodebaseForbidden() {
	sp := rest.createSpace(testsupport.TestIdentity2.ID)
	stackId := "stackId"
	ci := createSpaceCodebase("https://github.com/fabric8-services/fabric8-wit.git", &stackId)

	svc, ctrl := rest.SecuredController()
	// Codebase creation is forbidden if the user is not the space owner
	test.CreateSpaceCodebasesForbidden(rest.T(), svc.Context, svc, ctrl, sp.ID, ci)
}

func (rest *TestSpaceCodebaseREST) TestListCodebase() {
	t := rest.T()
	resource.Require(t, resource.Database)

	// Create a new space where we'll create 3 codebase
	sp := rest.createSpace(testsupport.TestIdentity.ID)
	// Create another space where we'll create 1 codebase.

	anotherSpace := rest.createSpace(testsupport.TestIdentity.ID)

	repo := "https://github.com/fabric8-services/fabric8-wit.git"

	svc, ctrl := rest.SecuredController()
	spaceId := sp.ID
	anotherSpaceId := anotherSpace.ID
	var createdSpacesUuids1 []uuid.UUID

	for i := 0; i < 3; i++ {
		repoURL := strings.Replace(repo, "wit", "wit"+strconv.Itoa(i), -1)
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
	stackId := "stackId"

	ci := createSpaceCodebase("https://github.com/fabric8io/fabric8-planner.git", &stackId)

	svc, ctrl := rest.SecuredController()
	test.CreateSpaceCodebasesNotFound(t, svc.Context, svc, ctrl, uuid.NewV4(), ci)
}

func (rest *TestSpaceCodebaseREST) TestFailCreateCodebaseNotAuthorized() {
	t := rest.T()
	resource.Require(t, resource.Database)
	stackId := "stackId"

	ci := createSpaceCodebase("https://github.com/fabric8io/fabric8-planner.git", &stackId)

	svc, ctrl := rest.UnSecuredController()
	test.CreateSpaceCodebasesUnauthorized(t, svc.Context, svc, ctrl, uuid.NewV4(), ci)
}

func (rest *TestSpaceCodebaseREST) TestFailCreateCodebaseExistingCodebase() {
	t := rest.T()
	resource.Require(t, resource.Database)

	// Create first codebase
	fxt := tf.NewTestFixture(t, rest.DB, tf.Codebases(1))

	svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])

	// add another codebase with same url, should fail
	spaceCodebaseContext := createSpaceCodebase(fxt.Codebases[0].URL, fxt.Codebases[0].StackID)
	_, err := test.CreateSpaceCodebasesConflict(t, svc.Context, svc, ctrl,
		fxt.Codebases[0].SpaceID, spaceCodebaseContext)
	require.NotNil(t, err)
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

func (rest *TestSpaceCodebaseREST) createSpace(ownerID uuid.UUID) *space.Space {
	var sp *space.Space
	var err error
	err = application.Transactional(rest.GormDB, func(app application.Application) error {
		repo := app.Spaces()
		newSpace := &space.Space{
			Name:            "TestSpaceCodebase " + uuid.NewV4().String(),
			OwnerID:         ownerID,
			SpaceTemplateID: spacetemplate.SystemLegacyTemplateID,
		}
		sp, err = repo.Create(context.Background(), newSpace)
		return err
	})
	require.NoError(rest.T(), err)
	return sp
}
