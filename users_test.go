package main_test

import (
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestShowUser(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	svc := goa.New("test")
	controller := NewUsersController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(t, controller)

	ctx := context.Background()
	userRepo := account.NewUserRepository(DB)
	identityRepo := account.NewIdentityRepository(DB)
	identity := account.Identity{
		FullName: "Test User Integration 123",
		ImageURL: "http://images.com/42",
	}
	email := "primary@example.com"

	err := identityRepo.Create(ctx, &identity)
	if err != nil {
		t.Fatal(err)
	}
	user1 := account.User{Email: email, Identity: identity}
	err = userRepo.Create(ctx, &user1)
	if err != nil {
		t.Fatal(err)
	}

	_, result := test.ShowUsersOK(t, nil, nil, controller, identity.ID.String())
	assert.Equal(t, identity.ID.String(), *result.Data.ID)
	assert.Equal(t, identity.FullName, *result.Data.Attributes.FullName)
	assert.Equal(t, identity.ImageURL, *result.Data.Attributes.ImageURL)
}
