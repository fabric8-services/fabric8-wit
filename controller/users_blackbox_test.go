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
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func createController(t *testing.T) (*UsersController, application.DB) {
	svc := goa.New("test")
	app := gormapplication.NewGormDB(DB)
	controller := NewUsersController(svc, app)
	assert.NotNil(t, controller)
	return controller, app
}

func TestShowUserOK(t *testing.T) {
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
	profile := "foobar.com/" + user.ID.String()
	identity := account.Identity{
		Username:     "TestUserIntegration123",
		ProviderType: account.KeycloakIDP,
		ID:           uuid.NewV4(),
		User:         user,
		UserID:       account.NullUUID{UUID: user.ID, Valid: true},
		ProfileURL:   &profile,
	}

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
