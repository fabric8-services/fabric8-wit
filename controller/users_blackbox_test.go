package controller_test

import (
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func createController(t *testing.T) (*UsersController, application.DB) {
	svc := goa.New("test")
	app := gormapplication.NewGormDB(DB)
	controller := NewUsersController(svc, app)
	assert.NotNil(t, controller)
	return controller, app
}

func createSecureController(t *testing.T, identity account.Identity) (*UsersController, *goa.Service) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("Users-Service", almtoken.NewManagerWithPrivateKey(priv), identity)
	app := gormapplication.NewGormDB(DB)
	controller := NewUsersController(svc, app)
	assert.NotNil(t, controller)
	return controller, svc
}

func TestShowUserOK(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	controller, app := createController(t)

	ctx := context.Background()
	userRepo := app.Users()
	identityRepo := app.Identities()

	user := createRandomUser()
	err := userRepo.Create(ctx, &user)
	if err != nil {
		t.Fatal(err)
	}

	identity := createRandomIdentity(user)
	err = identityRepo.Create(ctx, &identity)
	if err != nil {
		t.Fatal(err)
	}

	_, result := test.ShowUsersOK(t, nil, nil, controller, identity.ID.String())
	assert.Equal(t, identity.ID.String(), *result.Data.ID)
	assert.Equal(t, user.FullName, *result.Data.Attributes.FullName)
	assert.Equal(t, user.ImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(t, identity.ProviderType, *result.Data.Attributes.ProviderType)
	assert.Equal(t, identity.Username, *result.Data.Attributes.Username)
}

func createRandomUser() account.User {
	user := account.User{
		Email:    uuid.NewV4().String() + "primaryForUpdat7e@example.com",
		FullName: "A test user",
		ImageURL: "someURLForUpdate",
		ID:       uuid.NewV4(),
	}
	return user
}

func createRandomIdentity(user account.User) account.Identity {
	profile := "foobarforupdate.com/" + user.ID.String()
	identity := account.Identity{
		Username:     "TestUpdateUserIntegration123" + uuid.NewV4().String(),
		ProviderType: account.KeycloakIDP,
		User:         user,
		UserID:       account.NullUUID{UUID: user.ID, Valid: true},
		ProfileURL:   &profile,
	}
	return identity
}

func TestUpdateUserOK(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	controller, app := createController(t)

	ctx := context.Background()
	userRepo := app.Users()
	identityRepo := app.Identities()

	user := createRandomUser()
	err := userRepo.Create(ctx, &user)
	if err != nil {
		t.Fatal(err)
	}

	identity := createRandomIdentity(user)
	err = identityRepo.Create(ctx, &identity)
	if err != nil {
		t.Fatal(err)
	}

	_, result := test.ShowUsersOK(t, nil, nil, controller, identity.ID.String())
	assert.Equal(t, identity.ID.String(), *result.Data.ID)
	assert.Equal(t, user.FullName, *result.Data.Attributes.FullName)
	assert.Equal(t, user.ImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(t, identity.ProviderType, *result.Data.Attributes.ProviderType)
	assert.Equal(t, identity.Username, *result.Data.Attributes.Username)

	newEmail := "updated@email.com"
	newFullName := "newFull Name"
	newImageURL := "http://new.image.io/imageurl"
	newBio := "new bio"
	newProfileURL := "http://new.profile.url/url"

	secureController, secureService := createSecureController(t, identity)
	updateUsersPayload := createUpdateUsersPayload(&newEmail, &newFullName, &newBio, &newImageURL, &newProfileURL)
	_, result = test.UpdateUsersOK(t, secureService.Context, secureService, secureController, updateUsersPayload)
	require.NotNil(t, result)

	// let's fetch it and validate
	_, result = test.ShowUsersOK(t, nil, nil, controller, identity.ID.String())
	require.NotNil(t, result)
	assert.Equal(t, identity.ID.String(), *result.Data.ID)
	assert.Equal(t, newFullName, *result.Data.Attributes.FullName)
	assert.Equal(t, newImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(t, newBio, *result.Data.Attributes.Bio)
	assert.Equal(t, newProfileURL, *result.Data.Attributes.URL)

}

func TestUpdateUserUnauthorized(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	controller, app := createController(t)

	ctx := context.Background()
	userRepo := app.Users()
	identityRepo := app.Identities()

	user := createRandomUser()
	err := userRepo.Create(ctx, &user)
	if err != nil {
		t.Fatal(err)
	}

	identity := createRandomIdentity(user)
	err = identityRepo.Create(ctx, &identity)
	if err != nil {
		t.Fatal(err)
	}

	_, result := test.ShowUsersOK(t, nil, nil, controller, identity.ID.String())
	assert.Equal(t, identity.ID.String(), *result.Data.ID)
	assert.Equal(t, user.FullName, *result.Data.Attributes.FullName)
	assert.Equal(t, user.ImageURL, *result.Data.Attributes.ImageURL)
	assert.Equal(t, identity.ProviderType, *result.Data.Attributes.ProviderType)
	assert.Equal(t, identity.Username, *result.Data.Attributes.Username)

	newEmail := "updated@email.com"
	newFullName := "newFull Name"
	newImageURL := "http://new.image.io/imageurl"
	newBio := "new bio"
	newProfileURL := "http://new.profile.url/url"

	//secureController, secureService := createSecureController(t, identity)
	updateUsersPayload := createUpdateUsersPayload(&newEmail, &newFullName, &newBio, &newImageURL, &newProfileURL)
	test.UpdateUsersUnauthorized(t, ctx, nil, controller, updateUsersPayload)
}

func TestListUserOK(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	controller, app := createController(t)

	ctx := context.Background()
	userRepo := app.Users()
	identityRepo := app.Identities()

	user := account.User{
		Email:    "primary@example.com",
		FullName: "A test user",
		ImageURL: "someURL",
	}
	err := userRepo.Create(ctx, &user)
	if err != nil {
		t.Fatal(err)
	}
	profile := "github.com/" + uuid.NewV4().String()
	identityGitHub := account.Identity{
		Username:     "TestUserIntegration2",
		ProviderType: "github-test",
		ID:           uuid.NewV4(),
		User:         user,
		UserID:       account.NullUUID{UUID: user.ID, Valid: true},
		ProfileURL:   &profile,
	}
	err = identityRepo.Create(ctx, &identityGitHub)
	if err != nil {
		t.Fatal(err)
	}

	identity := account.Identity{
		Username:     "TestUserIntegration1",
		ProviderType: account.KeycloakIDP,
		ID:           uuid.NewV4(),
		User:         user,
		UserID:       account.NullUUID{UUID: user.ID, Valid: true},
	}
	err = identityRepo.Create(ctx, &identity)
	if err != nil {
		t.Fatal(err)
	}

	user2 := account.User{
		Email:    "primary2@example.com",
		FullName: "A test user 2",
		ImageURL: "someURL",
	}
	err = userRepo.Create(ctx, &user2)
	if err != nil {
		t.Fatal(err)
	}
	identity2 := account.Identity{
		Username:     "TestUserIntegration1",
		ProviderType: account.KeycloakIDP,
		ID:           uuid.NewV4(),
		User:         user2,
		UserID:       account.NullUUID{UUID: user2.ID, Valid: true},
	}
	err = identityRepo.Create(ctx, &identity2)
	if err != nil {
		t.Fatal(err)
	}

	_, result := test.ListUsersOK(t, nil, nil, controller)

	assertUser(t, findUser(identity.ID, result.Data), user, identity)
	assertUser(t, findUser(identity2.ID, result.Data), user2, identity2)
}

func findUser(id uuid.UUID, users []*app.IdentityData) *app.IdentityData {
	for _, user := range users {
		if *user.ID == id.String() {
			return user
		}
	}
	return nil
}

func assertUser(t *testing.T, actual *app.IdentityData, expectedUser account.User, expectedIdentity account.Identity) {
	assert.Equal(t, expectedIdentity.Username, *actual.Attributes.Username)
	assert.Equal(t, expectedIdentity.ProviderType, *actual.Attributes.ProviderType)
	assert.Equal(t, expectedUser.FullName, *actual.Attributes.FullName)
	assert.Equal(t, expectedUser.ImageURL, *actual.Attributes.ImageURL)
	assert.Equal(t, expectedUser.Email, *actual.Attributes.Email)
}

func createUpdateUsersPayload(email, fullName, bio, imageURL, profileURL *string) *app.UpdateUsersPayload {
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
