package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"context"

	"golang.org/x/oauth2"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/workitem"
	errs "github.com/pkg/errors"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/errors"
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
	configuration   *configuration.ConfigurationData
	loginService    *login.KeycloakOAuthProvider
	userRespository account.UserRepository
	db              *gormapplication.GormDB
	clean           func()
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
	rest.userRespository = account.NewUserRepository(rest.DB)
}

func (rest *TestLoginREST) TearDownTest() {
	rest.clean()
}

func (rest *TestLoginREST) UnSecuredController() (*goa.Service, *LoginController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	identityRepository := account.NewIdentityRepository(rest.DB)
	svc := testsupport.ServiceAsUser("Login-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, &LoginController{Controller: svc.NewController("login"), auth: TestLoginService{}, configuration: rest.Configuration, identities: identityRepository}
}

func (rest *TestLoginREST) SecuredController() (*goa.Service, *LoginController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	identityRepository := account.NewIdentityRepository(rest.DB)

	svc := testsupport.ServiceAsUser("Login-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewLoginController(svc, rest.loginService, rest.loginService.TokenManager, rest.Configuration, identityRepository)
}

func (rest *TestLoginREST) newTestKeycloakOAuthProvider(db application.DB) *login.KeycloakOAuthProvider {
	publicKey, err := token.ParsePublicKey([]byte(rest.configuration.GetTokenPublicKey()))
	//token.ParsePublicKey([]byte(token.RSAPublicKey))
	require.Nil(rest.T(), err)
	tokenManager := token.NewManager(publicKey)
	identityRepository := account.NewIdentityRepository(rest.DB)
	return login.NewKeycloakOAuthProvider(identityRepository, rest.userRespository, tokenManager, db)
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

func (rest *TestLoginREST) TestResourceRequestPayload() {
	t := rest.T()
	resource.Require(t, resource.Database)
	service, controller := rest.SecuredController()

	// Generate an access token for a test identity
	accessTokenString, err := rest.generateAccessToken()
	require.Nil(t, err)
	require.NotNil(t, accessTokenString)

	// Update DB
	r := &goa.RequestData{
		Request: &http.Request{Host: "demo.api.openshift.io"},
	}
	profileEndpoint, err := rest.configuration.GetKeycloakAccountEndpoint(r)
	require.Nil(t, err)
	identity, user, err := rest.loginService.CreateOrUpdateKeycloakUser(*accessTokenString, service.Context, profileEndpoint)
	require.Nil(t, err)
	require.NotNil(t, identity)
	require.NotNil(t, user)

	// Scenario 1 - Test user has a nil contextInformation, hence there are no recent spaces to
	// add to the resource object

	user.ContextInformation = nil
	err = rest.userRespository.Save(service.Context, user)
	require.Nil(t, err)

	//Use the same access token to retrieve
	resource, err := controller.getEntitlementResourceRequestPayload(service.Context, accessTokenString)
	require.Nil(t, err)

	// This will be nil because contextInformation for the test user is empty!
	require.Nil(t, resource)

	// Scenario 2 - Test user has 'some' contextInformation
	user.ContextInformation = workitem.Fields{
		"recentSpaces": []string{"29dd4613-3da1-4100-a2d6-414573eaa470"},
	}
	err = rest.userRespository.Save(service.Context, user)
	require.Nil(t, err)

	//Use the same access token to retrieve
	resource, err = controller.getEntitlementResourceRequestPayload(service.Context, accessTokenString)
	require.Nil(t, err)

	require.NotNil(t, resource)
	require.NotNil(t, resource.Permissions)
	assert.Len(t, resource.Permissions, 1)

}

func (rest *TestLoginREST) generateAccessToken() (*string, error) {
	t := rest.T()

	var scopes []account.Identity
	scopes = append(scopes, testsupport.TestIdentity)
	scopes = append(scopes, testsupport.TestObserverIdentity)

	client := &http.Client{Timeout: 10 * time.Second}
	r := &goa.RequestData{
		Request: &http.Request{Host: "api.example.org"},
	}
	tokenEndpoint, err := rest.configuration.GetKeycloakEndpointToken(r)

	res, err := client.PostForm(tokenEndpoint, url.Values{
		"client_id":     {rest.configuration.GetKeycloakClientID()},
		"client_secret": {rest.configuration.GetKeycloakSecret()},
		"username":      {rest.configuration.GetKeycloakTestUserName()},
		"password":      {rest.configuration.GetKeycloakTestUserSecret()},
		"grant_type":    {"password"},
	})
	if err != nil {
		return nil, errors.NewInternalError(context.Background(), errs.Wrap(err, "error when obtaining token"))
	}

	token, err := auth.ReadToken(context.Background(), res)
	require.Nil(t, err)
	return token.AccessToken, err
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
