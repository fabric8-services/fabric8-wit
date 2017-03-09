package controller_test

import (
	"context"
	"fmt"
	"net/http"
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
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var spaceConfiguration *configuration.ConfigurationData

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
	return svc, NewSpaceController(svc, rest.db, spaceConfiguration)
}

func (rest *TestSpaceREST) UnSecuredController() (*goa.Service, *SpaceController) {
	svc := goa.New("Space-Service")
	return svc, NewSpaceController(svc, rest.db, spaceConfiguration)
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

	svc, ctrl := rest.SecuredController(initTestIdenity(rest.DB))
	test.CreateSpaceBadRequest(t, svc.Context, svc, ctrl, p)
}

func (rest *TestSpaceREST) TestSuccessCreateSpace() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 24"

	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name

	svc, ctrl := rest.SecuredController(initTestIdenity(rest.DB))
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

	svc, ctrl := rest.SecuredController(initTestIdenity(rest.DB))
	_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)
	require.NotNil(t, created.Data)

	spaceAreaSvc, spaceAreaCtrl := rest.SecuredSpaceAreaController(initTestIdenity(rest.DB))
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

	svc, ctrl := rest.SecuredController(initTestIdenity(rest.DB))
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

	svc, ctrl := rest.SecuredController(initTestIdenity(rest.DB))
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

	svc, ctrl := rest.SecuredController(initTestIdenity(rest.DB))
	_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)

	u := minimumRequiredUpdateSpace()
	u.Data.ID = created.Data.ID
	u.Data.Attributes.Version = created.Data.Attributes.Version
	u.Data.Attributes.Name = &newName
	u.Data.Attributes.Description = &newDescription

	svc2, ctrl2 := rest.SecuredController(initTestIdenity2(rest.DB))
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

	svc, ctrl := rest.SecuredController(initTestIdenity(rest.DB))
	test.UpdateSpaceNotFound(t, svc.Context, svc, ctrl, id.String(), u)
}

func (rest *TestSpaceREST) TestFailUpdateSpaceMissingName() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 25"

	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name

	svc, ctrl := rest.SecuredController(initTestIdenity(rest.DB))
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

	svc, ctrl := rest.SecuredController(initTestIdenity(rest.DB))
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

	svc, ctrl := rest.SecuredController(initTestIdenity(rest.DB))
	_, created := test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)

	_, fetched := test.ShowSpaceOK(t, svc.Context, svc, ctrl, created.Data.ID.String())
	assert.Equal(t, created.Data.ID, fetched.Data.ID)
	assert.Equal(t, *created.Data.Attributes.Name, *fetched.Data.Attributes.Name)
	assert.Equal(t, *created.Data.Attributes.Description, *fetched.Data.Attributes.Description)
	assert.Equal(t, *created.Data.Attributes.Version, *fetched.Data.Attributes.Version)
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

	svc, ctrl := rest.SecuredController(initTestIdenity(rest.DB))
	test.CreateSpaceCreated(t, svc.Context, svc, ctrl, p)

	_, list := test.ListSpaceOK(t, svc.Context, svc, ctrl, nil, nil)
	assert.True(t, len(list.Data) > 0)
	for _, spc := range list.Data {
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

func initTestIdenity(db *gorm.DB) account.Identity {
	identityRepo := account.NewIdentityRepository(db)
	testIdentity := getTestIdentity()
	err := createIdentity(context.Background(), testIdentity, identityRepo)
	if err != nil {
		panic(err.Error())
	}
	return *testIdentity
}

func initTestIdenity2(db *gorm.DB) account.Identity {
	identityRepo := account.NewIdentityRepository(db)
	testIdentity := getTestIdentity2()
	err := createIdentity(context.Background(), testIdentity, identityRepo)
	if err != nil {
		panic(err.Error())
	}
	return *testIdentity
}

func createIdentity(ctx context.Context, identity *account.Identity, identityRepo account.IdentityRepository) error {
	_, err := identityRepo.Load(ctx, identity.ID)
	if err != nil {
		user := account.User{
			Email:    uuid.NewV4().String() + "@domain.com",
			FullName: identity.Username,
		}
		identity.User = user
		identity.UserID = account.NullUUID{UUID: user.ID, Valid: true}
		err := identityRepo.Create(ctx, identity)
		if err != nil {
			return err
		}
	}
	return nil
}

// getTestIdentity returns an identity which represents an existing Keycloak test identityonly
func getTestIdentity() *account.Identity {
	return getTestIdentityBySecret(spaceConfiguration.GetKeycloakTestUserName(), spaceConfiguration.GetKeycloakTestUserSecret())
}

// getTestIdentity returns an identity which represents an existing Keycloak test identityonly
func getTestIdentity2() *account.Identity {
	return getTestIdentityBySecret(spaceConfiguration.GetKeycloakTestUser2Name(), spaceConfiguration.GetKeycloakTestUser2Secret())
}

func getTestIdentityBySecret(username string, usersecret string) *account.Identity {
	testIdentity, err := getUserInfo(username, usersecret)
	if err != nil {
		panic(err.Error())
	}
	id, err := uuid.FromString(testIdentity.Sub)
	if err != nil {
		panic(err.Error())
	}
	idn := account.Identity{
		ID:           id,
		Username:     testIdentity.PreferredUsername,
		ProviderType: account.KeycloakIDP,
	}
	return &idn
}

func getUserInfo(username string, usersecret string) (*auth.UserInfo, error) {
	r := &goa.RequestData{
		Request: &http.Request{Host: "domain.io"},
	}
	tokenEndpoint, err := spaceConfiguration.GetKeycloakEndpointToken(r)
	if err != nil {
		return nil, err
	}
	userinfoEndpoint, err := spaceConfiguration.GetKeycloakEndpointUserInfo(r)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	testToken, err := GenerateUserToken(ctx, tokenEndpoint, spaceConfiguration, username, usersecret)
	if err != nil {
		return nil, err
	}
	accessToken := testToken.Token.AccessToken
	userinfo, err := auth.GetUserInfo(ctx, userinfoEndpoint, *accessToken)
	if err != nil {
		return nil, err
	}
	return userinfo, nil
}
