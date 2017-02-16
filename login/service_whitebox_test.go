package login

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/token"
	_ "github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

var loginService *KeycloakOAuthProvider

func setup() {

	var err error
	if err = configuration.Setup(""); err != nil {
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

	privateKey, err := token.ParsePrivateKey([]byte(configuration.GetTokenPrivateKey()))
	if err != nil {
		panic(err)
	}

	tokenManager := token.NewManagerWithPrivateKey(privateKey)
	userRepository := account.NewUserRepository(nil)
	identityRepository := account.NewIdentityRepository(nil)
	loginService = &KeycloakOAuthProvider{
		config:       oauth,
		Identities:   identityRepository,
		Users:        userRepository,
		TokenManager: tokenManager,
	}
}

func tearDown() {
	loginService = nil
}

func TestValidOAuthAccessToken(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	setup()
	defer tearDown()

	identity := account.Identity{
		ID:       uuid.NewV4(),
		Username: "testuser",
	}
	token, err := loginService.TokenManager.Generate(identity)
	assert.Nil(t, err)
	accessToken := &oauth2.Token{
		AccessToken: token,
		TokenType:   "Bearer",
	}

	claims, err := parseToken(accessToken.AccessToken, loginService.TokenManager.PublicKey())
	assert.Nil(t, err)
	assert.Equal(t, identity.ID.String(), claims.Subject)
	assert.Equal(t, identity.Username, claims.Username)
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

	_, err := parseToken(accessToken.AccessToken, loginService.TokenManager.PublicKey())
	assert.NotNil(t, err)
}

func TestCheckClaimsOK(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	claims := &keycloakTokenClaims{
		Email:    "somemail@domain.com",
		Username: "testuser",
	}
	claims.Subject = uuid.NewV4().String()

	assert.Nil(t, checkClaims(claims))
}

func TestCheckClaimsFails(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	claimsNoEmail := &keycloakTokenClaims{
		Username: "testuser",
	}
	claimsNoEmail.Subject = uuid.NewV4().String()
	assert.NotNil(t, checkClaims(claimsNoEmail))

	claimsNoUsername := &keycloakTokenClaims{
		Email: "somemail@domain.com",
	}
	claimsNoUsername.Subject = uuid.NewV4().String()
	assert.NotNil(t, checkClaims(claimsNoUsername))

	claimsNoSubject := &keycloakTokenClaims{
		Email:    "somemail@domain.com",
		Username: "testuser",
	}
	assert.NotNil(t, checkClaims(claimsNoSubject))
}

func TestGravatarURLGeneration(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	grURL, err := generateGravatarURL("alkazako@redhat.com")
	assert.Nil(t, err)
	assert.Equal(t, "https://www.gravatar.com/avatar/0fa6cfaa2812a200c566f671803cdf2d.jpg", grURL)
}

func TestEncodeTokenOK(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	referelURL, _ := url.Parse("https://example.domain.com")
	accessToken := "accessToken%@!/\\&?"
	refreshToken := "refreshToken%@!/\\&?"
	tokenType := "tokenType%@!/\\&?"
	expiresIn := 1800
	refreshExpiresIn := 1800
	outhToken := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenType,
	}
	extra := map[string]interface{}{
		"expires_in":         expiresIn,
		"refresh_expires_in": refreshExpiresIn,
	}
	err := encodeToken(referelURL, outhToken.WithExtra(extra))
	assert.Nil(t, err)
	encoded := referelURL.String()

	referelURL, _ = url.Parse(encoded)
	values := referelURL.Query()
	tJSON := values["token_json"]
	b := []byte(tJSON[0])
	tokenData := &app.TokenData{}
	err = json.Unmarshal(b, tokenData)
	assert.Nil(t, err)

	assert.Equal(t, accessToken, *tokenData.AccessToken)
	assert.Equal(t, refreshToken, *tokenData.RefreshToken)
	assert.Equal(t, tokenType, *tokenData.TokenType)
	assert.Equal(t, expiresIn, *tokenData.ExpiresIn)
	assert.Equal(t, refreshExpiresIn, *tokenData.RefreshExpiresIn)
}
