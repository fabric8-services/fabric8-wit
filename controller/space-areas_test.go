package controller_test

import (
	"os"
	"strconv"
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/area"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"

	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
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
	pwd, err := os.Getwd()
	if err != nil {
		require.Nil(t, err)
	}
	suite.Run(t, &TestSpaceAreaREST{DBTestSuite: gormsupport.NewDBTestSuite(pwd + "../config.yaml")})
}

func (rest *TestSpaceAreaREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestSpaceAreaREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSpaceAreaREST) SecuredController() (*goa.Service, *SpaceAreasController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	//priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Area-Service", almtoken.NewManager(pub), testsupport.TestIdentity)
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
		newSpace := &space.Space{
			Name: "Test 1" + uuid.NewV4().String(),
		}
		p, _ = repo.Create(context.Background(), newSpace)
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
		var err error
		repo := app.Spaces()
		newSpace := &space.Space{
			Name: "Test Space 1" + uuid.NewV4().String(),
		}
		s, err = repo.Create(context.Background(), newSpace)
		require.Nil(t, err)

		newSpace = &space.Space{
			Name: "Another space" + uuid.NewV4().String(),
		}
		anotherSpace, err = repo.Create(context.Background(), newSpace)
		require.Nil(t, err)
		return nil
	})

	svc, ctrl := rest.SecuredController()
	spaceId := s.ID
	anotherSpaceId := anotherSpace.ID
	var createdAreaUuids1 []uuid.UUID

	for i := 0; i < 3; i++ {
		name := "Test Area #20" + strconv.Itoa(i)
		spaceAreaContext := createSpaceArea(name, &name)
		_, c := test.CreateSpaceAreasCreated(t, svc.Context, svc, ctrl, spaceId.String(), spaceAreaContext)
		require.NotNil(t, c.Data.ID)
		require.NotNil(t, c.Data.Relationships.Space)
		createdAreaUuids1 = append(createdAreaUuids1, *c.Data.ID)
	}

	name := "area in a different space"
	anotherSpaceAreaContext := createSpaceArea(name, &name)
	_, createdArea := test.CreateSpaceAreasCreated(t, svc.Context, svc, ctrl, anotherSpaceId.String(), anotherSpaceAreaContext)
	require.NotNil(t, createdArea)

	_, areaList := test.ListSpaceAreasOK(t, svc.Context, svc, ctrl, spaceId.String())
	assert.Len(t, areaList.Data, 3)
	for i := 0; i < len(createdAreaUuids1); i++ {
		assert.NotNil(t, searchInAreaSlice(createdAreaUuids1[i], areaList))
	}

	_, anotherAreaList := test.ListSpaceAreasOK(t, svc.Context, svc, ctrl, anotherSpaceId.String())
	assert.Len(t, anotherAreaList.Data, 1)
	assert.Equal(t, anotherAreaList.Data[0].ID, createdArea.Data.ID)

}

func searchInAreaSlice(searchKey uuid.UUID, areaList *app.AreaList) *app.Area {
	for i := 0; i < len(areaList.Data); i++ {
		if searchKey == *areaList.Data[i].ID {
			return areaList.Data[i]
		}
	}
	return nil
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
