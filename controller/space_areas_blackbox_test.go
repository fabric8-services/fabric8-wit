package controller_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"

	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestSpaceAreaREST struct {
	gormtestsupport.DBTestSuite
	svcSpaceAreas  *goa.Service
	ctrlSpaceAreas *SpaceAreasController
}

func TestRunSpaceAreaREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestSpaceAreaREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (rest *TestSpaceAreaREST) SetupTest() {
	rest.svcSpaceAreas, rest.ctrlSpaceAreas = rest.SecuredController()
}

func (rest *TestSpaceAreaREST) SecuredController() (*goa.Service, *SpaceAreasController) {
	svc := testsupport.ServiceAsUser("Space-Area-Service", testsupport.TestIdentity)
	return svc, NewSpaceAreasController(svc, rest.GormDB, rest.Configuration)
}

func (rest *TestSpaceAreaREST) SecuredAreasController() (*goa.Service, *AreaController) {
	svc := testsupport.ServiceAsUser("Area-Service", testsupport.TestIdentity)
	return svc, NewAreaController(svc, rest.GormDB, rest.Configuration)
}

func (rest *TestSpaceAreaREST) SecuredAreasControllerWithIdentity(idn *account.Identity) (*goa.Service, *AreaController) {
	svc := testsupport.ServiceAsUser("Area-Service-With-Identity", *idn)
	return svc, NewAreaController(svc, rest.GormDB, rest.Configuration)
}

func (rest *TestSpaceAreaREST) UnSecuredController() (*goa.Service, *SpaceAreasController) {
	svc := goa.New("Area-Service")
	return svc, NewSpaceAreasController(svc, rest.GormDB, rest.Configuration)
}

func searchInAreaSlice(searchKey uuid.UUID, areaList *app.AreaList) *app.Area {
	for i := 0; i < len(areaList.Data); i++ {
		if searchKey == *areaList.Data[i].ID {
			return areaList.Data[i]
		}
	}
	return nil
}

func assertSpaceAreas(t *testing.T, areaList *app.AreaList, createdAreaUuids []uuid.UUID) {
	assert.Len(t, areaList.Data, 3)
	for i := 0; i < len(createdAreaUuids); i++ {
		assert.NotNil(t, searchInAreaSlice(createdAreaUuids[i], areaList))
	}
}

func (rest *TestSpaceAreaREST) TestListAreasOK() {
	fxt := tf.NewTestFixture(rest.T(), rest.DB,
		tf.CreateWorkItemEnvironment(),
		tf.Areas(3, func(fxt *tf.TestFixture, idx int) error {
			if idx > 0 {
				fxt.Areas[idx].MakeChildOf(*fxt.Areas[idx-1])
			}
			return nil
		}),
	)
	// given
	parentArea := fxt.Areas[0]
	createdAreaUuids := []uuid.UUID{
		fxt.Areas[0].ID,
		fxt.Areas[1].ID,
		fxt.Areas[2].ID,
	}
	// when
	res, areaList := test.ListSpaceAreasOK(rest.T(), rest.svcSpaceAreas.Context, rest.svcSpaceAreas, rest.ctrlSpaceAreas, parentArea.SpaceID, nil, nil)
	// then
	assertSpaceAreas(rest.T(), areaList, createdAreaUuids)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestSpaceAreaREST) TestListAreasOKUsingExpiredIfModifiedSinceHeader() {
	fxt := tf.NewTestFixture(rest.T(), rest.DB,
		tf.CreateWorkItemEnvironment(),
		tf.Areas(3, func(fxt *tf.TestFixture, idx int) error {
			if idx > 0 {
				fxt.Areas[idx].MakeChildOf(*fxt.Areas[idx-1])
			}
			return nil
		}),
	)
	// given
	parentArea := fxt.Areas[0]
	createdAreaUuids := []uuid.UUID{
		fxt.Areas[0].ID,
		fxt.Areas[1].ID,
		fxt.Areas[2].ID,
	}
	// when
	ifModifiedSince := app.ToHTTPTime(parentArea.UpdatedAt.Add(-1 * time.Hour))
	res, areaList := test.ListSpaceAreasOK(rest.T(), rest.svcSpaceAreas.Context, rest.svcSpaceAreas, rest.ctrlSpaceAreas, parentArea.SpaceID, &ifModifiedSince, nil)
	// then
	assertSpaceAreas(rest.T(), areaList, createdAreaUuids)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestSpaceAreaREST) TestListAreasOKUsingExpiredIfNoneMatchHeader() {
	fxt := tf.NewTestFixture(rest.T(), rest.DB,
		tf.CreateWorkItemEnvironment(),
		tf.Areas(3, func(fxt *tf.TestFixture, idx int) error {
			if idx > 0 {
				fxt.Areas[idx].MakeChildOf(*fxt.Areas[idx-1])
			}
			return nil
		}),
	)
	// given
	parentArea := fxt.Areas[0]
	createdAreaUuids := []uuid.UUID{
		fxt.Areas[0].ID,
		fxt.Areas[1].ID,
		fxt.Areas[2].ID,
	}
	// when
	ifNoneMatch := "foo"
	res, areaList := test.ListSpaceAreasOK(rest.T(), rest.svcSpaceAreas.Context, rest.svcSpaceAreas, rest.ctrlSpaceAreas, parentArea.SpaceID, nil, &ifNoneMatch)
	// then
	assertSpaceAreas(rest.T(), areaList, createdAreaUuids)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestSpaceAreaREST) TestListAreasNotModifiedUsingIfModifiedSinceHeader() {
	fxt := tf.NewTestFixture(rest.T(), rest.DB,
		tf.CreateWorkItemEnvironment(),
		tf.Areas(3, func(fxt *tf.TestFixture, idx int) error {
			if idx > 0 {
				fxt.Areas[idx].MakeChildOf(*fxt.Areas[idx-1])
			}
			return nil
		}),
	)
	// given
	parentArea := fxt.Areas[0]
	// when
	ifModifiedSince := app.ToHTTPTime(fxt.Areas[2].UpdatedAt)
	res := test.ListSpaceAreasNotModified(rest.T(), rest.svcSpaceAreas.Context, rest.svcSpaceAreas, rest.ctrlSpaceAreas, parentArea.SpaceID, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestSpaceAreaREST) TestListAreasNotModifiedUsingIfNoneMatchHeader() {
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB,
		tf.CreateWorkItemEnvironment(),
		tf.Areas(3, func(fxt *tf.TestFixture, idx int) error {
			if idx > 0 {
				fxt.Areas[idx].MakeChildOf(*fxt.Areas[idx-1])
			}
			return nil
		}),
	)
	parentArea := fxt.Areas[0]
	// when
	ifNoneMatch := app.GenerateEntitiesTag([]app.ConditionalRequestEntity{
		fxt.Areas[0],
		fxt.Areas[1],
		fxt.Areas[2],
	})
	res := test.ListSpaceAreasNotModified(rest.T(), rest.svcSpaceAreas.Context, rest.svcSpaceAreas, rest.ctrlSpaceAreas, parentArea.SpaceID, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(rest.T(), res)
}
