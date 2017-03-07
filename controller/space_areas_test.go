package controller_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
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

type TestSpaceAreaREST struct {
	gormtestsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunSpaceAreaREST(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		require.Nil(t, err)
	}
	suite.Run(t, &TestSpaceAreaREST{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
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

	svc := testsupport.ServiceAsUser("Space-Area-Service", almtoken.NewManager(pub), testsupport.TestIdentity)
	return svc, NewSpaceAreasController(svc, rest.db)
}

func (rest *TestSpaceAreaREST) SecuredAreasController() (*goa.Service, *AreaController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))

	svc := testsupport.ServiceAsUser("Area-Service", almtoken.NewManager(pub), testsupport.TestIdentity)
	return svc, NewAreaController(svc, rest.db)
}

func (rest *TestSpaceAreaREST) UnSecuredController() (*goa.Service, *SpaceAreasController) {
	svc := goa.New("Area-Service")
	return svc, NewSpaceAreasController(svc, rest.db)
}

func searchInAreaSlice(searchKey uuid.UUID, areaList *app.AreaList) *app.Area {
	for i := 0; i < len(areaList.Data); i++ {
		if searchKey == *areaList.Data[i].ID {
			return areaList.Data[i]
		}
	}
	return nil
}

func (rest *TestSpaceAreaREST) TestListAreas() {
	t := rest.T()
	resource.Require(t, resource.Database)

	/*
		Space X --> Area 2 ---> Area 21-0
	*/
	var createdAreaUuids []uuid.UUID

	parentArea := createSpaceAndArea(t, rest.db)
	createdAreaUuids = append(createdAreaUuids, parentArea.ID)

	parentID := parentArea.ID
	name := "Area 2000A"
	ci := createChildArea(&name)

	svc, ctrl := rest.SecuredAreasController()
	_, created := test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, parentID.String(), ci)
	assert.Equal(t, *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	assert.Equal(t, parentID.String(), *created.Data.Relationships.Parent.Data.ID)
	createdAreaUuids = append(createdAreaUuids, *created.Data.ID)

	// Create a child of the child created above.
	name = "Area 21-00000A"
	ci = createChildArea(&name)
	newParentID := *created.Data.Relationships.Parent.Data.ID
	_, created = test.CreateChildAreaCreated(t, svc.Context, svc, ctrl, newParentID, ci)
	assert.Equal(t, *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	assert.NotNil(t, *created.Data.Attributes.CreatedAt)
	assert.NotNil(t, *created.Data.Attributes.Version)
	assert.Equal(t, newParentID, *created.Data.Relationships.Parent.Data.ID)
	assert.Contains(t, *created.Data.Relationships.Children.Links.Self, "children")
	createdAreaUuids = append(createdAreaUuids, *created.Data.ID)

	// Now use Space-Areas list action to see all areas under this space.
	fmt.Println("Created areas...")
	svcSpaceAreas, ctrlSpaceAreas := rest.SecuredController()
	_, areaList := test.ListSpaceAreasOK(t, svcSpaceAreas.Context, svcSpaceAreas, ctrlSpaceAreas, parentArea.SpaceID.String())
	assert.Len(t, areaList.Data, 3)
	for i := 0; i < len(createdAreaUuids); i++ {
		assert.NotNil(t, searchInAreaSlice(createdAreaUuids[i], areaList))
	}

}
