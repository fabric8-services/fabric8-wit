package controller_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/area"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"

	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/test/resource"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type TestAreaREST struct {
	gormsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunAreaREST(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		require.Nil(t, err)
	}
	suite.Run(t, &TestAreaREST{DBTestSuite: gormsupport.NewDBTestSuite(pwd + "/../config.yaml")})
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
	return svc, NewAreaController(svc, rest.db)
}

func (rest *TestAreaREST) UnSecuredController() (*goa.Service, *AreaController) {
	svc := goa.New("Area-Service")
	return svc, NewAreaController(svc, rest.db)
}

func (rest *TestAreaREST) TestSuccessCreateChildArea() {
	t := rest.T()
	resource.Require(t, resource.Database)

	parentID := createSpaceAndArea(t, rest.db).ID
	name := "Area 21"
	ci := createChildArea(&name)

	svc, ctrl := rest.SecuredController()
	_, created := test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, parentID.String(), ci)
	assert.Equal(t, *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	fmt.Println(*created.Data.Relationships.Parent.Data.ID)
	assert.Equal(t, parentID.String(), *created.Data.Relationships.Parent.Data.ID)

}

func (rest *TestAreaREST) TestSuccessCreateMultiChildArea() {
	t := rest.T()
	resource.Require(t, resource.Database)

	/*
		Area 2 ---> Area 21-0 ----> Area 21-0-0
	*/

	parentID := createSpaceAndArea(t, rest.db).ID
	name := "Area 21-0"
	ci := createChildArea(&name)

	svc, ctrl := rest.SecuredController()
	_, created := test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, parentID.String(), ci)
	assert.Equal(t, *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	assert.Equal(t, parentID.String(), *created.Data.Relationships.Parent.Data.ID)

	// Create a child of the child created above.
	name = "Area 21-0-0"
	ci = createChildArea(&name)
	newParentID := *created.Data.Relationships.Parent.Data.ID
	_, created = test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, newParentID, ci)
	assert.Equal(t, *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	assert.NotNil(t, *created.Data.Attributes.CreatedAt)
	assert.NotNil(t, *created.Data.Attributes.Version)
	assert.Equal(t, newParentID, *created.Data.Relationships.Parent.Data.ID)
	assert.Contains(t, *created.Data.Relationships.Children.Links.Self, "children")

}

func (rest *TestAreaREST) TestFailCreateChildAreaMissingName() {
	t := rest.T()
	resource.Require(t, resource.Database)

	parentID := createSpaceAndArea(t, rest.db).ID
	childArea := createChildArea(nil)

	svc, ctrl := rest.SecuredController()
	test.CreateChildAreaBadRequest(t, svc.Context, svc, ctrl, parentID.String(), childArea)
}

func (rest *TestAreaREST) TestFailCreateChildAreaWithInvalidsParent() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Sprint #21"
	childArea := createChildArea(&name)

	svc, ctrl := rest.SecuredController()
	test.CreateChildAreaNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), childArea)
}

func (rest *TestAreaREST) TestFailCreateChildAreaNotAuthorized() {
	t := rest.T()
	resource.Require(t, resource.Database)

	parentID := createSpaceAndArea(t, rest.db).ID
	name := "Area #73467834"
	childArea := createChildArea(&name)

	svc, ctrl := rest.UnSecuredController()
	test.CreateChildAreaUnauthorized(t, svc.Context, svc, ctrl, parentID.String(), childArea)
}

func (rest *TestAreaREST) TestFailShowAreaNotFound() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.SecuredController()
	test.ShowAreaNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String())
}

func createChildArea(name *string) *app.CreateChildAreaPayload {
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
			Name: "Test Space 1" + uuid.NewV4().String(),
		}
		p, err := app.Spaces().Create(context.Background(), newSpace)
		if err != nil {
			t.Error(err)
		}
		name := "Area #2"

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
