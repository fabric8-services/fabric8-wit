package main_test

import (
	"strconv"
	"testing"

	"golang.org/x/net/context"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestSpaceAreaREST struct {
	gormsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunSpaceAreaREST(t *testing.T) {
	suite.Run(t, &TestSpaceAreaREST{DBTestSuite: gormsupport.NewDBTestSuite("config.yaml")})
}

func (rest *TestSpaceAreaREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = gormsupport.DeleteCreatedEntities(rest.DB)
}

func (rest *TestSpaceAreaREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSpaceAreaREST) SecuredController() (*goa.Service, *SpaceAreasController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Area-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
	return svc, NewSpaceAreasController(svc, rest.db)
}

func (rest *TestSpaceAreaREST) UnSecuredController() (*goa.Service, *SpaceAreasController) {
	svc := goa.New("Area-Service")
	return svc, NewSpaceAreasController(svc, rest.db)
}

func (rest *TestSpaceAreaREST) TestSuccessCreateArea() {
	t := rest.T()
	resource.Require(t, resource.Database)

	var p *space.Space
	ci := createSpaceArea("Area #21", nil)

	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Spaces()
		p, _ = repo.Create(context.Background(), "Test 1")
		return nil
	})
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateSpaceAreasCreated(t, svc.Context, svc, ctrl, p.ID.String(), ci)
	require.NotNil(t, c.Data.ID)
	require.NotNil(t, c.Data.Relationships.Space)
	assert.Equal(t, p.ID.String(), *c.Data.Relationships.Space.Data.ID)
}

func (rest *TestSpaceAreaREST) TestListAreas() {
	t := rest.T()
	resource.Require(t, resource.Database)

	// Create a new space where we'll create 3 areas
	var s *space.Space

	// Create another space where we'll create 1 area.
	var anotherSpace *space.Space

	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Spaces()
		s, _ = repo.Create(context.Background(), "Test 1")
		anotherSpace, _ = repo.Create(context.Background(), "Another space")
		return nil
	})

	svc, ctrl := rest.SecuredController()
	spaceId := s.ID
	anotherSpaceId := anotherSpace.ID

	for i := 0; i < 3; i++ {
		name := "Test Area #20" + strconv.Itoa(i)
		spaceAreaContext := createSpaceArea(name, &name)
		_, c := test.CreateSpaceAreasCreated(t, svc.Context, svc, ctrl, spaceId.String(), spaceAreaContext)
		require.NotNil(t, c.Data.ID)
		require.NotNil(t, c.Data.Relationships.Space)
	}

	name := "area in a different space"
	anotherSpaceAreaContext := createSpaceArea(name, &name)
	test.CreateSpaceAreasCreated(t, svc.Context, svc, ctrl, anotherSpaceId.String(), anotherSpaceAreaContext)

	_, areaList := test.ListSpaceAreasOK(t, svc.Context, svc, ctrl, spaceId.String())
	assert.Len(t, areaList.Data, 3)

	_, anotherAreaList := test.ListSpaceAreasOK(t, svc.Context, svc, ctrl, anotherSpaceId.String())
	assert.Len(t, anotherAreaList.Data, 1)

}

func createSpaceArea(name string, desc *string) *app.CreateSpaceAreasPayload {

	return &app.CreateSpaceAreasPayload{
		Data: &app.Area{
			Type: area.APIStringTypeAreas,
			Attributes: &app.AreaAttributes{
				Name: &name,
			},
		},
	}
}
