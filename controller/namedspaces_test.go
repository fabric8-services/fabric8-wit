package controller_test

import (
	"strings"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app/test"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestNamedSpaceREST struct {
	gormtestsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunNamedSpacesREST(t *testing.T) {
	suite.Run(t, &TestNamedSpaceREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestNamedSpaceREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestNamedSpaceREST) TearDownTest() {
	rest.clean()
}

func (rest *TestNamedSpaceREST) SecuredNamedSpaceController(identity account.Identity) (*goa.Service, *NamedspacesController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("NamedSpace-Service", almtoken.NewManagerWithPrivateKey(priv), identity)
	return svc, NewNamedspacesController(svc, rest.db)
}

func (rest *TestNamedSpaceREST) UnSecuredNamedSpaceController() (*goa.Service, *NamedspacesController) {
	svc := goa.New("NamedSpace-Service")
	return svc, NewNamedspacesController(svc, rest.db)
}

func (rest *TestNamedSpaceREST) SecuredSpaceController() (*goa.Service, *SpaceController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Space-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewSpaceController(svc, rest.db, rest.Configuration, &DummyResourceManager{})
}

func (rest *TestNamedSpaceREST) UnSecuredSpaceController() (*goa.Service, *SpaceController) {
	svc := goa.New("Space-Service")
	return svc, NewSpaceController(svc, rest.db, rest.Configuration, &DummyResourceManager{})
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

	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name

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

	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name

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
