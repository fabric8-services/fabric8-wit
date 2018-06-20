package controller_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestNamedSpaceREST struct {
	gormtestsupport.DBTestSuite
	db      *gormapplication.GormDB
	testDir string
}

func TestRunNamedSpacesREST(t *testing.T) {
	suite.Run(t, &TestNamedSpaceREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestNamedSpaceREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.testDir = filepath.Join("test-files", "namedspaces")
}

func (rest *TestNamedSpaceREST) SecuredNamedSpaceController(identity account.Identity) (*goa.Service, *NamedspacesController) {
	svc := testsupport.ServiceAsUser("NamedSpace-Service", identity)
	return svc, NewNamedspacesController(svc, rest.db)
}

func (rest *TestNamedSpaceREST) UnSecuredNamedSpaceController() (*goa.Service, *NamedspacesController) {
	svc := goa.New("NamedSpace-Service")
	return svc, NewNamedspacesController(svc, rest.db)
}

func (rest *TestNamedSpaceREST) SecuredSpaceController() (*goa.Service, *SpaceController) {
	svc := testsupport.ServiceAsUser("Space-Service", testsupport.TestIdentity)
	return svc, NewSpaceController(svc, rest.db, rest.Configuration, &DummyResourceManager{})
}

func (rest *TestNamedSpaceREST) UnSecuredSpaceController() (*goa.Service, *SpaceController) {
	svc := goa.New("Space-Service")
	return svc, NewSpaceController(svc, rest.db, rest.Configuration, &DummyResourceManager{})
}

func (rest *TestNamedSpaceREST) TestSuccessQuerySpace() {
	t := rest.T()
	resource.Require(t, resource.Database)

	spaceSvc, spaceCtrl := rest.SecuredSpaceController()

	identityRepo := account.NewIdentityRepository(rest.DB)
	identity := testsupport.TestIdentity
	identity.ProviderType = account.KeycloakIDP
	err := identityRepo.Create(spaceSvc.Context, &identity)
	if err != nil {
		assert.Fail(t, "Failed to create an identity")
	}

	name := testsupport.CreateRandomValidTestName("Test 24")

	p := newCreateSpacePayload(&name, nil)

	_, created := test.CreateSpaceCreated(t, spaceSvc.Context, spaceSvc, spaceCtrl, p)
	assert.NotNil(t, created.Data)
	assert.NotNil(t, created.Data.Attributes)
	assert.NotNil(t, created.Data.Attributes.CreatedAt)
	assert.NotNil(t, created.Data.Attributes.UpdatedAt)
	assert.NotNil(t, created.Data.Attributes.Name)
	assert.Equal(t, name, *created.Data.Attributes.Name)
	assert.NotNil(t, created.Data.Links)
	assert.NotNil(t, created.Data.Links.Self)

	namedSpaceSvc, namedSpacectrl := rest.SecuredNamedSpaceController(testsupport.TestIdentity)
	_, namedspace := test.ShowNamedspacesOK(t, namedSpaceSvc.Context, namedSpaceSvc, namedSpacectrl, testsupport.TestIdentity.Username, name)
	assert.NotNil(t, namedspace)
	assert.Equal(t, created.Data.Attributes.Name, namedspace.Data.Attributes.Name)
	assert.Equal(t, created.Data.Attributes.Description, namedspace.Data.Attributes.Description)
	assert.Equal(t, created.Data.Links.Self, namedspace.Data.Links.Self)

	// test that show namedspaces operation is not case sensitive for the space name
	_, namedspace = test.ShowNamedspacesOK(t, namedSpaceSvc.Context, namedSpaceSvc, namedSpacectrl, testsupport.TestIdentity.Username, strings.ToLower(name))
	assert.NotNil(t, namedspace)
	assert.Equal(t, created.Data.Attributes.Name, namedspace.Data.Attributes.Name)
}

func (rest *TestNamedSpaceREST) TestSuccessListSpaces() {
	t := rest.T()
	resource.Require(t, resource.Database)

	spaceSvc, spaceCtrl := rest.SecuredSpaceController()

	identityRepo := account.NewIdentityRepository(rest.DB)
	identity := testsupport.TestIdentity
	identity.ProviderType = account.KeycloakIDP
	err := identityRepo.Create(spaceSvc.Context, &identity)
	if err != nil {
		assert.Fail(t, "Failed to create an identity")
	}

	name := testsupport.CreateRandomValidTestName("Test 24")

	p := newCreateSpacePayload(&name, nil)

	_, created := test.CreateSpaceCreated(t, spaceSvc.Context, spaceSvc, spaceCtrl, p)
	assert.NotNil(t, created.Data)
	assert.NotNil(t, created.Data.Attributes)
	assert.NotNil(t, created.Data.Attributes.CreatedAt)
	assert.NotNil(t, created.Data.Attributes.UpdatedAt)
	assert.NotNil(t, created.Data.Attributes.Name)
	assert.Equal(t, name, *created.Data.Attributes.Name)
	assert.NotNil(t, created.Data.Links)
	assert.NotNil(t, created.Data.Links.Self)

	collabSpaceSvc, collabSpacectrl := rest.SecuredNamedSpaceController(testsupport.TestIdentity)
	_, collabspaces := test.ListNamedspacesOK(t, collabSpaceSvc.Context, collabSpaceSvc, collabSpacectrl, testsupport.TestIdentity.Username, nil, nil, nil)
	assert.True(t, len(collabspaces.Data) > 0)
	assert.Equal(t, created.Data.Attributes.Name, collabspaces.Data[0].Attributes.Name)
	assert.Equal(t, created.Data.Attributes.Description, collabspaces.Data[0].Attributes.Description)
	assert.Equal(t, created.Data.Links.Self, collabspaces.Data[0].Links.Self)
}

func (rest *TestNamedSpaceREST) TestShow() {
	rest.T().Run("ok", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, rest.DB, tf.Spaces(1), tf.Identities(2))

		namedSpaceSvc, namedSpacectrl := rest.SecuredNamedSpaceController(*fxt.Identities[0])
		res, namedspace := test.ShowNamedspacesOK(t, namedSpaceSvc.Context, namedSpaceSvc, namedSpacectrl, fxt.Identities[0].Username, fxt.Spaces[0].Name)

		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok.payload.golden.json"), namedspace)
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok.headers.golden.json"), res.Header())

		assert.NotNil(t, namedspace)
		assert.Equal(t, &fxt.Spaces[0].Name, namedspace.Data.Attributes.Name)
		assert.Equal(t, &fxt.Spaces[0].Description, namedspace.Data.Attributes.Description)
	})
}

func (rest *TestNamedSpaceREST) TestSuccessSortListSpaces() {
	t := rest.T()
	resource.Require(t, resource.Database)

	// names of the spaces to be created
	inputSpaces := []string{
		"Test 24",
		"Test 22",
		"Test 23",
	}
	// randominze the space names so that they don't clash
	random := testsupport.CreateRandomValidTestName("")
	for i := range inputSpaces {
		inputSpaces[i] += random
	}

	// create actual spaces with those names
	fxt := tf.NewTestFixture(t, rest.DB, tf.Spaces(len(inputSpaces),
		func(fxt *tf.TestFixture, idx int) error {
			fxt.Spaces[idx].Name = inputSpaces[idx]
			return nil
		}),
	)
	identity := *fxt.Identities[0]

	// Now try to list the data in various circumstances
	tests := []struct {
		name      string
		sortParam *string
		want      []string
	}{
		{
			name: "no sort param given",
			want: []string{
				"Test 23",
				"Test 22",
				"Test 24",
			},
		},
		{
			name:      "sort in ascending using name",
			sortParam: ptr.String("name"),
			want: []string{
				"Test 22",
				"Test 23",
				"Test 24",
			},
		},
		{
			name:      "sort in descending using name",
			sortParam: ptr.String("-name"),
			want: []string{
				"Test 24",
				"Test 23",
				"Test 22",
			},
		},
		{
			name:      "sort in ascending using created at",
			sortParam: ptr.String("created"),
			want: []string{
				"Test 24",
				"Test 22",
				"Test 23",
			},
		},
		{
			name:      "sort in descending using created at",
			sortParam: ptr.String("-created"),
			want: []string{
				"Test 23",
				"Test 22",
				"Test 24",
			},
		},
		{
			name:      "sort in ascending using updated at",
			sortParam: ptr.String("updated"),
			want: []string{
				"Test 24",
				"Test 22",
				"Test 23",
			},
		},
		{
			name:      "sort in descending using updated at",
			sortParam: ptr.String("-updated"),
			want: []string{
				"Test 23",
				"Test 22",
				"Test 24",
			},
		},
	}

	svc, ctrl := rest.SecuredNamedSpaceController(identity)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// list all the spaces created by fixture helper using the
			// code that we have written
			_, spaces := test.ListNamedspacesOK(t, svc.Context, svc, ctrl,
				identity.Username, nil, nil, tt.sortParam)
			require.True(t, len(spaces.Data) == len(tt.want))

			for i, sn := range tt.want {
				// since the the space names are randomized but we do know the prefix
				assert.True(t, strings.HasPrefix(*spaces.Data[i].Attributes.Name, sn))
			}
		})
	}
}
