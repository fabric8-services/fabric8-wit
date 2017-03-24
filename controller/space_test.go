package controller_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/auth"
	"github.com/almighty/almighty-core/configuration"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
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
)

var spaceConfiguration *configuration.ConfigurationData

type DummyResourceManager struct {
}

func (m *DummyResourceManager) CreateResource(ctx context.Context, request *goa.RequestData, name string, rType string, uri *string, scopes *[]string, userID string, policyName string) (*auth.Resource, error) {
	return &auth.Resource{ResourceID: uuid.NewV4().String(), PermissionID: uuid.NewV4().String(), PolicyID: uuid.NewV4().String()}, nil
}

func (m *DummyResourceManager) DeleteResource(ctx context.Context, request *goa.RequestData, resource auth.Resource) error {
	return nil
}

func init() {
	var err error
	spaceConfiguration, err = configuration.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

type TestSpaceREST struct {
	gormtestsupport.DBTestSuite
	db    *gormapplication.GormDB
	clean func()
}

func TestRunProjectREST(t *testing.T) {
	suite.Run(t, &TestSpaceREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestSpaceREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestSpaceREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSpaceREST) SecuredController(identity account.Identity) (*goa.Service, *SpaceController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Space-Service", almtoken.NewManagerWithPrivateKey(priv), identity)
	return svc, NewSpaceController(svc, rest.db, spaceConfiguration, &DummyResourceManager{})
}

func (rest *TestSpaceREST) UnSecuredController() (*goa.Service, *SpaceController) {
	svc := goa.New("Space-Service")
	return svc, NewSpaceController(svc, rest.db, spaceConfiguration, &DummyResourceManager{})
}

func (rest *TestSpaceREST) TestFailCreateSpaceUnsecure() {
	t := rest.T()
	resource.Require(t, resource.Database)

	p := minimumRequiredCreateSpace()

	svc, ctrl := rest.UnSecuredController()
	test.CreateSpaceUnauthorized(t, svc.Context, svc, ctrl, p)
}

func (rest *TestSpaceREST) TestFailCreateSpaceMissingName() {
	t := rest.T()
	resource.Require(t, resource.Database)

	p := minimumRequiredCreateSpace()

	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	test.CreateSpaceBadRequest(t, svc.Context, svc, ctrl, p)
}

func (rest *TestSpaceREST) TestSuccessCreateSpace() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 24"

	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name

	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
	assert.NotNil(t, created.Data)
	assert.NotNil(t, created.Data.Attributes)
	assert.NotNil(t, created.Data.Attributes.CreatedAt)
	assert.NotNil(t, created.Data.Attributes.UpdatedAt)
	assert.NotNil(t, created.Data.Attributes.Name)
	assert.Equal(t, name, *created.Data.Attributes.Name)
	assert.NotNil(t, created.Data.Links)
	assert.NotNil(t, created.Data.Links.Self)
}

func (rest *TestSpaceREST) SecuredSpaceAreaController(identity account.Identity) (*goa.Service, *SpaceAreasController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	svc := testsupport.ServiceAsUser("Area-Service", almtoken.NewManager(pub), identity)
	return svc, NewSpaceAreasController(svc, rest.db)
}

func (rest *TestSpaceREST) TestSuccessCreateSpaceAndDefaultArea() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test24ForSpaceAndArea2"

	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name

	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
	require.NotNil(t, created.Data)

	spaceAreaSvc, spaceAreaCtrl := rest.SecuredSpaceAreaController(testsupport.TestIdentity)
	createdID := created.Data.ID.String()
	_, areaList := test.ListSpaceAreasOK(t, spaceAreaSvc.Context, spaceAreaSvc, spaceAreaCtrl, createdID)

	// only 1 default gets created.
	assert.Len(t, areaList.Data, 1)
	assert.Equal(t, name, *areaList.Data[0].Attributes.Name)

}

func (rest *TestSpaceREST) TestSuccessCreateSpaceWithDescription() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 24"
	description := "Space for Test 24"

	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	p.Data.Attributes.Description = &description

	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
	assert.NotNil(t, created.Data)
	assert.NotNil(t, created.Data.Attributes)
	assert.NotNil(t, created.Data.Attributes.CreatedAt)
	assert.NotNil(t, created.Data.Attributes.UpdatedAt)
	assert.NotNil(t, created.Data.Attributes.Name)
	assert.Equal(t, name, *created.Data.Attributes.Name)
	assert.NotNil(t, created.Data.Attributes.Description)
	assert.Equal(t, description, *created.Data.Attributes.Description)
	assert.NotNil(t, created.Data.Links)
	assert.NotNil(t, created.Data.Links.Self)
}

func (rest *TestSpaceREST) TestSuccessUpdateProject() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 25"
	description := "Space for Test 25"
	newName := "Test 26"
	newDescription := "Space for Test 25"

	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	p.Data.Attributes.Description = &description

	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)

	u := minimumRequiredUpdateSpace()
	u.Data.ID = created.Data.ID
	u.Data.Attributes.Version = created.Data.Attributes.Version
	u.Data.Attributes.Name = &newName
	u.Data.Attributes.Description = &newDescription

	_, updated := test.UpdateSpaceOK(t, svc.Context, svc, ctrl, created.Data.ID.String(), u)
	assert.Equal(t, newName, *updated.Data.Attributes.Name)
	assert.Equal(t, newDescription, *updated.Data.Attributes.Description)
}

func (rest *TestSpaceREST) TestFailUpdateProjectDifferentOwner() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 25"
	description := "Space for Test 25"
	newName := "Test 26"
	newDescription := "Space for Test 25"

	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	p.Data.Attributes.Description = &description

	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)

	u := minimumRequiredUpdateSpace()
	u.Data.ID = created.Data.ID
	u.Data.Attributes.Version = created.Data.Attributes.Version
	u.Data.Attributes.Name = &newName
	u.Data.Attributes.Description = &newDescription

	svc2, ctrl2 := rest.SecuredController(testsupport.TestIdentity2)
	_, errors := test.UpdateSpaceForbidden(t, svc2.Context, svc2, ctrl2, created.Data.ID.String(), u)
	assert.NotEmpty(t, errors.Errors)
	assert.Contains(t, errors.Errors[0].Detail, "User is not the space owner")
}

func (rest *TestSpaceREST) TestFailUpdateProjectUnSecure() {
	t := rest.T()
	resource.Require(t, resource.Database)

	u := minimumRequiredUpdateSpace()

	svc, ctrl := rest.UnSecuredController()
	test.UpdateSpaceUnauthorized(t, svc.Context, svc, ctrl, uuid.NewV4().String(), u)
}

func (rest *TestSpaceREST) TestFailUpdateSpaceNotFound() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "test"
	version := 0
	id := uuid.NewV4()

	u := minimumRequiredUpdateSpace()
	u.Data.Attributes.Name = &name
	u.Data.Attributes.Version = &version
	u.Data.ID = &id

	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	test.UpdateSpaceNotFound(t, svc.Context, svc, ctrl, id.String(), u)
}

func (rest *TestSpaceREST) TestFailUpdateSpaceMissingName() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 25"

	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name

	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)

	u := minimumRequiredUpdateSpace()
	u.Data.ID = created.Data.ID
	u.Data.Attributes.Version = created.Data.Attributes.Version

	test.UpdateSpaceBadRequest(t, svc.Context, svc, ctrl, created.Data.ID.String(), u)
}

func (rest *TestSpaceREST) TestFailUpdateSpaceMissingVersion() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 25"
	newName := "Test 26"

	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name

	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)

	u := minimumRequiredUpdateSpace()
	u.Data.ID = created.Data.ID
	u.Data.Attributes.Name = &newName

	test.UpdateSpaceBadRequest(t, svc.Context, svc, ctrl, created.Data.ID.String(), u)
}

func (rest *TestSpaceREST) TestSuccessShowProject() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 27"
	description := "Space for Test 27"
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	p.Data.Attributes.Description = &description

	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)

	_, fetched := test.ShowSpaceOK(t, svc.Context, svc, ctrl, created.Data.ID.String())
	assert.Equal(t, created.Data.ID, fetched.Data.ID)
	assert.Equal(t, *created.Data.Attributes.Name, *fetched.Data.Attributes.Name)
	assert.Equal(t, *created.Data.Attributes.Description, *fetched.Data.Attributes.Description)
	assert.Equal(t, *created.Data.Attributes.Version, *fetched.Data.Attributes.Version)

	// verify list-WI URL exists in Relationships.Links
	require.NotNil(t, *fetched.Data.Relationships.Workitems)
	require.NotNil(t, *fetched.Data.Relationships.Workitems.Links)
	require.NotNil(t, *fetched.Data.Relationships.Workitems.Links.Related)
	subStringWI := fmt.Sprintf("/%s/workitems", created.Data.ID.String())
	assert.Contains(t, *fetched.Data.Relationships.Workitems.Links.Related, subStringWI)

	// verify list-WIT URL exists in Relationships.Links
	require.NotNil(t, *fetched.Data.Links)
	require.NotNil(t, fetched.Data.Links.WorkItemTypes)
	subStringWIL := fmt.Sprintf("/%s/workitemtypes", created.Data.ID.String())
	assert.Contains(t, *fetched.Data.Links.WorkItemTypes, subStringWIL)

	// verify list-WILT URL exists in Relationships.Links
	require.NotNil(t, *fetched.Data.Relationships.Workitemlinktypes)
	require.NotNil(t, *fetched.Data.Relationships.Workitemlinktypes.Links)
	require.NotNil(t, *fetched.Data.Relationships.Workitemlinktypes.Links.Related)
	subStringWILT := fmt.Sprintf("/%s/workitemlinktypes", created.Data.ID.String())
	assert.Contains(t, *fetched.Data.Relationships.Workitemlinktypes.Links.Related, subStringWILT)
}

func (rest *TestSpaceREST) TestFailShowSpaceNotFound() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.UnSecuredController()
	test.ShowSpaceNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String())
}

func (rest *TestSpaceREST) TestFailShowSpaceNotFoundBadID() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.UnSecuredController()
	test.ShowSpaceNotFound(t, svc.Context, svc, ctrl, "asfasfsaf")
}

func (rest *TestSpaceREST) TestSuccessListSpaces() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 24"

	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name

	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)

	_, list := test.ListSpaceOK(t, svc.Context, svc, ctrl, nil, nil)
	assert.True(t, len(list.Data) > 0)
	for _, spc := range list.Data {
		// Test that it contains the right link for backlog items
		subStringBacklogUrl := fmt.Sprintf("/%s/backlog", spc.ID.String())
		assert.Contains(t, *spc.Links.Backlog, subStringBacklogUrl)

		// Test that it contains the right relationship values
		subString := fmt.Sprintf("/%s/iterations", spc.ID.String())
		assert.Contains(t, *spc.Relationships.Iterations.Links.Related, subString)

		subStringAreaUrl := fmt.Sprintf("/%s/areas", spc.ID.String())
		assert.Contains(t, *spc.Relationships.Areas.Links.Related, subStringAreaUrl)
	}
}

func minimumRequiredCreateSpace() *app.CreateSpacePayload {
	return &app.CreateSpacePayload{
		Data: &app.Space{
			Type:       "spaces",
			Attributes: &app.SpaceAttributes{},
		},
	}
}

func CreateSpacePayload(name, description string) *app.CreateSpacePayload {
	return &app.CreateSpacePayload{
		Data: &app.Space{
			Type: "spaces",
			Attributes: &app.SpaceAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}
}

func minimumRequiredUpdateSpace() *app.UpdateSpacePayload {
	return &app.UpdateSpacePayload{
		Data: &app.Space{
			Type:       "spaces",
			Attributes: &app.SpaceAttributes{},
		},
	}
}
