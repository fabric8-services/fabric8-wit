package main

import (
	"testing"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
)

func newTestKeycloakOAuthProvider() *login.KeycloakOAuthProvider {
	oauth := &oauth2.Config{
		ClientID:     configuration.GetKeycloakClientID(),
		ClientSecret: configuration.GetKeycloakSecret(),
		Scopes:       []string{"user:email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://sso.demo.almighty.io/auth/realms/demo/protocol/openid-connect/auth",
			TokenURL: "http://sso.demo.almighty.io/auth/realms/demo/protocol/openid-connect/token",
		},
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
	controller := LoginController{auth: TestLoginService{}}
	test.AuthorizeLoginTemporaryRedirect(t, nil, nil, &controller)
}

func createControler(t *testing.T) (*goa.Service, *LoginController) {
	svc := goa.New("test")
	loginService := newTestKeycloakOAuthProvider()
	controller := NewLoginController(svc, loginService, loginService.TokenManager)
	// assert.NotNil(t, controller)
	return svc, controller
}

func TestTestUserTokenObtainedFromKeycloakOK(t *testing.T) {
	resource.Require(t, resource.Database)
	service, controller := createControler(t)
	_, result := test.GenerateLoginOK(t, nil, service, controller)
	assert.Len(t, result, 1, "The size of token array is not 1")
	for _, data := range result {
		assert.NotEmpty(t, data.Token.AccessToken, "Access token is empty")
		assert.NotEmpty(t, data.Token.RefreshToken, "Refresh token is empty")
		assert.NotEmpty(t, data.Token.TokenType, "Token type is empty")
		assert.NotNil(t, data.Token.ExpiresIn, "Expires-in is nil")
		assert.NotNil(t, data.Token.RefreshExpiresIn, "Refresh-expires-in is nil")
		assert.NotNil(t, data.Token.NotBeforePolicy, "Not-before-policy in is nil")
	}
}

type TestLoginService struct{}

func (t TestLoginService) Perform(ctx *app.AuthorizeLoginContext) error {
	return ctx.TemporaryRedirect()
}

func (t TestLoginService) CreateKeycloakUser(accessToken string, ctx context.Context) (*account.Identity, *account.User, error) {
	return nil, nil, nil
}
