package controller_test

import (
	"context"
	"github.com/fabric8-services/fabric8-wit/space"
	"testing"

	"github.com/fabric8-services/fabric8-common/id"
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestUsers(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestUsersSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

type TestUsersSuite struct {
	gormtestsupport.DBTestSuite
	svc          *goa.Service
	controller   *UsersController
	userRepo     account.UserRepository
	identityRepo account.IdentityRepository
	spaceRepo    space.Repository
}

func (s *TestUsersSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.svc = goa.New("test")
	s.controller = NewUsersController(s.svc, s.GormDB, s.Configuration)
	s.userRepo = s.GormDB.Users()
	s.identityRepo = s.GormDB.Identities()
	s.spaceRepo = s.GormDB.Spaces()
}

func (s *TestUsersSuite) SecuredController(identity account.Identity) (*goa.Service, *UsersController) {
	svc := testsupport.ServiceAsUser("Users-Service", identity)
	return svc, NewUsersController(svc, s.GormDB, s.Configuration)
}

func (s *TestUsersSuite) SecuredServiceAccountController(identity account.Identity) (*goa.Service, *UsersController) {
	svc := testsupport.ServiceAsServiceAccountUser("Users-ServiceAccount-Service", identity)
	return svc, NewUsersController(svc, s.GormDB, s.Configuration)
}

func (s *TestUsersSuite) TestDeleteUsers() {
	t := s.T()
	t.Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.Identities(1), tf.Spaces(1))
		// when
		secureService, secureController := s.SecuredServiceAccountController(*fxt.Identities[0])
		test.DeleteUsersOK(t, secureService.Context, secureService, secureController, fxt.Identities[0].Username)
		// then
		_, err := s.userRepo.Load(context.Background(), fxt.Users[0].ID)
		require.Error(t, err, "User should have been deleted")
		err = s.userRepo.CheckExists(context.Background(), fxt.Users[0].ID)
		require.Error(t, err, "User should not exist")
		_, errID := s.identityRepo.Load(context.Background(), fxt.Identities[0].ID)
		require.Error(t, errID, "Identity should have been deleted")
		err = s.identityRepo.CheckExists(context.Background(), fxt.Identities[0].ID)
		require.Error(t, err, "Identity should not exist")
		_, errSpace := s.spaceRepo.Load(context.Background(), fxt.Spaces[0].ID)
		require.Error(t, errSpace, "Space should have been deleted")
		err = s.spaceRepo.CheckExists(context.Background(), fxt.Spaces[0].ID)
		require.Error(t, err, "Space should not exist")
	})
	t.Run("a user with multiple identities", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.Identities(3), tf.Spaces(1))
		// when
		secureService, secureController := s.SecuredServiceAccountController(*fxt.Identities[0])
		test.DeleteUsersOK(t, secureService.Context, secureService, secureController, fxt.Identities[0].Username)
		// then
		_, err := s.userRepo.Load(context.Background(), fxt.Users[0].ID)
		require.Error(t, err, "User should have been deleted")
		err = s.userRepo.CheckExists(context.Background(), fxt.Users[0].ID)
		require.Error(t, err, "User should not exist")
		for _, identity := range fxt.Identities {
			_, errID := s.identityRepo.Load(context.Background(), identity.ID)
			require.Error(t, errID, "Identity should have been deleted")
			err = s.identityRepo.CheckExists(context.Background(), identity.ID)
			require.Error(t, err, "Identity should not exist")
		}
		_, errSpace := s.spaceRepo.Load(context.Background(), fxt.Spaces[0].ID)
		require.Error(t, errSpace, "Space should have been deleted")
		err = s.spaceRepo.CheckExists(context.Background(), fxt.Spaces[0].ID)
		require.Error(t, err, "Space should not exist")
	})
	t.Run("bad request", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.Identities(1))
		secureService, secureController := s.SecuredServiceAccountController(*fxt.Identities[0])
		// when
		emptyUsername := ""
		test.DeleteUsersBadRequest(t, secureService.Context, secureService, secureController, emptyUsername)
	})
	t.Run("user not found", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.Identities(1))
		// when
		secureService, secureController := s.SecuredServiceAccountController(*fxt.Identities[0])
		usernameAsString := uuid.NewV4().String() // will never be found.
		test.DeleteUsersNotFound(t, secureService.Context, secureService, secureController, usernameAsString)
	})
	t.Run("not authorized", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.Identities(1))
		// when
		secureService, secureController := s.SecuredController(*fxt.Identities[0])
		usernameAsString := (fxt.Identities[0].ID).String()
		test.DeleteUsersUnauthorized(t, secureService.Context, secureService, secureController, usernameAsString)
	})
}

func (s *TestUsersSuite) TestObfuscateUser() {
	// given
	t := s.T()
	user := s.createRandomUser("TestObfuscateUser")
	identity := s.createRandomIdentity(user, account.KeycloakIDP, t)
	t.Run("bad request", func(t *testing.T) {
		// given
		secureService, secureController := s.SecuredServiceAccountController(identity)
		idAsString := "bad-uuid"
		// when
		test.ObfuscateUsersBadRequest(t, secureService.Context, secureService, secureController, idAsString)
	})
	t.Run("obfuscated is ok", func(t *testing.T) {
		// given
		secureService, secureController := s.SecuredServiceAccountController(identity)
		// when
		test.ObfuscateUsersOK(t, secureService.Context, secureService, secureController, (user.ID).String())
		// then
		obfUser, err := s.userRepo.Load(context.Background(), user.ID)
		require.NoError(t, err)
		obsString := obfUser.FullName
		assert.Equal(t, len(obsString), 12)
		assert.Equal(t, obfUser.Email, obsString+"@mail.com")
		assert.Equal(t, obfUser.FullName, obsString)
		assert.Equal(t, obfUser.ImageURL, obsString)
		assert.Equal(t, obfUser.Bio, obsString)
		assert.Equal(t, obfUser.URL, obsString)
		assert.Equal(t, obfUser.Company, obsString)
		assert.Nil(t, obfUser.ContextInformation)
		obfIdentity, err := s.identityRepo.Load(context.Background(), identity.ID)
		require.NoError(t, err)
		assert.Equal(t, obfIdentity.Username, obsString)
		assert.Equal(t, obfIdentity.ProfileURL, &obsString)
	})
	t.Run("user not found", func(t *testing.T) {
		// given
		secureService, secureController := s.SecuredServiceAccountController(identity)
		idAsString := uuid.NewV4().String() // will never be found.
		// when
		test.ObfuscateUsersNotFound(t, secureService.Context, secureService, secureController, idAsString)
	})
	t.Run("unauthorized", func(t *testing.T) {
		// given
		secureService, secureController := s.SecuredController(identity)
		idAsString := (identity.ID).String()
		// when
		test.ObfuscateUsersUnauthorized(t, secureService.Context, secureService, secureController, idAsString)
	})
}

func (s *TestUsersSuite) TestUpdateUser() {
	// given
	t := s.T()
	user := s.createRandomUser("TestUpdateUser")
	identity := s.createRandomIdentity(user, account.KeycloakIDP, t)
	t.Run("unauthorized", func(t *testing.T) {
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
		test.UpdateUserAsServiceAccountUsersUnauthorized(t, secureService.Context, secureService, secureController, idAsString, updateUsersPayload)

	})
	t.Run("bad request", func(t *testing.T) {
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
		test.UpdateUserAsServiceAccountUsersBadRequest(t, secureService.Context, secureService, secureController, idAsString, updateUsersPayload)

	})
	t.Run("ok", func(t *testing.T) {
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
		test.UpdateUserAsServiceAccountUsersOK(t, secureService.Context, secureService, secureController, (identity.ID).String(), updateUsersPayload)

	})
	t.Run("not found", func(t *testing.T) {
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
		test.UpdateUserAsServiceAccountUsersNotFound(t, secureService.Context, secureService, secureController, idAsString, updateUsersPayload)
	})
}

func (s *TestUsersSuite) TestCreateUser() {
	// given
	t := s.T()
	user := s.createRandomUserObject("TestCreateUser")
	identity := s.createRandomIdentityObject(user, "KC")
	t.Run("ok", func(t *testing.T) {
		user.ContextInformation = map[string]interface{}{
			"last_visited": "yesterday",
			"space":        "3d6dab8d-f204-42e8-ab29-cdb1c93130ad",
			"rate":         100.00,
			"count":        3,
		}
		secureService, secureController := s.SecuredServiceAccountController(identity)

		// when
		createUserPayload := createCreateUsersAsServiceAccountPayload(&user.Email, &user.FullName, &user.Bio, &user.ImageURL, &user.URL, &user.Company, &identity.Username, &identity.RegistrationCompleted, user.ContextInformation, user.ID.String())
		test.CreateUserAsServiceAccountUsersOK(t, secureService.Context, secureService, secureController, identity.ID.String(), createUserPayload)
	})
	t.Run("unauthorized", func(t *testing.T) {
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
		test.CreateUserAsServiceAccountUsersUnauthorized(t, secureService.Context, secureService, secureController, identityId.String(), createUserPayload)
	})
	t.Run("bad request", func(t *testing.T) {
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
		test.CreateUserAsServiceAccountUsersBadRequest(t, secureService.Context, secureService, secureController, "invalid-uuid", createUserPayload)
	})
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
		UserID:       id.NullUUID{UUID: user.ID, Valid: true},
	}
	return identity
}

func (s *TestUsersSuite) createRandomIdentity(user account.User, providerType string, t *testing.T) account.Identity {
	profile := "foobarforupdate.com/" + uuid.NewV4().String() + "/" + user.ID.String()
	identity := account.Identity{
		Username:     "TestUpdateUserIntegration123" + uuid.NewV4().String(),
		ProviderType: providerType,
		ProfileURL:   &profile,
		User:         user,
		UserID:       id.NullUUID{UUID: user.ID, Valid: true},
	}
	err := s.identityRepo.Create(context.Background(), &identity)
	require.NoError(t, err)
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
