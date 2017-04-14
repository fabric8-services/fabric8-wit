package login

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	testtoken "github.com/almighty/almighty-core/test/token"
	"github.com/almighty/almighty-core/token"

	_ "github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

var (
	oauth         *oauth2.Config
	configuration *config.ConfigurationData
	loginService  *KeycloakOAuthProvider
	privateKey    *rsa.PrivateKey
)

func init() {
	var err error
	configuration, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	privateKey, err = token.ParsePrivateKey([]byte(configuration.GetTokenPrivateKey()))
	if err != nil {
		panic(err)
	}

	oauth = &oauth2.Config{
		ClientID:     configuration.GetKeycloakClientID(),
		ClientSecret: configuration.GetKeycloakSecret(),
		Scopes:       []string{"user:email"},
		Endpoint:     oauth2.Endpoint{},
	}
}

func setup() {

	tokenManager := token.NewManagerWithPrivateKey(privateKey)
	userRepository := account.NewUserRepository(nil)
	identityRepository := account.NewIdentityRepository(nil)
	loginService = &KeycloakOAuthProvider{
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
	token, err := testtoken.GenerateToken(identity.ID.String(), identity.Username, privateKey)
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

func TestApprovedUserOK(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	var attributes KeycloakUserProfileAttributes
	attributes = make(map[string][]string)
	attributes[ApprovedAttributeName] = []string{"true"}
	profile := &KeycloakUserProfileResponse{Attributes: &attributes}
	approved, err := checkApproved(context.Background(), newDummyUserProfileService(profile), "", "")
	assert.Nil(t, err)
	assert.True(t, approved)
}

func TestNotApprovedUserFails(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	approved, err := checkApproved(context.Background(), newDummyUserProfileService(&KeycloakUserProfileResponse{}), "", "")
	assert.Nil(t, err)
	assert.False(t, approved)

	var attributes KeycloakUserProfileAttributes
	attributes = make(map[string][]string)
	profile := &KeycloakUserProfileResponse{Attributes: &attributes}

	approved, err = checkApproved(context.Background(), newDummyUserProfileService(profile), "", "")
	assert.Nil(t, err)
	assert.False(t, approved)

	attributes[ApprovedAttributeName] = []string{"false"}

	approved, err = checkApproved(context.Background(), newDummyUserProfileService(profile), "", "")
	assert.Nil(t, err)
	assert.False(t, approved)

	attributes[ApprovedAttributeName] = []string{"blahblah", "anydata"}

	approved, err = checkApproved(context.Background(), newDummyUserProfileService(profile), "", "")
	assert.NotNil(t, err)
	assert.False(t, approved)
}

type dummyUserProfileService struct {
	profile *KeycloakUserProfileResponse
}

func newDummyUserProfileService(profile *KeycloakUserProfileResponse) *dummyUserProfileService {
	return &dummyUserProfileService{profile: profile}
}

func (d *dummyUserProfileService) Update(keycloakUserProfile *KeycloakUserProfile, accessToken string, keycloakProfileURL string) error {
	return nil
}

func (d *dummyUserProfileService) Get(accessToken string, keycloakProfileURL string) (*KeycloakUserProfileResponse, error) {
	return d.profile, nil
}
