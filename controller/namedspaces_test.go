package controller_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestNamedSpaceREST struct {
	gormtestsupport.DBTestSuite
	testDir string
}

func TestRunNamedSpacesREST(t *testing.T) {
	suite.Run(t, &TestNamedSpaceREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (rest *TestNamedSpaceREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
	rest.testDir = filepath.Join("test-files", "namedspaces")
}

func (rest *TestNamedSpaceREST) SecuredNamedSpaceController(identity account.Identity) (*goa.Service, *NamedspacesController) {
	svc := testsupport.ServiceAsUser("NamedSpace-Service", identity)
	return svc, NewNamedspacesController(svc, rest.GormDB)
}

func (rest *TestNamedSpaceREST) UnSecuredNamedSpaceController() (*goa.Service, *NamedspacesController) {
	svc := goa.New("NamedSpace-Service")
	return svc, NewNamedspacesController(svc, rest.GormDB)
}

func (rest *TestNamedSpaceREST) SecuredSpaceController() (*goa.Service, *SpaceController) {
	svc := testsupport.ServiceAsUser("Space-Service", testsupport.TestIdentity)
	return svc, NewSpaceController(svc, rest.GormDB, rest.Configuration, &DummyResourceManager{})
}

func (rest *TestNamedSpaceREST) UnSecuredSpaceController() (*goa.Service, *SpaceController) {
	svc := goa.New("Space-Service")
	return svc, NewSpaceController(svc, rest.GormDB, rest.Configuration, &DummyResourceManager{})
}

func (rest *TestNamedSpaceREST) TestSuccessQuerySpace() {
	t := rest.T()
	resource.Require(t, resource.Database)

	spaceSvc, spaceCtrl := rest.SecuredSpaceController()

	identityRepo := account.NewIdentityRepository(rest.DB)
	identity := testsupport.TestIdentity
	identity.ProviderType = account.KeycloakIDP
	err := identityRepo.Create(spaceSvc.Context, &identity)
	if err != nil {
		assert.Fail(t, "Failed to create an identity")
	}

	name := testsupport.CreateRandomValidTestName("Test 24")

	p := newCreateSpacePayload(&name, nil)

	_, created := test.CreateSpaceCreated(t, spaceSvc.Context, spaceSvc, spaceCtrl, p)
	assert.NotNil(t, created.Data)
	assert.NotNil(t, created.Data.Attributes)
	assert.NotNil(t, created.Data.Attributes.CreatedAt)
	assert.NotNil(t, created.Data.Attributes.UpdatedAt)
	assert.NotNil(t, created.Data.Attributes.Name)
	assert.Equal(t, name, *created.Data.Attributes.Name)
	assert.NotNil(t, created.Data.Links)
	assert.NotNil(t, created.Data.Links.Self)

	namedSpaceSvc, namedSpacectrl := rest.SecuredNamedSpaceController(testsupport.TestIdentity)
	_, namedspace := test.ShowNamedspacesOK(t, namedSpaceSvc.Context, namedSpaceSvc, namedSpacectrl, testsupport.TestIdentity.Username, name)
	assert.NotNil(t, namedspace)
	assert.Equal(t, created.Data.Attributes.Name, namedspace.Data.Attributes.Name)
	assert.Equal(t, created.Data.Attributes.Description, namedspace.Data.Attributes.Description)
	assert.Equal(t, created.Data.Links.Self, namedspace.Data.Links.Self)

	// test that show namedspaces operation is not case sensitive for the space name
	_, namedspace = test.ShowNamedspacesOK(t, namedSpaceSvc.Context, namedSpaceSvc, namedSpacectrl, testsupport.TestIdentity.Username, strings.ToLower(name))
	assert.NotNil(t, namedspace)
	assert.Equal(t, created.Data.Attributes.Name, namedspace.Data.Attributes.Name)
}

func (rest *TestNamedSpaceREST) TestSuccessListSpaces() {
	t := rest.T()
	resource.Require(t, resource.Database)

	spaceSvc, spaceCtrl := rest.SecuredSpaceController()

	identityRepo := account.NewIdentityRepository(rest.DB)
	identity := testsupport.TestIdentity
	identity.ProviderType = account.KeycloakIDP
	err := identityRepo.Create(spaceSvc.Context, &identity)
	if err != nil {
		assert.Fail(t, "Failed to create an identity")
	}

	name := testsupport.CreateRandomValidTestName("Test 24")

	p := newCreateSpacePayload(&name, nil)

	_, created := test.CreateSpaceCreated(t, spaceSvc.Context, spaceSvc, spaceCtrl, p)
	assert.NotNil(t, created.Data)
	assert.NotNil(t, created.Data.Attributes)
	assert.NotNil(t, created.Data.Attributes.CreatedAt)
	assert.NotNil(t, created.Data.Attributes.UpdatedAt)
	assert.NotNil(t, created.Data.Attributes.Name)
	assert.Equal(t, name, *created.Data.Attributes.Name)
	assert.NotNil(t, created.Data.Links)
	assert.NotNil(t, created.Data.Links.Self)

	collabSpaceSvc, collabSpacectrl := rest.SecuredNamedSpaceController(testsupport.TestIdentity)
	_, collabspaces := test.ListNamedspacesOK(t, collabSpaceSvc.Context, collabSpaceSvc, collabSpacectrl, testsupport.TestIdentity.Username, nil, nil)
	assert.True(t, len(collabspaces.Data) > 0)
	assert.Equal(t, created.Data.Attributes.Name, collabspaces.Data[0].Attributes.Name)
	assert.Equal(t, created.Data.Attributes.Description, collabspaces.Data[0].Attributes.Description)
	assert.Equal(t, created.Data.Links.Self, collabspaces.Data[0].Links.Self)
}

func (rest *TestNamedSpaceREST) TestShow() {
	rest.T().Run("ok", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment())

		namedSpaceSvc, namedSpacectrl := rest.SecuredNamedSpaceController(*fxt.Identities[0])
		res, namedspace := test.ShowNamedspacesOK(t, namedSpaceSvc.Context, namedSpaceSvc, namedSpacectrl, fxt.Identities[0].Username, fxt.Spaces[0].Name)

		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok.payload.golden.json"), namedspace)
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok.headers.golden.json"), res.Header())

		assert.NotNil(t, namedspace)
		assert.Equal(t, &fxt.Spaces[0].Name, namedspace.Data.Attributes.Name)
		assert.Equal(t, &fxt.Spaces[0].Description, namedspace.Data.Attributes.Description)
	})
}
