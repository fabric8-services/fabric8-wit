package controller

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"context"

	"golang.org/x/oauth2"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"

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
	"github.com/fabric8-services/fabric8-wit/token"
	almtoken "github.com/fabric8-services/fabric8-wit/token"

	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestLoginREST struct {
	gormtestsupport.DBTestSuite
	configuration *configuration.ConfigurationData
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
	c, err := configuration.GetConfigurationData()
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
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	identityRepository := account.NewIdentityRepository(rest.DB)
	svc := testsupport.ServiceAsUser("Login-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, &LoginController{Controller: svc.NewController("login"), auth: TestLoginService{}, configuration: rest.Configuration, identityRepository: identityRepository}
}

func (rest *TestLoginREST) SecuredController() (*goa.Service, *LoginController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	identityRepository := account.NewIdentityRepository(rest.DB)

	svc := testsupport.ServiceAsUser("Login-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewLoginController(svc, rest.loginService, rest.loginService.TokenManager, rest.Configuration, identityRepository)
}

func (rest *TestLoginREST) newTestKeycloakOAuthProvider(db application.DB) *login.KeycloakOAuthProvider {
	publicKey, err := token.ParsePublicKey([]byte(rest.configuration.GetTokenPublicKey()))
	require.Nil(rest.T(), err)
	tokenManager := token.NewManager(publicKey)
	return login.NewKeycloakOAuthProvider(db.Identities(), db.Users(), tokenManager, db)
}

func (rest *TestLoginREST) TestAuthorizeLoginOK() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	svc, ctrl := rest.UnSecuredController()

	test.AuthorizeLoginTemporaryRedirect(t, svc.Context, svc, ctrl, nil, nil)
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

func (rest *TestLoginREST) TestRefreshTokenUsingValidRefreshTokenOK() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	service, controller := rest.SecuredController()
	_, result := test.GenerateLoginOK(t, service.Context, service, controller)
	if len(result) != 2 || result[0].Token.RefreshToken == nil {
		t.Fatal("Can't get the test user token")
	}
	refreshToken := result[0].Token.RefreshToken

	payload := &app.RefreshToken{RefreshToken: refreshToken}
	resp, newToken := test.RefreshLoginOK(t, service.Context, service, controller, payload)

	assert.Equal(t, resp.Header().Get("Cache-Control"), "no-cache")
	validateToken(t, newToken, controller)
}

func (rest *TestLoginREST) TestRefreshTokenUsingNilTokenFails() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	service, controller := rest.SecuredController()

	payload := &app.RefreshToken{}
	_, err := test.RefreshLoginBadRequest(t, service.Context, service, controller, payload)
	assert.NotNil(t, err)
}

func (rest *TestLoginREST) TestRefreshTokenUsingInvalidTokenFails() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	service, controller := rest.SecuredController()

	refreshToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.S-vR8LZTQ92iqGCR3rNUG0MiGx2N5EBVq0frCHP_bJ8"
	payload := &app.RefreshToken{RefreshToken: &refreshToken}
	_, err := test.RefreshLoginBadRequest(t, service.Context, service, controller, payload)
	assert.NotNil(t, err)
}

func (rest *TestLoginREST) TestLinkIdPWithoutTokenFails() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	service, controller := rest.SecuredController()

	resp, err := test.LinkLoginUnauthorized(t, service.Context, service, controller, nil, nil)
	assert.NotNil(t, err)
	assert.Equal(t, resp.Header().Get("Cache-Control"), "no-cache")
}

func (rest *TestLoginREST) TestLinkIdPWithTokenRedirects() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	svc, ctrl := rest.UnSecuredController()

	test.LinkLoginTemporaryRedirect(t, svc.Context, svc, ctrl, nil, nil)
}

func validateToken(t *testing.T, token *app.AuthToken, controler *LoginController) {
	assert.NotNil(t, token, "Token data is nil")
	assert.NotEmpty(t, token.Token.AccessToken, "Access token is empty")
	assert.NotEmpty(t, token.Token.RefreshToken, "Refresh token is empty")
	assert.NotEmpty(t, token.Token.TokenType, "Token type is empty")
	assert.NotNil(t, token.Token.ExpiresIn, "Expires-in is nil")
	assert.NotNil(t, token.Token.RefreshExpiresIn, "Refresh-expires-in is nil")
	assert.NotNil(t, token.Token.NotBeforePolicy, "Not-before-policy is nil")
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		return controler.tokenManager.PublicKey(), nil
	}
	_, err := jwt.Parse(*token.Token.AccessToken, keyFunc)
	assert.Nil(t, err, "Invalid access token")
	_, err = jwt.Parse(*token.Token.RefreshToken, keyFunc)
	assert.Nil(t, err, "Invalid refresh token")
}

type TestLoginService struct{}

func (t TestLoginService) Perform(ctx *app.AuthorizeLoginContext, oauth *oauth2.Config, brokerEndpoint string, entitlementEndpoint string, profileEndpoint string, validRedirectURL string, userNotApprovedRedirectURL string) error {
	return ctx.TemporaryRedirect()
}

func (t TestLoginService) CreateOrUpdateKeycloakUser(accessToken string, ctx context.Context, profileEndpoint string) (*account.Identity, *account.User, error) {
	return nil, nil, nil
}

func (t TestLoginService) Link(ctx *app.LinkLoginContext, brokerEndpoint string, clientID string, validRedirectURL string) error {
	return ctx.TemporaryRedirect()
}

func (t TestLoginService) LinkSession(ctx *app.LinksessionLoginContext, brokerEndpoint string, clientID string, validRedirectURL string) error {
	return ctx.TemporaryRedirect()
}

func (t TestLoginService) LinkCallback(ctx *app.LinkcallbackLoginContext, brokerEndpoint string, clientID string) error {
	return ctx.TemporaryRedirect()
}

/*  For recent spaces test */

type TestRecentSpacesREST struct {
	gormtestsupport.RemoteTestSuite
	configuration      *configuration.ConfigurationData
	tokenManager       token.Manager
	identityRepository *MockIdentityRepository
	userRepository     *MockUserRepository
	loginService       *TestLoginService

	clean func()
}

func TestRunRecentSpacesREST(t *testing.T) {
	suite.Run(t, &TestRecentSpacesREST{RemoteTestSuite: gormtestsupport.NewRemoteTestSuite("../config.yaml")})
}

func (rest *TestRecentSpacesREST) newTestKeycloakOAuthProvider(db application.DB) *login.KeycloakOAuthProvider {
	publicKey, err := token.ParsePublicKey([]byte(rest.configuration.GetTokenPublicKey()))
	require.Nil(rest.T(), err)
	tokenManager := token.NewManager(publicKey)

	return login.NewKeycloakOAuthProvider(rest.identityRepository, rest.userRepository, tokenManager, db)
}

func (rest *TestRecentSpacesREST) SetupTest() {
	c, err := configuration.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	rest.configuration = c
	publicKey, err := token.ParsePublicKey([]byte(rest.configuration.GetTokenPublicKey()))
	require.Nil(rest.T(), err)
	rest.tokenManager = token.NewManager(publicKey)

	identity := account.Identity{}
	user := account.User{}
	identity.User = user

	rest.loginService = &TestLoginService{}

	rest.identityRepository = &MockIdentityRepository{testIdentity: &identity}
	rest.userRepository = &MockUserRepository{}

}

/* MockUserRepositoryService */

type MockIdentityRepository struct {
	testIdentity *account.Identity
}

func (rest *TestRecentSpacesREST) SecuredController() (*goa.Service, *LoginController) {
	svc := testsupport.ServiceAsUser("Login-Service", rest.tokenManager, testsupport.TestIdentity)
	loginController := &LoginController{
		Controller:         svc.NewController("login"),
		auth:               rest.loginService,
		tokenManager:       rest.tokenManager,
		configuration:      rest.configuration,
		identityRepository: rest.identityRepository,
	}
	return svc, loginController
}

func (rest *TestRecentSpacesREST) TestResourceRequestPayload() {
	t := rest.T()
	resource.Require(t, resource.Remote)
	service, controller := rest.SecuredController()

	// Generate an access token for a test identity
	r := &goa.RequestData{
		Request: &http.Request{Host: "api.example.org"},
	}
	tokenEndpoint, err := rest.configuration.GetKeycloakEndpointToken(r)
	require.Nil(t, err)

	accessToken, err := GenerateUserToken(service.Context, tokenEndpoint, rest.configuration, rest.configuration.GetKeycloakTestUserName(), rest.configuration.GetKeycloakTestUserSecret())
	require.Nil(t, err)

	accessTokenString := accessToken.Token.AccessToken

	require.Nil(t, err)
	require.NotNil(t, accessTokenString)

	require.Nil(t, err)

	// Scenario 1 - Test user has a nil contextInformation, hence there are no recent spaces to
	// add to the resource object

	rest.identityRepository.testIdentity.User.ContextInformation = nil
	resource, err := controller.getEntitlementResourceRequestPayload(service.Context, accessTokenString)
	require.Nil(t, err)

	// This will be nil because contextInformation for the test user is empty!
	require.Nil(t, resource)

	// Scenario 2 - Test user has 'some' contextInformation
	identity := account.Identity{}
	user := account.User{
		ContextInformation: account.ContextInformation{
			"recentSpaces": []interface{}{"29dd4613-3da1-4100-a2d6-414573eaa470"},
		},
	}
	identity.User = user
	rest.identityRepository.testIdentity = &identity

	//Use the same access token to retrieve
	resource, err = controller.getEntitlementResourceRequestPayload(service.Context, accessTokenString)
	require.Nil(t, err)

	require.NotNil(t, resource)
	require.NotNil(t, resource.Permissions)
	assert.Len(t, resource.Permissions, 1)

}

// Load returns a single Identity as a Database Model
// This is more for use internally, and probably not what you want in  your controllers
func (m *MockIdentityRepository) Load(ctx context.Context, id uuid.UUID) (*account.Identity, error) {
	return m.testIdentity, nil
}

// Exists returns true|false whether an identity exists with a specific identifier
func (m *MockIdentityRepository) Exists(ctx context.Context, id string) (bool, error) {
	return true, nil
}

// Create creates a new record.
func (m *MockIdentityRepository) Create(ctx context.Context, model *account.Identity) error {
	return nil
}

// Lookup looks for an existing identity with the given `profileURL` or creates a new one
func (m *MockIdentityRepository) Lookup(ctx context.Context, username, profileURL, providerType string) (*account.Identity, error) {
	return m.testIdentity, nil
}

// Save modifies a single record.
func (m *MockIdentityRepository) Save(ctx context.Context, model *account.Identity) error {
	m.testIdentity = model
	return nil
}

// Delete removes a single record.
func (m *MockIdentityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

// Query expose an open ended Query model
func (m *MockIdentityRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]account.Identity, error) {
	var identities []account.Identity
	identities = append(identities, *m.testIdentity)
	return identities, nil
}

// First returns the first Identity element that matches the given criteria
func (m *MockIdentityRepository) First(funcs ...func(*gorm.DB) *gorm.DB) (*account.Identity, error) {
	return m.testIdentity, nil
}

func (m *MockIdentityRepository) List(ctx context.Context) ([]account.Identity, error) {
	var rows []account.Identity
	rows = append(rows, *m.testIdentity)
	return rows, nil
}

func (m *MockIdentityRepository) CheckExists(ctx context.Context, id string) error {
	return nil
}

func (m *MockIdentityRepository) IsValid(ctx context.Context, id uuid.UUID) bool {
	return true
}

func (m *MockIdentityRepository) Search(ctx context.Context, q string, start int, limit int) ([]account.Identity, int, error) {
	result := []account.Identity{}
	result = append(result, *m.testIdentity)
	return result, 1, nil
}

type MockUserRepository struct {
	User *account.User
}

func (m MockUserRepository) Load(ctx context.Context, id uuid.UUID) (*account.User, error) {
	if m.User == nil {
		return nil, errors.New("not found")
	}
	return m.User, nil
}

func (m MockUserRepository) Exists(ctx context.Context, id string) (bool, error) {
	if m.User == nil {
		return false, errors.New("not found")
	}
	return true, nil
}

// Create creates a new record.
func (m MockUserRepository) Create(ctx context.Context, u *account.User) error {
	m.User = u
	return nil
}

// Save modifies a single record
func (m MockUserRepository) Save(ctx context.Context, model *account.User) error {
	return m.Create(ctx, model)
}

// Save modifies a single record
func (m MockUserRepository) CheckExists(ctx context.Context, id string) error {
	return nil
}

// Delete removes a single record.
func (m MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.User = nil
	return nil
}

// List return all users
func (m MockUserRepository) List(ctx context.Context) ([]account.User, error) {
	return []account.User{*m.User}, nil
}

// Query expose an open ended Query model
func (m MockUserRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]account.User, error) {
	return []account.User{*m.User}, nil
}
