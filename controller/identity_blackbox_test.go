package controller_test

import (
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type TestIdentityREST struct {
	gormtestsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunIdentityREST(t *testing.T) {
	suite.Run(t, &TestIdentityREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestIdentityREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestIdentityREST) TearDownTest() {
	rest.clean()
}

func (rest *TestIdentityREST) SecuredController() (*goa.Service, *IdentityController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Status-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewIdentityController(svc, rest.db)
}

func (rest *TestIdentityREST) UnSecuredController() (*goa.Service, *IdentityController) {
	svc := goa.New("Status-Service")
	return svc, NewIdentityController(svc, rest.db)
}

func (rest *TestIdentityREST) TestListIdentities() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.UnSecuredController()
	_, ic := test.ListIdentityOK(t, svc.Context, svc, ctrl)

	numberOfCurrentIdent := len(ic.Data)

	ctx := context.Background()

	identityRepo := rest.db.Identities()

	id := uuid.NewV4()
	identity := account.Identity{
		Username:     "TestUser",
		ProviderType: "test-idp",
		ID:           id,
	}

	err := identityRepo.Create(ctx, &identity)
	if err != nil {
		t.Fatal(err)
	}

	_, ic2 := test.ListIdentityOK(t, svc.Context, svc, ctrl)
	require.NotNil(t, ic2)

	assert.Equal(t, numberOfCurrentIdent+1, len(ic2.Data))

	assertIdent(t, identity, findIdent(identity.ID, ic2.Data))

	id = uuid.NewV4()
	identity2 := account.Identity{
		Username:     "TestUser2",
		ProviderType: "test-idp",
		ID:           id,
	}

	err = identityRepo.Create(ctx, &identity2)
	if err != nil {
		t.Fatal(err)
	}

	_, ic3 := test.ListIdentityOK(t, svc.Context, svc, ctrl)
	require.NotNil(t, ic3)
	assert.Equal(t, numberOfCurrentIdent+2, len(ic3.Data))

	assertIdent(t, identity, findIdent(identity.ID, ic3.Data))
	assertIdent(t, identity2, findIdent(identity2.ID, ic3.Data))
}

func findIdent(id uuid.UUID, idents []*app.IdentityData) *app.IdentityData {
	for _, ident := range idents {
		if *ident.ID == id.String() {
			return ident
		}
	}
	return nil
}

func assertIdent(t *testing.T, expected account.Identity, actual *app.IdentityData) {
	assert.Equal(t, expected.Username, *actual.Attributes.Username)
	assert.Equal(t, expected.ProviderType, *actual.Attributes.ProviderType)
}
