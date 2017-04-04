package login_test

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/auth"
	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/test"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"

	"github.com/almighty/almighty-core/login"

	"github.com/almighty/almighty-core/resource"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ProfileBlackBoxTest struct {
	gormtestsupport.RemoteTestSuite
	clean          func()
	profileService login.UserProfileService
	configuration  *config.ConfigurationData
	loginService   *login.KeycloakOAuthProvider
	accessToken    *string
	profileAPIURL  *string
}

func TestRunProfileBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Remote)
	suite.Run(t, &ProfileBlackBoxTest{RemoteTestSuite: gormtestsupport.NewRemoteTestSuite("../config.yaml")})
}

// SetupSuite overrides the RemoteTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
func (s *ProfileBlackBoxTest) SetupSuite() {

	var err error
	s.configuration, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	keycloakUserProfileService := login.NewKeycloakUserProfileClient()
	s.profileService = keycloakUserProfileService

	// Get the API endpoint - avoid repition in every test.
	r := &goa.RequestData{
		Request: &http.Request{Host: "api.example.org"},
	}
	profileAPIURL, err := s.configuration.GetKeycloakAccountEndpoint(r)
	s.profileAPIURL = &profileAPIURL

	// Get the access token ONCE which we will use for all profile related tests.
	//  - avoid repition in every test.
	token, err := s.generateAccessToken() // TODO: Use a simpler way to do this.
	assert.Nil(s.T(), err)
	s.accessToken = token
}

func (s *ProfileBlackBoxTest) SetupTest() {

	// Get the initial profile state.
	profile, err := s.profileService.Get(*s.accessToken, *s.profileAPIURL)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), profile)

	initialProfileState := &login.KeycloakUserProfile{
		Attributes: profile.Attributes,
		FirstName:  profile.FirstName,
		LastName:   profile.LastName,
		Email:      profile.Email,
	}

	// Schedule it for restoring of the initial state of the keycloak user after the test
	s.clean = s.updateUserProfile(initialProfileState)
}

func (s *ProfileBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *ProfileBlackBoxTest) generateAccessToken() (*string, error) {

	var scopes []account.Identity
	scopes = append(scopes, test.TestIdentity)
	scopes = append(scopes, test.TestObserverIdentity)

	client := &http.Client{Timeout: 10 * time.Second}
	r := &goa.RequestData{
		Request: &http.Request{Host: "api.example.org"},
	}
	tokenEndpoint, err := s.configuration.GetKeycloakEndpointToken(r)

	res, err := client.PostForm(tokenEndpoint, url.Values{
		"client_id":     {s.configuration.GetKeycloakClientID()},
		"client_secret": {s.configuration.GetKeycloakSecret()},
		"username":      {s.configuration.GetKeycloakTestUserName()},
		"password":      {s.configuration.GetKeycloakTestUserSecret()},
		"grant_type":    {"password"},
	})
	if err != nil {
		return nil, errors.NewInternalError("error when obtaining token " + err.Error())
	}

	token, err := auth.ReadToken(res)
	require.Nil(s.T(), err)
	return token.AccessToken, err
}

func (s *ProfileBlackBoxTest) TestKeycloakUserProfileUpdate() {

	// UPDATE the user profile

	testFirstName := "updatedFirstNameAgainNew" + uuid.NewV4().String()
	testLastName := "updatedLastNameNew" + uuid.NewV4().String()
	testEmail := "updatedEmail" + uuid.NewV4().String() + "@email.com"
	testBio := "updatedBioNew" + uuid.NewV4().String()
	testURL := "updatedURLNew" + uuid.NewV4().String()
	testImageURL := "updatedBio" + uuid.NewV4().String()

	testKeycloakUserProfileAttributes := &login.KeycloakUserProfileAttributes{
		login.ImageURLAttributeName: []string{testImageURL},
		login.BioAttributeName:      []string{testBio},
		login.URLAttributeName:      []string{testURL},
	}

	testKeycloakUserProfileData := login.NewKeycloakUserProfile(&testFirstName, &testLastName, &testEmail, testKeycloakUserProfileAttributes)

	updateProfileFunc := s.updateUserProfile(testKeycloakUserProfileData)
	updateProfileFunc()

	// Do a GET on the user profile
	// Use the token to update user profile
	retrievedkeycloakUserProfileData, err := s.profileService.Get(*s.accessToken, *s.profileAPIURL)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), retrievedkeycloakUserProfileData)

	assert.Equal(s.T(), testFirstName, *retrievedkeycloakUserProfileData.FirstName)
	assert.Equal(s.T(), testLastName, *retrievedkeycloakUserProfileData.LastName)

	// email is automatically stored in lower case
	assert.Equal(s.T(), strings.ToLower(testEmail), *retrievedkeycloakUserProfileData.Email)

	// validate Attributes
	retrievedBio := (*retrievedkeycloakUserProfileData.Attributes)[login.BioAttributeName]
	assert.Equal(s.T(), retrievedBio[0], testBio)
}

func (s *ProfileBlackBoxTest) TestKeycloakUserProfileGet() {
	profile, err := s.profileService.Get(*s.accessToken, *s.profileAPIURL)

	require.Nil(s.T(), err)
	assert.NotNil(s.T(), profile)

	keys := reflect.ValueOf(*profile.Attributes).MapKeys()
	assert.NotEqual(s.T(), len(keys), 0)
	assert.NotNil(s.T(), *profile.FirstName)
	assert.NotNil(s.T(), *profile.LastName)
	assert.NotNil(s.T(), *profile.Email)
	assert.NotNil(s.T(), *profile.Attributes)
}

func (s *ProfileBlackBoxTest) updateUserProfile(userProfile *login.KeycloakUserProfile) func() {
	return func() {
		err := s.profileService.Update(userProfile, *s.accessToken, *s.profileAPIURL)
		require.Nil(s.T(), err)
	}
}
