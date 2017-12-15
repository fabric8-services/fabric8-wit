package controller_test

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"

	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestUsers(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestUsersSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type TestUsersSuite struct {
	gormtestsupport.DBTestSuite
	db           *gormapplication.GormDB
	svc          *goa.Service
	clean        func()
	controller   *UsersController
	userRepo     account.UserRepository
	identityRepo account.IdentityRepository
}

func (s *TestUsersSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.svc = goa.New("test")
	s.db = gormapplication.NewGormDB(s.DB)
	s.controller = NewUsersController(s.svc, s.db, s.Configuration)
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
	svc := testsupport.ServiceAsUser("Users-Service", identity)
	return svc, NewUsersController(svc, s.db, s.Configuration)
}

func (s *TestUsersSuite) SecuredServiceAccountController(identity account.Identity) (*goa.Service, *UsersController) {
	svc := testsupport.ServiceAsServiceAccountUser("Users-ServiceAccount-Service", identity)
	return svc, NewUsersController(svc, s.db, s.Configuration)
}

func (s *TestUsersSuite) TestUpdateUserAsServiceAccountUnauthorized() {
	// given
	user := s.createRandomUser("TestUpdateUserAsSvcAcUnauthorized")
	identity := s.createRandomIdentity(user, account.KeycloakIDP)

	// when
	newEmail := "TestUpdateUserOK-" + uuid.NewV4().String() + "@email.com"
	newFullName := "TestUpdateUserOK"
	newImageURL := "http://new.image.io/imageurl"
	newBio := "new bio"
	newProfileURL := "http://new.profile.url/url"
	newCompany := "updateCompany " + uuid.NewV4().String()
	secureService, secureController := s.SecuredController(identity)

	contextInformation := map[string]interface{}{
		"last_visited": "yesterday",
		"space":        "3d6dab8d-f204-42e8-ab29-cdb1c93130ad",
		"rate":         100.00,
		"count":        3,
	}
	updateUsersPayload := createUpdateUsersAsServiceAccountPayload(&newEmail, &newFullName, &newBio, &newImageURL, &newProfileURL, &newCompany, nil, nil, contextInformation)

	idAsString := (identity.ID).String()
	test.UpdateUserAsServiceAccountUsersUnauthorized(s.T(), secureService.Context, secureService, secureController, idAsString, updateUsersPayload)

}

func (s *TestUsersSuite) TestUpdateUserAsServiceAccountBadRequest() {
	// given
	user := s.createRandomUser("TestUpdateUserAsServiceAccountBadRequest")
	identity := s.createRandomIdentity(user, account.KeycloakIDP)

	// when
	newEmail := "TestUpdateUserOK-" + uuid.NewV4().String() + "@email.com"
	newFullName := "TestUpdateUserOK"
	newImageURL := "http://new.image.io/imageurl"
	newBio := "new bio"
	newProfileURL := "http://new.profile.url/url"
	newCompany := "updateCompany " + uuid.NewV4().String()
	secureService, secureController := s.SecuredServiceAccountController(identity)

	contextInformation := map[string]interface{}{
		"last_visited": "yesterday",
		"space":        "3d6dab8d-f204-42e8-ab29-cdb1c93130ad",
		"rate":         100.00,
		"count":        3,
	}
	updateUsersPayload := createUpdateUsersAsServiceAccountPayload(&newEmail, &newFullName, &newBio, &newImageURL, &newProfileURL, &newCompany, nil, nil, contextInformation)

	idAsString := "bad-uuid"
	test.UpdateUserAsServiceAccountUsersBadRequest(s.T(), secureService.Context, secureService, secureController, idAsString, updateUsersPayload)

}

func (s *TestUsersSuite) TestUpdateUserAsServiceAccountOK() {
	// given
	user := s.createRandomUser("TestUpdateUserAsServiceAccountOK")
	identity := s.createRandomIdentity(user, account.KeycloakIDP)

	// when
	user.Email = "TestUpdateUserOK-" + uuid.NewV4().String() + "@email.com"
	user.FullName = "TestUpdateUserOK"
	user.ImageURL = "http://new.image.io/imageurl"
	user.Bio = "new bio"
	user.URL = "http://new.profile.url/url"
	user.Company = "updateCompany " + uuid.NewV4().String()
	secureService, secureController := s.SecuredServiceAccountController(identity)

	contextInformation := map[string]interface{}{
		"last_visited": "yesterday",
		"space":        "3d6dab8d-f204-42e8-ab29-cdb1c93130ad",
		"rate":         100.00,
		"count":        3,
	}
	updateUsersPayload := createUpdateUsersAsServiceAccountPayload(&user.Email, &user.FullName, &user.Bio, &user.ImageURL, &user.URL, &user.Company, nil, nil, contextInformation)
	test.UpdateUserAsServiceAccountUsersOK(s.T(), secureService.Context, secureService, secureController, (identity.ID).String(), updateUsersPayload)
}

func (s *TestUsersSuite) TestUpdateUserAsServiceAccountNotFound() {
	// given
	user := s.createRandomUser("TestUpdateUserAsServiceAccountNotFound")
	identity := s.createRandomIdentity(user, account.KeycloakIDP)

	// when
	newEmail := "TestUpdateUserOK-" + uuid.NewV4().String() + "@email.com"
	newFullName := "TestUpdateUserOK"
	newImageURL := "http://new.image.io/imageurl"
	newBio := "new bio"
	newProfileURL := "http://new.profile.url/url"
	newCompany := "updateCompany " + uuid.NewV4().String()
	secureService, secureController := s.SecuredServiceAccountController(identity)

	contextInformation := map[string]interface{}{
		"last_visited": "yesterday",
		"space":        "3d6dab8d-f204-42e8-ab29-cdb1c93130ad",
		"rate":         100.00,
		"count":        3,
	}
	updateUsersPayload := createUpdateUsersAsServiceAccountPayload(&newEmail, &newFullName, &newBio, &newImageURL, &newProfileURL, &newCompany, nil, nil, contextInformation)

	idAsString := uuid.NewV4().String() // will never be found.
	test.UpdateUserAsServiceAccountUsersNotFound(s.T(), secureService.Context, secureService, secureController, idAsString, updateUsersPayload)

}

func (s *TestUsersSuite) TestCreateUserAsServiceAccountOK() {
	// given
	user := s.createRandomUserObject("TestCreateUserAsServiceAccountOK")
	identity := s.createRandomIdentityObject(user, "KC")

	user.ContextInformation = map[string]interface{}{
		"last_visited": "yesterday",
		"space":        "3d6dab8d-f204-42e8-ab29-cdb1c93130ad",
		"rate":         100.00,
		"count":        3,
	}
	secureService, secureController := s.SecuredServiceAccountController(identity)

	// when
	createUserPayload := createCreateUsersAsServiceAccountPayload(&user.Email, &user.FullName, &user.Bio, &user.ImageURL, &user.URL, &user.Company, &identity.Username, &identity.RegistrationCompleted, user.ContextInformation, user.ID.String())
	test.CreateUserAsServiceAccountUsersOK(s.T(), secureService.Context, secureService, secureController, identity.ID.String(), createUserPayload)
}

func (s *TestUsersSuite) TestCreateUserAsServiceAccountUnAuthorized() {

	// given

	newEmail := "T" + uuid.NewV4().String() + "@email.com"
	newFullName := "TesTCreateUserOK"
	newImageURL := "http://new.image.io/imageurl"
	newBio := "new bio"
	newProfileURL := "http://new.profile.url/url"
	newCompany := "u" + uuid.NewV4().String()
	username := "T" + uuid.NewV4().String()
	secureService, secureController := s.SecuredController(testsupport.TestIdentity)
	registrationCompleted := false
	identityId := uuid.NewV4()
	userID := uuid.NewV4()

	contextInformation := map[string]interface{}{
		"last_visited": "yesterday",
		"space":        "3d6dab8d-f204-42e8-ab29-cdb1c93130ad",
		"rate":         100.00,
		"count":        3,
	}

	// then
	createUserPayload := createCreateUsersAsServiceAccountPayload(&newEmail, &newFullName, &newBio, &newImageURL, &newProfileURL, &newCompany, &username, &registrationCompleted, contextInformation, userID.String())
	test.CreateUserAsServiceAccountUsersUnauthorized(s.T(), secureService.Context, secureService, secureController, identityId.String(), createUserPayload)
}

func (s *TestUsersSuite) TestCreateUserAsServiceAccountBadRequest() {

	// given

	newEmail := "T" + uuid.NewV4().String() + "@email.com"
	newFullName := "TesTCreateUserOK"
	newImageURL := "http://new.image.io/imageurl"
	newBio := "new bio"
	newProfileURL := "http://new.profile.url/url"
	newCompany := "u" + uuid.NewV4().String()
	username := "T" + uuid.NewV4().String()
	secureService, secureController := s.SecuredServiceAccountController(testsupport.TestIdentity)
	registrationCompleted := false
	userID := uuid.NewV4()

	contextInformation := map[string]interface{}{
		"last_visited": "yesterday",
		"space":        "3d6dab8d-f204-42e8-ab29-cdb1c93130ad",
		"rate":         100.00,
		"count":        3,
	}

	createUserPayload := createCreateUsersAsServiceAccountPayload(&newEmail, &newFullName, &newBio, &newImageURL, &newProfileURL, &newCompany, &username, &registrationCompleted, contextInformation, userID.String())

	// then
	test.CreateUserAsServiceAccountUsersBadRequest(s.T(), secureService.Context, secureService, secureController, "invalid-uuid", createUserPayload)
}

func (s *TestUsersSuite) createRandomUser(fullname string) account.User {
	user := account.User{
		Email:    uuid.NewV4().String() + "primaryForUpdat7e@example.com",
		FullName: fullname,
		ImageURL: "someURLForUpdate",
		ID:       uuid.NewV4(),
		Company:  uuid.NewV4().String() + "company",
	}
	err := s.userRepo.Create(context.Background(), &user)
	require.NoError(s.T(), err)
	return user
}

func (s *TestUsersSuite) createRandomUserObject(fullname string) account.User {
	user := account.User{
		Email:    uuid.NewV4().String() + "primaryForUpdat7e@example.com",
		FullName: fullname,
		ImageURL: "someURLForUpdate",
		ID:       uuid.NewV4(),
		Company:  uuid.NewV4().String() + "company",
	}
	return user
}
func (s *TestUsersSuite) createRandomIdentityObject(user account.User, providerType string) account.Identity {
	profile := "foobarforupdate.com/" + uuid.NewV4().String() + "/" + user.ID.String()
	identity := account.Identity{
		Username:     "TestUpdateUserIntegration123" + uuid.NewV4().String(),
		ProviderType: providerType,
		ProfileURL:   &profile,
		User:         user,
		UserID:       account.NullUUID{UUID: user.ID, Valid: true},
	}
	return identity
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
	require.NoError(s.T(), err)
	return identity
}

func createUpdateUsersAsServiceAccountPayload(email, fullName, bio, imageURL, profileURL, company, username *string, registrationCompleted *bool, contextInformation map[string]interface{}) *app.UpdateUserAsServiceAccountUsersPayload {
	return &app.UpdateUserAsServiceAccountUsersPayload{
		Data: &app.UpdateUserData{
			Type: "identities",
			Attributes: &app.UpdateIdentityDataAttributes{
				Email:                 email,
				FullName:              fullName,
				Bio:                   bio,
				ImageURL:              imageURL,
				URL:                   profileURL,
				Company:               company,
				ContextInformation:    contextInformation,
				Username:              username,
				RegistrationCompleted: registrationCompleted,
			},
		},
	}
}

func createCreateUsersAsServiceAccountPayload(email, fullName, bio, imageURL, profileURL, company, username *string, registrationCompleted *bool, contextInformation map[string]interface{}, userID string) *app.CreateUserAsServiceAccountUsersPayload {

	return &app.CreateUserAsServiceAccountUsersPayload{
		Data: &app.CreateUserData{
			Type: "identities",
			Attributes: &app.CreateIdentityDataAttributes{
				UserID:                userID,
				Email:                 *email,
				FullName:              fullName,
				Bio:                   bio,
				ImageURL:              imageURL,
				URL:                   profileURL,
				Company:               company,
				ContextInformation:    contextInformation,
				Username:              *username,
				RegistrationCompleted: registrationCompleted,
				ProviderType:          "KC",
			},
		},
	}
}
