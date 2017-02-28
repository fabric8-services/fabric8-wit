package controller

import (
	"fmt"
	"testing"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/test/resource"
	"github.com/almighty/almighty-core/token"
	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
)

var loginTestConfiguration *config.ConfigurationData

func init() {
	var err error
	loginTestConfiguration, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

func newTestKeycloakOAuthProvider() *login.KeycloakOAuthProvider {

	oauth := &oauth2.Config{
		ClientID:     loginTestConfiguration.GetKeycloakClientID(),
		ClientSecret: loginTestConfiguration.GetKeycloakSecret(),
		Scopes:       []string{"user:email"},
		Endpoint:     oauth2.Endpoint{},
	}

	publicKey, err := token.ParsePublicKey([]byte(token.RSAPublicKey))
	if err != nil {
		panic(err)
	}

	tokenManager := token.NewManager(publicKey)
	userRepository := account.NewUserRepository(DB)
	identityRepository := account.NewIdentityRepository(DB)
	app := gormapplication.NewGormDB(DB)
	return login.NewKeycloakOAuthProvider(oauth, identityRepository, userRepository, tokenManager, app)
}

func TestAuthorizeLoginOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	controller := LoginController{auth: TestLoginService{}, configuration: loginTestConfiguration}
	test.AuthorizeLoginTemporaryRedirect(t, nil, nil, &controller)
}

func createControler(t *testing.T) (*goa.Service, *LoginController) {
	svc := goa.New("test")
	loginService := newTestKeycloakOAuthProvider()

	controller := NewLoginController(svc, loginService, loginService.TokenManager, loginTestConfiguration)
	// assert.NotNil(t, controller)
	return svc, controller
}

func TestTestUserTokenObtainedFromKeycloakOK(t *testing.T) {
	resource.Require(t, resource.Database)
	service, controller := createControler(t)
	_, result := test.GenerateLoginOK(t, nil, service, controller)
	assert.Len(t, result, 1, "The size of token array is not 1")
	for _, data := range result {
		validateToken(t, data, controller)
	}
}

func TestRefreshTokenUsingValidRefreshTokenOK(t *testing.T) {
	resource.Require(t, resource.Database)
	service, controller := createControler(t)
	_, result := test.GenerateLoginOK(t, nil, service, controller)
	if len(result) != 1 || result[0].Token.RefreshToken == nil {
		t.Fatal("Can't get the test user token")
	}
	refreshToken := result[0].Token.RefreshToken

	payload := &app.RefreshToken{RefreshToken: refreshToken}
	_, newToken := test.RefreshLoginOK(t, nil, service, controller, payload)
	validateToken(t, newToken, controller)
}

func TestRefreshTokenUsingNilTokenFails(t *testing.T) {
	resource.Require(t, resource.Database)
	service, controller := createControler(t)

	payload := &app.RefreshToken{}
	_, err := test.RefreshLoginBadRequest(t, nil, service, controller, payload)
	assert.NotNil(t, err)
}

func TestRefreshTokenUsingInvalidTokenFails(t *testing.T) {
	resource.Require(t, resource.Database)
	service, controller := createControler(t)

	refreshToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.S-vR8LZTQ92iqGCR3rNUG0MiGx2N5EBVq0frCHP_bJ8"
	payload := &app.RefreshToken{RefreshToken: &refreshToken}
	_, err := test.RefreshLoginBadRequest(t, nil, service, controller, payload)
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
