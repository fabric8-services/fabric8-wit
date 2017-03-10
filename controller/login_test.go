package controller

import (
	"testing"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/token"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestLoginREST struct {
	gormtestsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunLoginREST(t *testing.T) {
	suite.Run(t, &TestLoginREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestLoginREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestLoginREST) TearDownTest() {
	rest.clean()
}

func (rest *TestLoginREST) UnSecuredController() (*goa.Service, *LoginController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Login-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, &LoginController{Controller: svc.NewController("login"), auth: TestLoginService{}, configuration: rest.Configuration}
}

func (rest *TestLoginREST) SecuredController() (*goa.Service, *LoginController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	loginService := newTestKeycloakOAuthProvider(rest.db, rest.Configuration)

	svc := testsupport.ServiceAsUser("Login-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewLoginController(svc, loginService, loginService.TokenManager, rest.Configuration)
}

func newTestKeycloakOAuthProvider(db application.DB, configuration loginConfiguration) *login.KeycloakOAuthProvider {

	oauth := &oauth2.Config{
		ClientID:     configuration.GetKeycloakClientID(),
		ClientSecret: configuration.GetKeycloakSecret(),
		Scopes:       []string{"user:email"},
		Endpoint:     oauth2.Endpoint{},
	}

	publicKey, err := token.ParsePublicKey([]byte(token.RSAPublicKey))
	if err != nil {
		panic(err)
	}

	tokenManager := token.NewManager(publicKey)
	return login.NewKeycloakOAuthProvider(oauth, db.Identities(), db.Users(), tokenManager, db)
}

func (rest *TestLoginREST) TestAuthorizeLoginOK() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	svc, ctrl := rest.UnSecuredController()

	test.AuthorizeLoginTemporaryRedirect(t, svc.Context, svc, ctrl)
}

func (rest *TestLoginREST) TestTestUserTokenObtainedFromKeycloakOK() {
	t := rest.T()
	resource.Require(t, resource.Database)
	service, controller := rest.SecuredController()
	_, result := test.GenerateLoginOK(t, service.Context, service, controller)
	assert.Len(t, result, 1, "The size of token array is not 1")
	for _, data := range result {
		validateToken(t, data, controller)
	}
}

func (rest *TestLoginREST) TestRefreshTokenUsingValidRefreshTokenOK() {
	t := rest.T()
	resource.Require(t, resource.Database)
	service, controller := rest.SecuredController()
	_, result := test.GenerateLoginOK(t, service.Context, service, controller)
	if len(result) != 1 || result[0].Token.RefreshToken == nil {
		t.Fatal("Can't get the test user token")
	}
	refreshToken := result[0].Token.RefreshToken

	payload := &app.RefreshToken{RefreshToken: refreshToken}
	_, newToken := test.RefreshLoginOK(t, service.Context, service, controller, payload)
	validateToken(t, newToken, controller)
}

func (rest *TestLoginREST) TestRefreshTokenUsingNilTokenFails() {
	t := rest.T()
	resource.Require(t, resource.Database)
	service, controller := rest.SecuredController()

	payload := &app.RefreshToken{}
	_, err := test.RefreshLoginBadRequest(t, service.Context, service, controller, payload)
	assert.NotNil(t, err)
}

func (rest *TestLoginREST) TestRefreshTokenUsingInvalidTokenFails() {
	t := rest.T()
	resource.Require(t, resource.Database)
	service, controller := rest.SecuredController()

	refreshToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.S-vR8LZTQ92iqGCR3rNUG0MiGx2N5EBVq0frCHP_bJ8"
	payload := &app.RefreshToken{RefreshToken: &refreshToken}
	_, err := test.RefreshLoginBadRequest(t, service.Context, service, controller, payload)
	assert.NotNil(t, err)
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

func (t TestLoginService) Perform(ctx *app.AuthorizeLoginContext, authEndpoint string, tokenEndpoint string) error {
	return ctx.TemporaryRedirect()
}

func (t TestLoginService) CreateKeycloakUser(accessToken string, ctx context.Context) (*account.Identity, *account.User, error) {
	return nil, nil, nil
}
