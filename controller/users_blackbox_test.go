package controller_test

import (
	"fmt"
	"testing"

	"github.com/almighty/almighty-core/account"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"

	config "github.com/almighty/almighty-core/configuration"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/login"

	"github.com/almighty/almighty-core/gormsupport/cleaner"

	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

func TestUsers(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestUsersSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type TestUsersSuite struct {
	gormtestsupport.DBTestSuite
	db             *gormapplication.GormDB
	svc            *goa.Service
	clean          func()
	controller     *UsersController
	userRepo       account.UserRepository
	identityRepo   account.IdentityRepository
	configuration  *config.ConfigurationData
	profileService login.UserProfileService
}

func (s *TestUsersSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.svc = goa.New("test")
	s.db = gormapplication.NewGormDB(s.DB)
	configData, err := config.GetConfigurationData()
	s.configuration = configData
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	testAttributeValue := "a"
	dummyProfileResponse := createDummyUserProfileResponse(&testAttributeValue, &testAttributeValue, &testAttributeValue)
	keycloakUserProfileService := newDummyUserProfileService(dummyProfileResponse)
	s.profileService = keycloakUserProfileService
	s.controller = NewUsersController(s.svc, s.db, s.configuration, s.profileService)
	s.userRepo = s.db.Users()
	s.identityRepo = s.db.Identities()

}

func (s *TestUsersSuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *TestUsersSuite) TearDownTest() {
	s.clean()
}

func (s *TestUsersSuite) SecuredController(identity account.Identity) (*goa.Service, *UsersController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))

	svc := testsupport.ServiceAsUser("Users-Service", almtoken.NewManager(pub), identity)
	return svc, NewUsersController(svc, s.db, s.configuration, s.profileService)
}
func (s *TestUsersSuite) TestUpdateUserOK() {
	// given
	user := s.createRandomUser("TestUpdateUserOK")
	identity := s.createRandomIdentity(user, account.KeycloakIDP)
	_, result := test.ShowUsersOK(s.T(), nil, nil, s.controller, identity.ID.String())
	assert.Equal(s.T(), identity.ID.String(), *result.Data.ID)
	assert.Equal(s.T(), user.FullName, *result.Data.Attributes.FullName)
	assert.Equal(s.T(), user.ImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(s.T(), identity.ProviderType, *result.Data.Attributes.ProviderType)
	assert.Equal(s.T(), identity.Username, *result.Data.Attributes.Username)
	// when
	newEmail := "updated-" + uuid.NewV4().String() + "@email.com"
	newFullName := "TestUpdateUserOK"
	newImageURL := "http://new.image.io/imageurl"
	newBio := "new bio"
	newProfileURL := "http://new.profile.url/url"
	secureService, secureController := s.SecuredController(identity)

	contextInformation := map[string]interface{}{
		"last_visited": "yesterday",
		"space":        "3d6dab8d-f204-42e8-ab29-cdb1c93130ad",
		"rate":         100.00,
		"count":        3,
	}
	//secureController, secureService := createSecureController(t, identity)
	updateUsersPayload := createUpdateUsersPayload(&newEmail, &newFullName, &newBio, &newImageURL, &newProfileURL, contextInformation)
	_, result = test.UpdateUsersOK(s.T(), secureService.Context, secureService, secureController, updateUsersPayload)

	// then
	require.NotNil(s.T(), result)
	// let's fetch it and validate
	_, result = test.ShowUsersOK(s.T(), nil, nil, s.controller, identity.ID.String())
	require.NotNil(s.T(), result)
	assert.Equal(s.T(), identity.ID.String(), *result.Data.ID)
	assert.Equal(s.T(), newFullName, *result.Data.Attributes.FullName)
	assert.Equal(s.T(), newImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(s.T(), newBio, *result.Data.Attributes.Bio)
	assert.Equal(s.T(), newProfileURL, *result.Data.Attributes.URL)
	updatedContextInformation := result.Data.Attributes.ContextInformation
	assert.Equal(s.T(), contextInformation["last_visited"], updatedContextInformation["last_visited"])

	countValue, ok := updatedContextInformation["count"].(float64)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), contextInformation["count"], int(countValue))
	assert.Equal(s.T(), contextInformation["rate"], updatedContextInformation["rate"])
}

func (s *TestUsersSuite) TestUpdateUserVariableSpacesInNameOK() {

	// given
	user := s.createRandomUser("OK")
	identity := s.createRandomIdentity(user, account.KeycloakIDP)
	_, result := test.ShowUsersOK(s.T(), nil, nil, s.controller, identity.ID.String())
	assert.Equal(s.T(), identity.ID.String(), *result.Data.ID)
	assert.Equal(s.T(), user.FullName, *result.Data.Attributes.FullName)
	assert.Equal(s.T(), user.ImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(s.T(), identity.ProviderType, *result.Data.Attributes.ProviderType)
	assert.Equal(s.T(), identity.Username, *result.Data.Attributes.Username)
	// when
	newEmail := "updated-" + uuid.NewV4().String() + "@email.com"

	// This is the special thing we are testing - everything else
	// has been tested in other tests.
	// We use the full name to derive the first and the last name
	// This test checks that the splitting is done correctly,
	// ie, the first word is the first name ,and the rest is the last name

	newFullName := " This name   has a   lot of spaces   in it"
	newImageURL := "http://new.image.io/imageurl"
	newBio := "new bio"
	newProfileURL := "http://new.profile.url/url"
	secureService, secureController := s.SecuredController(identity)

	contextInformation := map[string]interface{}{
		"last_visited": "yesterday",
		"space":        "3d6dab8d-f204-42e8-ab29-cdb1c93130ad",
		"rate":         100.00,
		"count":        3,
	}
	//secureController, secureService := createSecureController(t, identity)
	updateUsersPayload := createUpdateUsersPayload(&newEmail, &newFullName, &newBio, &newImageURL, &newProfileURL, contextInformation)
	_, result = test.UpdateUsersOK(s.T(), secureService.Context, secureService, secureController, updateUsersPayload)
	// then
	require.NotNil(s.T(), result)
	// let's fetch it and validate
	_, result = test.ShowUsersOK(s.T(), nil, nil, s.controller, identity.ID.String())
	require.NotNil(s.T(), result)
	assert.Equal(s.T(), identity.ID.String(), *result.Data.ID)
	assert.Equal(s.T(), newFullName, *result.Data.Attributes.FullName)
	assert.Equal(s.T(), newImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(s.T(), newBio, *result.Data.Attributes.Bio)
	assert.Equal(s.T(), newProfileURL, *result.Data.Attributes.URL)
	updatedContextInformation := result.Data.Attributes.ContextInformation
	assert.Equal(s.T(), contextInformation["last_visited"], updatedContextInformation["last_visited"])

	countValue, ok := updatedContextInformation["count"].(float64)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), contextInformation["count"], int(countValue))
	assert.Equal(s.T(), contextInformation["rate"], updatedContextInformation["rate"])
}

/*
	Test to unset variable in contextInformation
*/

func (s *TestUsersSuite) TestUpdateUserUnsetVariableInContextInfo() {

	// given
	user := s.createRandomUser("TestUpdateUserUnsetVariableInContextInfo")
	identity := s.createRandomIdentity(user, account.KeycloakIDP)

	_, result := test.ShowUsersOK(s.T(), nil, nil, s.controller, identity.ID.String())
	assert.Equal(s.T(), identity.ID.String(), *result.Data.ID)
	assert.Equal(s.T(), user.FullName, *result.Data.Attributes.FullName)
	assert.Equal(s.T(), user.ImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(s.T(), identity.ProviderType, *result.Data.Attributes.ProviderType)
	assert.Equal(s.T(), identity.Username, *result.Data.Attributes.Username)
	// when
	newEmail := "updated-" + uuid.NewV4().String() + "@email.com"
	newFullName := "TestUpdateUserUnsetVariableInContextInfo"
	newImageURL := "http://new.image.io/imageurl"
	newBio := "new bio"
	newProfileURL := "http://new.profile.url/url"
	secureService, secureController := s.SecuredController(identity)

	contextInformation := map[string]interface{}{
		"last_visited": "yesterday",
		"space":        "3d6dab8d-f204-42e8-ab29-cdb1c93130ad",
		"rate":         100.00,
		"count":        3,
	}
	//secureController, secureService := createSecureController(t, identity)
	updateUsersPayload := createUpdateUsersPayload(&newEmail, &newFullName, &newBio, &newImageURL, &newProfileURL, contextInformation)
	_, result = test.UpdateUsersOK(s.T(), secureService.Context, secureService, secureController, updateUsersPayload)
	// then
	require.NotNil(s.T(), result)
	// let's fetch it and validate the usual stuff.
	_, result = test.ShowUsersOK(s.T(), nil, nil, s.controller, identity.ID.String())
	require.NotNil(s.T(), result)
	assert.Equal(s.T(), identity.ID.String(), *result.Data.ID)
	assert.Equal(s.T(), newFullName, *result.Data.Attributes.FullName)
	assert.Equal(s.T(), newImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(s.T(), newBio, *result.Data.Attributes.Bio)
	assert.Equal(s.T(), newProfileURL, *result.Data.Attributes.URL)
	updatedContextInformation := result.Data.Attributes.ContextInformation
	assert.Equal(s.T(), contextInformation["last_visited"], updatedContextInformation["last_visited"])

	/** Usual stuff done, now lets unset **/
	contextInformation = map[string]interface{}{
		"last_visited": nil,
		"space":        "3d6dab8d-f204-42e8-ab29-cdb1c93130ad",
		"rate":         100.00,
		"count":        3,
	}

	updateUsersPayload = createUpdateUsersPayload(&newEmail, &newFullName, &newBio, &newImageURL, &newProfileURL, contextInformation)
	_, result = test.UpdateUsersOK(s.T(), secureService.Context, secureService, secureController, updateUsersPayload)
	// then
	require.NotNil(s.T(), result)
	// let's fetch it and validate the usual stuff.
	_, result = test.ShowUsersOK(s.T(), nil, nil, s.controller, identity.ID.String())
	require.NotNil(s.T(), result)
	updatedContextInformation = result.Data.Attributes.ContextInformation

	// what was passed as non-nill should be intact.
	assert.Equal(s.T(), contextInformation["space"], updatedContextInformation["space"])

	// what was pass as nil should not be found!
	_, ok := updatedContextInformation["last_visited"]
	assert.Equal(s.T(), false, ok)
}

/*
	Pass no contextInformation and no one complains.
	This is as per general service behaviour.
*/

func (s *TestUsersSuite) TestUpdateUserOKWithoutContextInfo() {

	// given
	user := s.createRandomUser("TestUpdateUserOKWithoutContextInfo")
	identity := s.createRandomIdentity(user, account.KeycloakIDP)
	_, result := test.ShowUsersOK(s.T(), nil, nil, s.controller, identity.ID.String())
	assert.Equal(s.T(), identity.ID.String(), *result.Data.ID)
	assert.Equal(s.T(), user.FullName, *result.Data.Attributes.FullName)
	assert.Equal(s.T(), user.ImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(s.T(), identity.ProviderType, *result.Data.Attributes.ProviderType)
	assert.Equal(s.T(), identity.Username, *result.Data.Attributes.Username)
	// when
	newEmail := "updated-" + uuid.NewV4().String() + "@email.com"
	newFullName := "TestUpdateUserOKWithoutContextInfo"
	newImageURL := "http://new.image.io/imageurl"
	newBio := "new bio"
	newProfileURL := "http://new.profile.url/url"
	secureService, secureController := s.SecuredController(identity)

	updateUsersPayload := createUpdateUsersPayloadWithoutContextInformation(&newEmail, &newFullName, &newBio, &newImageURL, &newProfileURL)
	test.UpdateUsersOK(s.T(), secureService.Context, secureService, secureController, updateUsersPayload)
}

func (s *TestUsersSuite) TestPatchUserContextInformation() {

	// given
	user := s.createRandomUser("TestPatchUserContextInformation")
	identity := s.createRandomIdentity(user, account.KeycloakIDP)
	_, result := test.ShowUsersOK(s.T(), nil, nil, s.controller, identity.ID.String())
	assert.Equal(s.T(), identity.ID.String(), *result.Data.ID)
	assert.Equal(s.T(), user.FullName, *result.Data.Attributes.FullName)
	assert.Equal(s.T(), user.ImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(s.T(), identity.ProviderType, *result.Data.Attributes.ProviderType)
	assert.Equal(s.T(), identity.Username, *result.Data.Attributes.Username)
	// when
	secureService, secureController := s.SecuredController(identity)

	contextInformation := map[string]interface{}{
		"last_visited": "yesterday",
		"count":        3,
	}
	//secureController, secureService := createSecureController(t, identity)
	updateUsersPayload := createUpdateUsersPayload(nil, nil, nil, nil, nil, contextInformation)
	_, result = test.UpdateUsersOK(s.T(), secureService.Context, secureService, secureController, updateUsersPayload)
	// then
	require.NotNil(s.T(), result)

	// let's fetch it and validate the usual stuff.
	_, result = test.ShowUsersOK(s.T(), nil, nil, s.controller, identity.ID.String())
	require.NotNil(s.T(), result)
	assert.Equal(s.T(), identity.ID.String(), *result.Data.ID)
	updatedContextInformation := result.Data.Attributes.ContextInformation

	// Before we PATCH, ensure that the 1st time update has worked well.
	assert.Equal(s.T(), contextInformation["last_visited"], updatedContextInformation["last_visited"])
	countValue, ok := updatedContextInformation["count"].(float64)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), contextInformation["count"], int(countValue))

	/** Usual stuff done, now lets PATCH only 1 contextInformation attribute **/
	patchedContextInformation := map[string]interface{}{
		"count": 5,
	}

	updateUsersPayload = createUpdateUsersPayload(nil, nil, nil, nil, nil, patchedContextInformation)
	_, result = test.UpdateUsersOK(s.T(), secureService.Context, secureService, secureController, updateUsersPayload)
	require.NotNil(s.T(), result)

	// let's fetch it and validate the usual stuff.
	_, result = test.ShowUsersOK(s.T(), nil, nil, s.controller, identity.ID.String())
	require.NotNil(s.T(), result)
	updatedContextInformation = result.Data.Attributes.ContextInformation

	// what was NOT passed, should remain intact.
	assert.Equal(s.T(), contextInformation["last_visited"], updatedContextInformation["last_visited"])

	// what WAS PASSED, should be updated.
	countValue, ok = updatedContextInformation["count"].(float64)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), patchedContextInformation["count"], int(countValue))

}

func (s *TestUsersSuite) TestUpdateUserUnauthorized() {
	// given
	user := s.createRandomUser("TestUpdateUserUnauthorized")
	identity := s.createRandomIdentity(user, account.KeycloakIDP)
	_, result := test.ShowUsersOK(s.T(), nil, nil, s.controller, identity.ID.String())
	assert.Equal(s.T(), identity.ID.String(), *result.Data.ID)
	assert.Equal(s.T(), user.FullName, *result.Data.Attributes.FullName)
	assert.Equal(s.T(), user.ImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(s.T(), identity.ProviderType, *result.Data.Attributes.ProviderType)
	assert.Equal(s.T(), identity.Username, *result.Data.Attributes.Username)
	newEmail := "updated@email.com"
	newFullName := "TestUpdateUserUnauthorized"
	newImageURL := "http://new.image.io/imageurl"
	newBio := "new bio"
	newProfileURL := "http://new.profile.url/url"
	contextInformation := map[string]interface{}{
		"last_visited": "yesterday",
		"space":        "3d6dab8d-f204-42e8-ab29-cdb1c93130ad",
	}
	//secureController, secureService := createSecureController(t, identity)
	updateUsersPayload := createUpdateUsersPayload(&newEmail, &newFullName, &newBio, &newImageURL, &newProfileURL, contextInformation)
	// when/then
	test.UpdateUsersUnauthorized(s.T(), context.Background(), nil, s.controller, updateUsersPayload)
}

func (s *TestUsersSuite) TestShowUserOK() {
	// given user
	user := s.createRandomUser("TestShowUserOK")
	identity := s.createRandomIdentity(user, account.KeycloakIDP)
	// when
	_, result := test.ShowUsersOK(s.T(), nil, nil, s.controller, identity.ID.String())
	// then
	assert.Equal(s.T(), identity.ID.String(), *result.Data.ID)
	assert.Equal(s.T(), user.FullName, *result.Data.Attributes.FullName)
	assert.Equal(s.T(), user.ImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(s.T(), identity.ProviderType, *result.Data.Attributes.ProviderType)
	assert.Equal(s.T(), identity.Username, *result.Data.Attributes.Username)
}

func (s *TestUsersSuite) TestListUsersOK() {
	// given user1
	user1 := s.createRandomUser("TestListUsersOK1")
	identity11 := s.createRandomIdentity(user1, account.KeycloakIDP)
	identity12 := s.createRandomIdentity(user1, "github-test")
	// given user2
	user2 := s.createRandomUser("TestListUsersOK2")
	identity2 := s.createRandomIdentity(user2, account.KeycloakIDP)
	// when
	_, result := test.ListUsersOK(s.T(), nil, nil, s.controller)
	// then
	s.T().Log(fmt.Sprintf("User1 #%s: %s %s", user1.ID.String(), identity11.ID.String(), identity12.ID.String()))
	s.T().Log(fmt.Sprintf("User2 #%s: %s", user2.ID.String(), identity2.ID.String()))
	for i, data := range result.Data {
		s.T().Log(fmt.Sprintf("Result #%d: %s %v", i, *data.ID, *data.Attributes.Username))
	}
	assertUser(s.T(), findUser(identity11.ID, result.Data), user1, identity11)
	assertUser(s.T(), findUser(identity2.ID, result.Data), user2, identity2)
}

func (s *TestUsersSuite) createRandomUser(fullname string) account.User {
	user := account.User{
		Email:    uuid.NewV4().String() + "primaryForUpdat7e@example.com",
		FullName: fullname,
		ImageURL: "someURLForUpdate",
		ID:       uuid.NewV4(),
	}
	err := s.userRepo.Create(context.Background(), &user)
	require.Nil(s.T(), err)
	return user
}
func (s *TestUsersSuite) createRandomIdentity(user account.User, providerType string) account.Identity {
	profile := "foobarforupdate.com/" + uuid.NewV4().String() + "/" + user.ID.String()
	identity := account.Identity{
		Username:     "TestUpdateUserIntegration123" + uuid.NewV4().String(),
		ProviderType: providerType,
		ProfileURL:   &profile,
		User:         user,
		UserID:       account.NullUUID{UUID: user.ID, Valid: true},
	}
	err := s.identityRepo.Create(context.Background(), &identity)
	require.Nil(s.T(), err)
	return identity
}

func findUser(id uuid.UUID, identityData []*app.IdentityData) *app.IdentityData {
	for _, identity := range identityData {
		if *identity.ID == id.String() {
			return identity
		}
	}
	return nil
}

func assertUser(t *testing.T, actual *app.IdentityData, expectedUser account.User, expectedIdentity account.Identity) {
	require.NotNil(t, actual)
	assert.Equal(t, expectedIdentity.Username, *actual.Attributes.Username)
	assert.Equal(t, expectedIdentity.ProviderType, *actual.Attributes.ProviderType)
	assert.Equal(t, expectedUser.FullName, *actual.Attributes.FullName)
	assert.Equal(t, expectedUser.ImageURL, *actual.Attributes.ImageURL)
	assert.Equal(t, expectedUser.Email, *actual.Attributes.Email)
}

func createUpdateUsersPayload(email, fullName, bio, imageURL, profileURL *string, contextInformation map[string]interface{}) *app.UpdateUsersPayload {
	return &app.UpdateUsersPayload{
		Data: &app.UpdateIdentityData{
			Type: "identities",
			Attributes: &app.IdentityDataAttributes{
				Email:              email,
				FullName:           fullName,
				Bio:                bio,
				ImageURL:           imageURL,
				URL:                profileURL,
				ContextInformation: contextInformation,
			},
		},
	}
}

func createUpdateUsersPayloadWithoutContextInformation(email, fullName, bio, imageURL, profileURL *string) *app.UpdateUsersPayload {
	return &app.UpdateUsersPayload{
		Data: &app.UpdateIdentityData{
			Type: "identities",
			Attributes: &app.IdentityDataAttributes{
				Email:    email,
				FullName: fullName,
				Bio:      bio,
				ImageURL: imageURL,
				URL:      profileURL,
			},
		},
	}
}

type dummyUserProfileService struct {
	dummyGetResponse *login.KeycloakUserProfileResponse
}

func newDummyUserProfileService(dummyGetResponse *login.KeycloakUserProfileResponse) *dummyUserProfileService {
	return &dummyUserProfileService{
		dummyGetResponse: dummyGetResponse,
	}
}

func (d *dummyUserProfileService) Update(keycloakUserProfile *login.KeycloakUserProfile, accessToken string, keycloakProfileURL string) error {
	return nil
}

func (d *dummyUserProfileService) Get(accessToken string, keycloakProfileURL string) (*login.KeycloakUserProfileResponse, error) {
	return d.dummyGetResponse, nil
}

func (d *dummyUserProfileService) SetDummyGetResponse(dummyGetResponse *login.KeycloakUserProfileResponse) {
	d.dummyGetResponse = dummyGetResponse
}

func createDummyUserProfileResponse(updatedBio, updatedImageURL, updatedURL *string) *login.KeycloakUserProfileResponse {
	profile := &login.KeycloakUserProfileResponse{}
	profile.Attributes = &login.KeycloakUserProfileAttributes{}

	(*profile.Attributes)[login.BioAttributeName] = []string{*updatedBio}
	(*profile.Attributes)[login.ImageURLAttributeName] = []string{*updatedImageURL}
	(*profile.Attributes)[login.URLAttributeName] = []string{*updatedURL}

	return profile

}
