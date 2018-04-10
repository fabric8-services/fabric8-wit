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

	"time"

	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type workItemTypesSuite struct {
	gormtestsupport.DBTestSuite
	typeCtrl     *WorkitemtypesController
	linkTypeCtrl *WorkItemLinkTypeController
	linkCatCtrl  *WorkItemLinkCategoryController
	spaceCtrl    *SpaceController
	svc          *goa.Service
	testDir      string
}

func TestSuiteWorkItemTypes(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemTypesSuite{
		DBTestSuite: gormtestsupport.NewDBTestSuite(""),
	})
}

func (s *workItemTypesSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.testDir = filepath.Join("test-files", "work_item_type")
}

func (s *workItemTypesSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	idn := &account.Identity{
		ID:           uuid.Nil,
		Username:     "TestDeveloper",
		ProviderType: "test provider",
	}
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", *idn)
	s.spaceCtrl = NewSpaceController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})
	require.NotNil(s.T(), s.spaceCtrl)
	s.typeCtrl = NewWorkitemtypesController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(s.svc, gormapplication.NewGormDB(s.DB))
}

func (s *workItemTypesSuite) TestList() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemTypes(2), tf.Spaces(1))

	s.T().Run("not found using non existing space", func(t *testing.T) {
		// given
		spaceID := uuid.NewV4()
		// when
		res, jerrs := test.ListWorkitemtypesNotFound(t, nil, nil, s.typeCtrl, spaceID, nil, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_found_using_non_existing_space.res.payload.golden.json"), jerrs)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_found_using_non_existing_space.res.headers.golden.json"), res.Header())
	})

	s.T().Run("ok", func(t *testing.T) {
		// when
		res, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, nil, nil)
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

		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok.res.payload.golden.json"), witCollection)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok - using expired IfModifiedSince header", func(t *testing.T) {
		// when
		lastModified := app.ToHTTPTime(time.Now().Add(-1 * time.Hour))
		res, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &lastModified, nil)
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

		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifmodifiedsince_header.res.payload.golden.json"), witCollection)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifmodifiedsince_header.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok - using IfNoneMatch header", func(t *testing.T) {
		// when
		etag := "foo"
		res, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, nil, &etag)
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
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifnonematch_header.res.payload.golden.json"), witCollection)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifnonematch_header.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified - using IfModifiedSince header", func(t *testing.T) {
		// given
		lastModified := app.ToHTTPTime(fxt.WorkItemTypes[1].UpdatedAt)
		// when
		res := test.ListWorkitemtypesNotModified(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &lastModified, nil)
		// then
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_modified_using_ifmodifiedsince_header.res.headers.golden.json"), res.Header())
	})

	s.T().Run("not modified - using IfNoneMatch header", func(t *testing.T) {
		// given
		_, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, nil, nil)
		require.NotNil(t, witCollection)
		// when
		ifNoneMatch := generateWorkItemTypesTag(*witCollection)
		res := test.ListWorkitemtypesNotModified(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, nil, &ifNoneMatch)
		// then
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_modified_using_ifnonematch_header.res.headers.golden.json"), res.Header())
	})
}
