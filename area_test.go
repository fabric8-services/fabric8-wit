package main_test

import (
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type TestAreaREST struct {
	gormsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunAreaREST(t *testing.T) {
	suite.Run(t, &TestAreaREST{DBTestSuite: gormsupport.NewDBTestSuite("config.yaml")})
}

func (rest *TestAreaREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = gormsupport.DeleteCreatedEntities(rest.DB)
}

func (rest *TestAreaREST) TearDownTest() {
	rest.clean()
}

func (rest *TestAreaREST) SecuredController() (*goa.Service, *AreaController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Area-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
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
	name := "Sprint #21"
	ci := createChildArea(&name)

	svc, ctrl := rest.SecuredController()
	_, created := test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, parentID.String(), ci)
	assert.Equal(t, *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
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

		p, err := app.Spaces().Create(context.Background(), "Test Space 1"+uuid.NewV4().String())
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
