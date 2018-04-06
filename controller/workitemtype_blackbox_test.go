package controller_test

import (
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"time"

	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The WorkItemTypeTestSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type workItemTypeSuite struct {
	gormtestsupport.DBTestSuite
	typeCtrl     *WorkitemtypeController
	linkTypeCtrl *WorkItemLinkTypeController
	linkCatCtrl  *WorkItemLinkCategoryController
	spaceCtrl    *SpaceController
	svc          *goa.Service
	testDir      string
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemType(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemTypeSuite{
		DBTestSuite: gormtestsupport.NewDBTestSuite(""),
	})
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemTypeSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.testDir = filepath.Join("test-files", "work_item_type")
}

// The SetupTest method will be run before every test in the suite.
func (s *workItemTypeSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	idn := &account.Identity{
		ID:           uuid.Nil,
		Username:     "TestDeveloper",
		ProviderType: "test provider",
	}
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", *idn)
	s.spaceCtrl = NewSpaceController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})
	require.NotNil(s.T(), s.spaceCtrl)
	s.typeCtrl = NewWorkitemtypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(s.svc, gormapplication.NewGormDB(s.DB))
}

func (s *workItemTypeSuite) TestShow() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemTypes(1), tf.Spaces(1))

	s.T().Run("ok", func(t *testing.T) {
		// when
		res, actual := test.ShowWorkitemtypeOK(t, nil, nil, s.typeCtrl, fxt.WorkItemTypes[0].SpaceID, fxt.WorkItemTypes[0].ID, nil, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.wit.golden.json"), actual)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.headers.golden.json"), res.Header())
	})

	s.T().Run("ok - using expired IfModifiedSince header", func(t *testing.T) {
		// when
		lastModified := app.ToHTTPTime(fxt.WorkItemTypes[0].CreatedAt.Add(-1 * time.Hour))
		res, actual := test.ShowWorkitemtypeOK(t, nil, nil, s.typeCtrl, fxt.WorkItemTypes[0].SpaceID, fxt.WorkItemTypes[0].ID, &lastModified, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_lastmodified_header.wit.golden.json"), actual)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_lastmodified_header.headers.golden.json"), res.Header())
	})

	s.T().Run("ok - using IfNoneMatch header", func(t *testing.T) {
		// when
		ifNoneMatch := "foo"
		res, actual := test.ShowWorkitemtypeOK(t, nil, nil, s.typeCtrl, fxt.WorkItemTypes[0].SpaceID, fxt.WorkItemTypes[0].ID, nil, &ifNoneMatch)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_etag_header.wit.golden.json"), actual)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_etag_header.headers.golden.json"), res.Header())
	})

	s.T().Run("not modified - using IfModifiedSince header", func(t *testing.T) {
		// when
		lastModified := app.ToHTTPTime(time.Now().Add(119 * time.Second))
		res := test.ShowWorkitemtypeNotModified(t, nil, nil, s.typeCtrl, fxt.WorkItemTypes[0].SpaceID, fxt.WorkItemTypes[0].ID, &lastModified, nil)
		// then
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_modified_using_if_modified_since_header.headers.golden.json"), res.Header())
	})

	s.T().Run("not modified - using IfNoneMatch header", func(t *testing.T) {
		// when
		etag := app.GenerateEntityTag(fxt.WorkItemTypes[0])
		res := test.ShowWorkitemtypeNotModified(t, nil, nil, s.typeCtrl, fxt.WorkItemTypes[0].SpaceID, fxt.WorkItemTypes[0].ID, nil, &etag)
		// then
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_modified_using_ifnonematch_header.headers.golden.json"), res.Header())
	})
}

func (s *workItemTypeSuite) TestList() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemTypes(2), tf.Spaces(1))

	s.T().Run("ok", func(t *testing.T) {
		// when
		// Paging in the format <start>,<limit>"
		page := "0,-1"
		res, witCollection := test.ListWorkitemtypeOK(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &page, nil, nil)
		// then
		require.NotNil(t, witCollection)
		require.Nil(t, witCollection.Validate())

		toBeFound := id.Slice{fxt.WorkItemTypes[0].ID, fxt.WorkItemTypes[1].ID}.ToMap()
		for _, wit := range witCollection.Data {
			_, ok := toBeFound[*wit.ID]
			assert.True(t, ok, "failed to find work item type %s in expected list", *wit.ID)
			delete(toBeFound, *wit.ID)
		}
		require.Empty(t, toBeFound, "failed to find these expected work item types: %v", toBeFound)

		require.NotNil(t, res.Header()[app.LastModified])
		assert.Equal(t, app.ToHTTPTime(fxt.WorkItemTypes[1].UpdatedAt), res.Header()[app.LastModified][0])
		require.NotNil(t, res.Header()[app.CacheControl])
		assert.NotNil(t, res.Header()[app.CacheControl][0])
		require.NotNil(t, res.Header()[app.ETag])
		assert.Equal(t, generateWorkItemTypesTag(*witCollection), res.Header()[app.ETag][0])
	})

	s.T().Run("ok - using expired IfModifiedSince header", func(t *testing.T) {
		// when
		// Paging in the format <start>,<limit>"
		lastModified := app.ToHTTPTime(time.Now().Add(-1 * time.Hour))
		page := "0,-1"
		res, witCollection := test.ListWorkitemtypeOK(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &page, &lastModified, nil)
		// then
		require.NotNil(t, witCollection)
		require.Nil(t, witCollection.Validate())

		toBeFound := id.Slice{fxt.WorkItemTypes[0].ID, fxt.WorkItemTypes[1].ID}.ToMap()
		for _, wit := range witCollection.Data {
			_, ok := toBeFound[*wit.ID]
			assert.True(t, ok, "failed to find work item type %s in expected list", *wit.ID)
			delete(toBeFound, *wit.ID)
		}
		require.Empty(t, toBeFound, "failed to find these expected work item types: %v", toBeFound)

		require.NotNil(t, res.Header()[app.LastModified])
		assert.Equal(t, app.ToHTTPTime(fxt.WorkItemTypes[1].UpdatedAt), res.Header()[app.LastModified][0])
		require.NotNil(t, res.Header()[app.CacheControl])
		assert.NotNil(t, res.Header()[app.CacheControl][0])
		require.NotNil(t, res.Header()[app.ETag])
		assert.Equal(t, generateWorkItemTypesTag(*witCollection), res.Header()[app.ETag][0])
	})

	s.T().Run("ok - using IfNoneMatch header", func(t *testing.T) {
		// when
		// Paging in the format <start>,<limit>"
		etag := "foo"
		page := "0,-1"
		res, witCollection := test.ListWorkitemtypeOK(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &page, nil, &etag)
		// then
		require.NotNil(t, witCollection)
		require.Nil(t, witCollection.Validate())

		toBeFound := id.Slice{fxt.WorkItemTypes[0].ID, fxt.WorkItemTypes[1].ID}.ToMap()
		for _, wit := range witCollection.Data {
			_, ok := toBeFound[*wit.ID]
			assert.True(t, ok, "failed to find work item type %s in expected list", *wit.ID)
			delete(toBeFound, *wit.ID)
		}
		require.Empty(t, toBeFound, "failed to find these expected work item types: %v", toBeFound)

		require.NotNil(t, res.Header()[app.LastModified])
		assert.Equal(t, app.ToHTTPTime(fxt.WorkItemTypes[1].UpdatedAt), res.Header()[app.LastModified][0])
		require.NotNil(t, res.Header()[app.CacheControl])
		assert.NotNil(t, res.Header()[app.CacheControl][0])
		require.NotNil(t, res.Header()[app.ETag])
		assert.Equal(t, generateWorkItemTypesTag(*witCollection), res.Header()[app.ETag][0])
	})

	s.T().Run("not modified - using IfModifiedSince header", func(t *testing.T) {
		// when/then
		// Paging in the format <start>,<limit>"
		lastModified := app.ToHTTPTime(fxt.WorkItemTypes[1].UpdatedAt)
		page := "0,-1"
		test.ListWorkitemtypeNotModified(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &page, &lastModified, nil)
	})

	s.T().Run("not modified - using IfNoneMatch header", func(t *testing.T) {
		// given
		// Paging in the format <start>,<limit>"
		page := "0,-1"
		_, witCollection := test.ListWorkitemtypeOK(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &page, nil, nil)
		require.NotNil(t, witCollection)
		// when/then
		ifNoneMatch := generateWorkItemTypesTag(*witCollection)
		test.ListWorkitemtypeNotModified(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &page, nil, &ifNoneMatch)
	})
}

//-----------------------------------------------------------------------------
// Test on work item type links retrieval
//-----------------------------------------------------------------------------

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
