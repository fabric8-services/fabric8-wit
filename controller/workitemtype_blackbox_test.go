package controller_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type workItemTypeSuite struct {
	gormtestsupport.DBTestSuite
	typeCtrl     *WorkitemtypeController
	linkTypeCtrl *WorkItemLinkTypeController
	linkCatCtrl  *WorkItemLinkCategoryController
	spaceCtrl    *SpaceController
	svc          *goa.Service
	testDir      string
}

func TestSuiteWorkItemType(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemTypeSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *workItemTypeSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.testDir = filepath.Join("test-files", "work_item_type")
}

func (s *workItemTypeSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	idn := &account.Identity{
		ID:           uuid.Nil,
		Username:     "TestDeveloper",
		ProviderType: "test provider",
	}
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", *idn)
	s.spaceCtrl = NewSpaceController(s.svc, s.GormDB, s.Configuration, &DummyResourceManager{})
	require.NotNil(s.T(), s.spaceCtrl)
	s.typeCtrl = NewWorkitemtypeController(s.svc, s.GormDB, s.Configuration)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(s.svc, s.GormDB, s.Configuration)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(s.svc, s.GormDB)

}

func (s *workItemTypeSuite) TestShow() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemTypes(1), tf.Spaces(1))

	s.T().Run("ok", func(t *testing.T) {
		// when
		res, actual := test.ShowWorkitemtypeOK(t, nil, nil, s.typeCtrl, fxt.WorkItemTypes[0].ID, nil, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.wit.golden.json"), actual)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.headers.golden.json"), res.Header())
	})

	s.T().Run("ok - using expired IfModifiedSince header", func(t *testing.T) {
		// when
		lastModified := app.ToHTTPTime(fxt.WorkItemTypes[0].CreatedAt.Add(-1 * time.Hour))
		res, actual := test.ShowWorkitemtypeOK(t, nil, nil, s.typeCtrl, fxt.WorkItemTypes[0].ID, &lastModified, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_lastmodified_header.wit.golden.json"), actual)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_lastmodified_header.headers.golden.json"), res.Header())
	})

	s.T().Run("ok - using IfNoneMatch header", func(t *testing.T) {
		// when
		ifNoneMatch := "foo"
		res, actual := test.ShowWorkitemtypeOK(t, nil, nil, s.typeCtrl, fxt.WorkItemTypes[0].ID, nil, &ifNoneMatch)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_etag_header.wit.golden.json"), actual)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_etag_header.headers.golden.json"), res.Header())
	})

	s.T().Run("not modified - using IfModifiedSince header", func(t *testing.T) {
		// when
		lastModified := app.ToHTTPTime(time.Now().Add(119 * time.Second))
		res := test.ShowWorkitemtypeNotModified(t, nil, nil, s.typeCtrl, fxt.WorkItemTypes[0].ID, &lastModified, nil)
		// then
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_modified_using_if_modified_since_header.headers.golden.json"), res.Header())
	})

	s.T().Run("not modified - using IfNoneMatch header", func(t *testing.T) {
		// when
		etag := app.GenerateEntityTag(fxt.WorkItemTypes[0])
		res := test.ShowWorkitemtypeNotModified(t, nil, nil, s.typeCtrl, fxt.WorkItemTypes[0].ID, nil, &etag)
		// then
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_modified_using_ifnonematch_header.headers.golden.json"), res.Header())
	})
}

// used for testing purpose only
func ConvertWorkItemTypeToModel(data app.WorkItemTypeData) workitem.WorkItemType {
	return workitem.WorkItemType{
		ID:      *data.ID,
		Version: *data.Attributes.Version,
	}
}

func generateWorkItemTypesTag(entities app.WorkItemTypeList) string {
	modelEntities := make([]app.ConditionalRequestEntity, len(entities.Data))
	for i, entityData := range entities.Data {
		modelEntities[i] = ConvertWorkItemTypeToModel(*entityData)
	}
	return app.GenerateEntitiesTag(modelEntities)
}

func generateWorkItemTypeTag(entity app.WorkItemTypeSingle) string {
	return app.GenerateEntityTag(ConvertWorkItemTypeToModel(*entity.Data))
}

func generateWorkItemLinkTypesTag(entities app.WorkItemLinkTypeList) string {
	modelEntities := make([]app.ConditionalRequestEntity, len(entities.Data))
	for i, entityData := range entities.Data {
		e, _ := ConvertWorkItemLinkTypeToModel(app.WorkItemLinkTypeSingle{Data: entityData})
		modelEntities[i] = e
	}
	return app.GenerateEntitiesTag(modelEntities)
}

func generateWorkItemLinkTypeTag(entity app.WorkItemLinkTypeSingle) string {
	e, _ := ConvertWorkItemLinkTypeToModel(entity)
	return app.GenerateEntityTag(e)
}

func ConvertWorkItemTypesToConditionalEntities(workItemTypeList app.WorkItemTypeList) []app.ConditionalRequestEntity {
	conditionalWorkItemTypes := make([]app.ConditionalRequestEntity, len(workItemTypeList.Data))
	for i, data := range workItemTypeList.Data {
		conditionalWorkItemTypes[i] = ConvertWorkItemTypeToModel(*data)
	}
	return conditionalWorkItemTypes
}

func getWorkItemLinkTypeUpdatedAt(appWorkItemLinkType app.WorkItemLinkTypeSingle) time.Time {
	return *appWorkItemLinkType.Data.Attributes.UpdatedAt
}
