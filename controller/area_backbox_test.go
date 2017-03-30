package controller_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/area"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"

	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type TestAreaREST struct {
	gormtestsupport.DBTestSuite
	db    *gormapplication.GormDB
	clean func()
}

func TestRunAreaREST(t *testing.T) {
	resource.Require(t, resource.Database)
	pwd, err := os.Getwd()
	if err != nil {
		require.Nil(t, err)
	}
	suite.Run(t, &TestAreaREST{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (rest *TestAreaREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestAreaREST) TearDownTest() {
	rest.clean()
}

func (rest *TestAreaREST) SecuredController() (*goa.Service, *AreaController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	//priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("Area-Service", almtoken.NewManager(pub), testsupport.TestIdentity)
	return svc, NewAreaController(svc, rest.db, rest.Configuration)
}

func (rest *TestAreaREST) UnSecuredController() (*goa.Service, *AreaController) {
	svc := goa.New("Area-Service")
	return svc, NewAreaController(svc, rest.db, rest.Configuration)
}

func (rest *TestAreaREST) TestSuccessCreateChildArea() {
	// given
	parentID := createSpaceAndArea(rest.T(), rest.db).ID
	name := "TestSuccessCreateChildArea"
	ci := getCreateChildAreaPayload(&name)
	svc, ctrl := rest.SecuredController()
	// when
	_, created := test.CreateChildAreaCreated(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
	// then
	assert.Equal(rest.T(), *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	fmt.Println(*created.Data.Relationships.Parent.Data.ID)
	assert.Equal(rest.T(), parentID.String(), *created.Data.Relationships.Parent.Data.ID)
}

func (rest *TestAreaREST) TestSuccessCreateMultiChildArea() {
	/*
		TestAreaREST ---> TestSuccessCreateMultiChildArea-0 ----> TestSuccessCreateMultiChildArea-0-0
	*/
	// given
	parentID := createSpaceAndArea(rest.T(), rest.db).ID
	name := "TestSuccessCreateMultiChildArea-0"
	ci := getCreateChildAreaPayload(&name)
	svc, ctrl := rest.SecuredController()
	// when
	_, created := test.CreateChildAreaCreated(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
	// then
	assert.Equal(rest.T(), *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	assert.Equal(rest.T(), parentID.String(), *created.Data.Relationships.Parent.Data.ID)
	// Create a child of the child created above.
	name = "TestSuccessCreateMultiChildArea-0-0"
	ci = getCreateChildAreaPayload(&name)
	newParentID := *created.Data.Relationships.Parent.Data.ID
	// when
	_, created = test.CreateChildAreaCreated(rest.T(), svc.Context, svc, ctrl, newParentID, ci)
	// then
	assert.Equal(rest.T(), *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	assert.NotNil(rest.T(), *created.Data.Attributes.CreatedAt)
	assert.NotNil(rest.T(), *created.Data.Attributes.Version)
	assert.Equal(rest.T(), newParentID, *created.Data.Relationships.Parent.Data.ID)
	assert.Contains(rest.T(), *created.Data.Relationships.Children.Links.Self, "children")
}

func (rest *TestAreaREST) TestFailCreateChildAreaMissingName() {
	// given
	parentID := createSpaceAndArea(rest.T(), rest.db).ID
	createChildAreaPayload := getCreateChildAreaPayload(nil)
	svc, ctrl := rest.SecuredController()
	// when/then
	test.CreateChildAreaBadRequest(rest.T(), svc.Context, svc, ctrl, parentID.String(), createChildAreaPayload)
}

func (rest *TestAreaREST) TestFailCreateChildAreaWithInvalidsParent() {
	// given
	name := "TestFailCreateChildAreaWithInvalidsParent"
	createChildAreaPayload := getCreateChildAreaPayload(&name)
	svc, ctrl := rest.SecuredController()
	// when/then
	test.CreateChildAreaNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), createChildAreaPayload)
}

func (rest *TestAreaREST) TestFailCreateChildAreaNotAuthorized() {
	// given
	parentID := createSpaceAndArea(rest.T(), rest.db).ID
	name := "TestFailCreateChildAreaNotAuthorized"
	createChildAreaPayload := getCreateChildAreaPayload(&name)
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.CreateChildAreaUnauthorized(rest.T(), svc.Context, svc, ctrl, parentID.String(), createChildAreaPayload)
}

func (rest *TestAreaREST) TestFailShowAreaNotFound() {
	// given
	svc, ctrl := rest.SecuredController()
	// when/then
	test.ShowAreaNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), nil, nil)
}

func (rest *TestAreaREST) TestShowAreaOK() {
	// given
	a := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	res, _ := test.ShowAreaOK(rest.T(), svc.Context, svc, ctrl, a.ID.String(), nil, nil)
	//then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowAreaOKUsingExpiredIfModifedSinceHeader() {
	// given
	a := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	ifModifiedSince := app.ToHTTPTime(a.UpdatedAt.Add(-1 * time.Hour))
	res, _ := test.ShowAreaOK(rest.T(), svc.Context, svc, ctrl, a.ID.String(), &ifModifiedSince, nil)
	//then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowAreaOKUsingExpiredIfNoneMatchHeader() {
	// given
	a := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	ifNoneMatch := "foo"
	res, _ := test.ShowAreaOK(rest.T(), svc.Context, svc, ctrl, a.ID.String(), nil, &ifNoneMatch)
	//then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowAreaNotModifiedUsingIfModifedSinceHeader() {
	// given
	a := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	ifModifiedSince := app.ToHTTPTime(a.UpdatedAt)
	res := test.ShowAreaNotModified(rest.T(), svc.Context, svc, ctrl, a.ID.String(), &ifModifiedSince, nil)
	//then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowAreaNotModifiedIfNoneMatchHeader() {
	// given
	a := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	ifNoneMatch := app.GenerateEntityTag(a)
	res := test.ShowAreaNotModified(rest.T(), svc.Context, svc, ctrl, a.ID.String(), nil, &ifNoneMatch)
	//then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) createChildArea(name string, parent area.Area) *app.AreaSingle {
	ci := getCreateChildAreaPayload(&name)
	svc, ctrl := rest.SecuredController()
	// when
	_, created := test.CreateChildAreaCreated(rest.T(), svc.Context, svc, ctrl, parent.ID.String(), ci)
	return created
}

func (rest *TestAreaREST) TestShowChildrenAreaOK() {
	// given
	parentArea := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	rest.createChildArea("TestShowChildrenAreaOK", parentArea)
	// when
	res, result := test.ShowChildrenAreaOK(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), nil, nil)
	//then
	assert.Equal(rest.T(), 1, len(result.Data))
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowChildrenAreaOKUsingExpiredIfModifedSinceHeader() {
	// given
	parentArea := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	rest.createChildArea("TestShowChildrenAreaOKUsingExpiredIfModifedSinceHeader", parentArea)
	// when
	ifModifiedSince := app.ToHTTPTime(parentArea.UpdatedAt.Add(-1 * time.Hour))
	res, result := test.ShowChildrenAreaOK(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), &ifModifiedSince, nil)
	//then
	assert.Equal(rest.T(), 1, len(result.Data))
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowChildrenAreaOKUsingExpiredIfNoneMatchHeader() {
	// given
	parentArea := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	rest.createChildArea("TestShowChildrenAreaOKUsingExpiredIfNoneMatchHeader", parentArea)
	// when
	ifNoneMatch := "foo"
	res, result := test.ShowChildrenAreaOK(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), nil, &ifNoneMatch)
	//then
	assert.Equal(rest.T(), 1, len(result.Data))
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowChildrenAreaNotModifiedUsingIfModifedSinceHeader() {
	// given
	parentArea := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	childArea := rest.createChildArea("TestShowChildrenAreaNotModifiedUsingIfModifedSinceHeader", parentArea)
	// when
	ifModifiedSince := app.ToHTTPTime(*childArea.Data.Attributes.UpdatedAt)
	res := test.ShowChildrenAreaNotModified(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), &ifModifiedSince, nil)
	//then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowChildrenAreaNotModifiedIfNoneMatchHeader() {
	// given
	parentArea := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	childArea := rest.createChildArea("TestShowChildrenAreaNotModifiedIfNoneMatchHeader", parentArea)
	modelChildArea := convertAreaToModel(*childArea)
	// when
	ifNoneMatch := app.GenerateEntityTag(modelChildArea)
	res := test.ShowChildrenAreaNotModified(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), nil, &ifNoneMatch)
	//then
	assertResponseHeaders(rest.T(), res)
}

func convertAreaToModel(appArea app.AreaSingle) area.Area {
	return area.Area{
		ID:      *appArea.Data.ID,
		Version: *appArea.Data.Attributes.Version,
		Lifecycle: gormsupport.Lifecycle{
			UpdatedAt: *appArea.Data.Attributes.UpdatedAt,
		},
	}
}

func getCreateChildAreaPayload(name *string) *app.CreateChildAreaPayload {
	areaType := area.APIStringTypeAreas
	return &app.CreateChildAreaPayload{
		Data: &app.Area{
			Type: areaType,
			Attributes: &app.AreaAttributes{
				Name: name,
			},
		},
	}
}

func createSpaceAndArea(t *testing.T, db *gormapplication.GormDB) area.Area {
	var areaObj area.Area
	application.Transactional(db, func(app application.Application) error {
		repo := app.Areas()
		newSpace := &space.Space{
			Name: "TestAreaREST" + uuid.NewV4().String(),
		}
		p, err := app.Spaces().Create(context.Background(), newSpace)
		if err != nil {
			t.Error(err)
		}
		name := "Main Area" + uuid.NewV4().String()
		i := area.Area{
			Name:    name,
			SpaceID: p.ID,
		}
		repo.Create(context.Background(), &i)
		areaObj = i
		return nil
	})
	return areaObj
}
