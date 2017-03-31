package login_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/auth"
	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/test"
	testtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"

	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/migration"

	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type profileBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	db             *gormapplication.GormDB
	clean          func()
	ctx            context.Context
	profileService login.UserProfileService
	configuration  *config.ConfigurationData
	loginService   *login.KeycloakOAuthProvider
}

func TestRunProfileBlackBoxTest(t *testing.T) {
	suite.Run(t, &profileBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *profileBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if _, c := os.LookupEnv(resource.Database); c != false {
		if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			s.ctx = migration.NewMigrationContext(context.Background())
			return migration.PopulateCommonTypes(s.ctx, tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}

	var err error
	s.configuration, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	keycloakUserProfileService := login.NewKeycloakUserProfileClient()
	s.profileService = keycloakUserProfileService

}

func newTestKeycloakOAuthProvider(db application.DB, configuration config.ConfigurationData) *login.KeycloakOAuthProvider {
	oauth := &oauth2.Config{
		ClientID:     configuration.GetKeycloakClientID(),
		ClientSecret: configuration.GetKeycloakSecret(),
		Scopes:       []string{"user:email"},
		Endpoint:     oauth2.Endpoint{},
	}

	publicKey, err := testtoken.ParsePublicKey([]byte(testtoken.RSAPublicKey))
	if err != nil {
		panic(err)
	}

	tokenManager := testtoken.NewManager(publicKey)
	return login.NewKeycloakOAuthProvider(oauth, db.Identities(), db.Users(), tokenManager, db)
}
func (s *profileBlackBoxTest) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *profileBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *profileBlackBoxTest) generateAccessToken() (*string, error) {

	var scopes []account.Identity
	scopes = append(scopes, test.TestIdentity)
	scopes = append(scopes, test.TestObserverIdentity)

	client := &http.Client{Timeout: 10 * time.Second}
	r := &goa.RequestData{
		Request: &http.Request{Host: "api.example.org"},
	}
	tokenEndpoint, err := s.configuration.GetKeycloakEndpointToken(r)

	assert.Equal(s.T(), tokenEndpoint, "https://sso.prod-preview.openshift.io/auth/realms/fabric8-test/protocol/openid-connect/token")

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

func (s *profileBlackBoxTest) TestKeycloakUserProfileUpdate() {

	// UPDATE the user profile

	token, err := s.generateAccessToken() // TODO: Use a simpler way to do this.
	assert.Nil(s.T(), err)
	fmt.Println(*token)

	// Use the token to update user profile
	//keycloakUserProfileData := login.KeycloakUserProfile{}
	//keycloakUserProfileData.Attributes = &login.KeycloakUserProfileAttributes{}

	testFirstName := "updatedFirstNameAgainNew"
	testLastName := "updatedLastNameNew"
	testEmail := "updatedEmail"
	testBio := "updatedBioNew"
	testURL := "updatedURLNew"
	testImageURL := "updatedBio"

	testKeycloakUserProfileAttributes := &login.KeycloakUserProfileAttributes{
		login.ImageURLAttributeName: []string{testImageURL},
		login.BioAttributeName:      []string{testBio},
		login.URLAttributeName:      []string{testURL},
	}

	testKeycloakUserProfileData := login.NewKeycloakUserProfile(&testFirstName, &testLastName, &testEmail, testKeycloakUserProfileAttributes)

	/*
		keycloakUserProfileData.FirstName = &testFirstName
		keycloakUserProfileData.LastName = &testLastName
		keycloakUserProfileData.Email = &testEmail
		//keycloakUserProfileData.Attributes.Bio = &testBio
		(*keycloakUserProfileData.Attributes)["URL"] = &testURL
		(*keycloakUserProfileData.Attributes)["ImageURL"] = &testImageURL
	*/

	// TODO: take from configuration
	r := &goa.RequestData{
		Request: &http.Request{Host: "api.example.org"},
	}
	profileAPIURL, err := s.configuration.GetKeycloakAccountEndpoint(r)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "https://sso.prod-preview.openshift.io/auth/realms/fabric8-test/account", profileAPIURL)

	err = s.profileService.Update(testKeycloakUserProfileData, *token, profileAPIURL)
	require.Nil(s.T(), err)

	// Do a GET on the user profile
	// Use the token to update user profile
	retrievedkeycloakUserProfileData, err := s.profileService.Get(*token, profileAPIURL)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), retrievedkeycloakUserProfileData)

	fmt.Println(*retrievedkeycloakUserProfileData)

	assert.Equal(s.T(), testFirstName, *retrievedkeycloakUserProfileData.FirstName)
	assert.Equal(s.T(), testLastName, *retrievedkeycloakUserProfileData.LastName)

	// email is automatically stored in lower case
	assert.Equal(s.T(), strings.ToLower(testEmail), *retrievedkeycloakUserProfileData.Email)

	// validate Attributes
	retrievedBio := (*retrievedkeycloakUserProfileData.Attributes)[login.BioAttributeName]
	assert.Equal(s.T(), retrievedBio[0], testBio)
}

func (s *profileBlackBoxTest) TestKeycloakUserProfileGet() {

	token, err := s.generateAccessToken() // TODO: Use a simpler way to do this.
	require.Nil(s.T(), err)

	// TODO: take from configuration
	r := &goa.RequestData{
		Request: &http.Request{Host: "api.example.org"},
	}
	profileAPIURL, err := s.configuration.GetKeycloakAccountEndpoint(r)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "https://sso.prod-preview.openshift.io/auth/realms/fabric8-test/account", profileAPIURL)

	profile, err := s.profileService.Get(*token, profileAPIURL)

	require.Nil(s.T(), err)
	assert.NotNil(s.T(), profile)

	keys := reflect.ValueOf(*profile.Attributes).MapKeys()
	assert.NotEqual(s.T(), len(keys), 0)

}
