package login

import (
	"fmt"
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/token"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

var loginService *keycloakOAuthProvider

func setup() {

	configuration, err := config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	oauth := &oauth2.Config{
		ClientID:     configuration.GetKeycloakClientID(),
		ClientSecret: configuration.GetKeycloakSecret(),
		Scopes:       []string{"user:email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://sso.demo.almighty.io/auth/realms/demo/protocol/openid-connect/auth",
			TokenURL: "http://sso.demo.almighty.io/auth/realms/demo/protocol/openid-connect/token",
		},
	}

	publicKey, err := token.ParsePublicKey(configuration.GetTokenPublicKey())
	if err != nil {
		panic(err)
	}

	privateKey, err := token.ParsePrivateKey(configuration.GetTokenPrivateKey())
	if err != nil {
		panic(err)
	}

	tokenManager := token.NewManager(publicKey, privateKey)
	userRepository := account.NewUserRepository(nil)
	identityRepository := account.NewIdentityRepository(nil)
	loginService = &keycloakOAuthProvider{
		config:       oauth,
		identities:   identityRepository,
		users:        userRepository,
		tokenManager: tokenManager,
	}
}

func tearDown() {
	loginService = nil
}

func TestValidOAuthAccessToken(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	t.Skip("Not implemented")
}

func TestInvalidOAuthAccessToken(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	setup()
	defer tearDown()

	invalidAccessToken := "7423742yuuiy-INVALID-73842342389h"

	accessToken := &oauth2.Token{
		AccessToken: invalidAccessToken,
		TokenType:   "Bearer",
	}

	u, err := loginService.getUser(context.Background(), accessToken)
	assert.Nil(t, err)
	assert.Equal(t, &openIDConnectUser{}, u)
}

func TestGetUser(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	t.Skip("Not implemented")
}

func TestGravatarURLGeneration(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	grURL, err := generateGravatarURL("alkazako@redhat.com")
	assert.Nil(t, err)
	assert.Equal(t, "https://www.gravatar.com/avatar/0fa6cfaa2812a200c566f671803cdf2d.jpg", grURL)
}
