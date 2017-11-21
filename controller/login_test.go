package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	testtoken "github.com/fabric8-services/fabric8-wit/test/token"

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestLoginREST struct {
	gormtestsupport.DBTestSuite
	configuration *configuration.Registry
	loginService  *login.KeycloakOAuthProvider
	db            *gormapplication.GormDB
	clean         func()
}

func TestRunLoginREST(t *testing.T) {
	suite.Run(t, &TestLoginREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestLoginREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
	c, err := configuration.Get()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	rest.configuration = c
	rest.loginService = rest.newTestKeycloakOAuthProvider(rest.db)
}

func (rest *TestLoginREST) TearDownTest() {
	rest.clean()
}

func (rest *TestLoginREST) UnSecuredController() (*goa.Service, *LoginController) {
	identityRepository := account.NewIdentityRepository(rest.DB)
	svc := testsupport.ServiceAsUser("Login-Service", testsupport.TestIdentity)
	return svc, &LoginController{Controller: svc.NewController("login"), auth: TestLoginService{}, configuration: rest.Configuration, identityRepository: identityRepository}
}

func (rest *TestLoginREST) SecuredController() (*goa.Service, *LoginController) {
	identityRepository := account.NewIdentityRepository(rest.DB)

	svc := testsupport.ServiceAsUser("Login-Service", testsupport.TestIdentity)
	return svc, NewLoginController(svc, rest.loginService, rest.Configuration, identityRepository)
}

func (rest *TestLoginREST) newTestKeycloakOAuthProvider(db application.DB) *login.KeycloakOAuthProvider {
	return login.NewKeycloakOAuthProvider(db.Identities(), db.Users(), testtoken.TokenManager, db)
}

func (rest *TestLoginREST) TestAuthorizeLoginRedirected() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	svc, ctrl := rest.UnSecuredController()

	test.AuthorizeLoginTemporaryRedirect(t, svc.Context, svc, ctrl)
}

func (rest *TestLoginREST) TestTestUserTokenObtainedFromKeycloakOK() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	service, controller := rest.SecuredController()
	resp, result := test.GenerateLoginOK(t, service.Context, service, controller)

	assert.Equal(t, resp.Header().Get("Cache-Control"), "no-cache")
	assert.Len(t, result, 2, "The size of token array is not 2")
	for _, data := range result {
		validateToken(t, data, controller)
	}
}

func (rest *TestLoginREST) TestLinkRedirected() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	svc, ctrl := rest.UnSecuredController()

	test.LinkLoginTemporaryRedirect(t, svc.Context, svc, ctrl)
}

func validateToken(t *testing.T, token *app.AuthToken, controler *LoginController) {
	assert.NotNil(t, token, "Token data is nil")
	assert.NotEmpty(t, token.Token.AccessToken, "Access token is empty")
	assert.NotEmpty(t, token.Token.RefreshToken, "Refresh token is empty")
	assert.NotEmpty(t, token.Token.TokenType, "Token type is empty")
	assert.NotNil(t, token.Token.ExpiresIn, "Expires-in is nil")
	assert.NotNil(t, token.Token.RefreshExpiresIn, "Refresh-expires-in is nil")
	assert.NotNil(t, token.Token.NotBeforePolicy, "Not-before-policy is nil")
}

type TestLoginService struct{}

func (t TestLoginService) CreateOrUpdateKeycloakUser(accessToken string, ctx context.Context) (*account.Identity, *account.User, error) {
	return nil, nil, nil
}
