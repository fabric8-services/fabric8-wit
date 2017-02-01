package main_test

import (
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestListIdentities(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()

	service := goa.New("Test-Identities")
	identityController := NewIdentityController(service, gormapplication.NewGormDB(DB))
	_, ic := test.ListIdentityOK(t, service.Context, service, identityController)
	require.NotNil(t, ic)

	numberOfCurrentIdent := len(ic.Data)

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

	_, ic2 := test.ListIdentityOK(t, service.Context, service, identityController)
	require.NotNil(t, ic2)

	assert.Equal(t, numberOfCurrentIdent+1, len(ic2.Data))

	assertIdent(t, findIdent(identity.ID, ic2.Data), identity.FullName, identity.ImageURL)

	identity2 := account.Identity{
		FullName: "Test User 2",
		ImageURL: "http://images.com/1234",
	}

	err = identityRepo.Create(ctx, &identity2)
	if err != nil {
		t.Fatal(err)
	}

	_, ic3 := test.ListIdentityOK(t, service.Context, service, identityController)
	require.NotNil(t, ic3)
	assert.Equal(t, numberOfCurrentIdent+2, len(ic3.Data))

	assertIdent(t, findIdent(identity.ID, ic3.Data), identity.FullName, identity.ImageURL)
	assertIdent(t, findIdent(identity2.ID, ic3.Data), identity2.FullName, identity2.ImageURL)
}

func findIdent(id uuid.UUID, idents []*app.IdentityData) *app.IdentityData {
	for _, ident := range idents {
		if *ident.ID == id.String() {
			return ident
		}
	}
	return nil
}

func assertIdent(t *testing.T, ident *app.IdentityData, fullName, imageURL string) {
	assert.Equal(t, fullName, *ident.Attributes.FullName)
	assert.Equal(t, imageURL, *ident.Attributes.ImageURL)
}
