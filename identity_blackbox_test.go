package main_test

import (
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestListIdentities(t *testing.T) {
	resource.Require(t, resource.Database)
	DB.Unscoped().Delete(&account.Identity{})
	pub, _ := almtoken.ParsePublicKey(configuration.GetTokenPublicKey())
	priv, _ := almtoken.ParsePrivateKey(configuration.GetTokenPrivateKey())
	service := testsupport.ServiceAsUser("TestListIdentities-Service", almtoken.NewManager(pub, priv), account.TestIdentity)

	identityController := NewIdentityController(service, gormapplication.NewGormDB(DB))
	_, ic := test.ListIdentityOK(t, service.Context, service, identityController)
	require.NotNil(t, ic)

	ctx := context.Background()
	identityRepo := account.NewIdentityRepository(DB)
	identity := account.Identity{
		FullName: "Test User",
		ImageURL: "http://images.com/123",
	}

	err := identityRepo.Create(ctx, &identity)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		DB.Unscoped().Delete(&identity)
	}()

	_, ic2 := test.ListIdentityOK(t, service.Context, service, identityController)
	require.NotNil(t, ic2)
	assert.Equal(t, identity.FullName, *ic2.Data[0].Attributes.FullName)
	assert.Equal(t, identity.ImageURL, *ic2.Data[0].Attributes.ImageURL)

	identity2 := account.Identity{
		FullName: "Test User 2",
		ImageURL: "http://images.com/1234",
	}

	err = identityRepo.Create(ctx, &identity2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		DB.Unscoped().Delete(&identity2)
	}()

	_, ic3 := test.ListIdentityOK(t, service.Context, service, identityController)
	require.NotNil(t, ic3)
	assert.Equal(t, identity.FullName, *ic3.Data[0].Attributes.FullName)
	assert.Equal(t, identity.ImageURL, *ic3.Data[0].Attributes.ImageURL)

	assert.Equal(t, identity2.FullName, *ic3.Data[1].Attributes.FullName)
	assert.Equal(t, identity2.ImageURL, *ic3.Data[1].Attributes.ImageURL)
}
