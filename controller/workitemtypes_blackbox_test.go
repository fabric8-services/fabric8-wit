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

	s.T().Run("ok", func(t *testing.T) {
		// when
		// Paging in the format <start>,<limit>"
		page := "0,-1"
		res, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &page, nil, nil)
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
		res, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &page, &lastModified, nil)
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
		res, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &page, nil, &etag)
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
		test.ListWorkitemtypesNotModified(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &page, &lastModified, nil)
	})

	s.T().Run("not modified - using IfNoneMatch header", func(t *testing.T) {
		// given
		// Paging in the format <start>,<limit>"
		page := "0,-1"
		_, witCollection := test.ListWorkitemtypesOK(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &page, nil, nil)
		require.NotNil(t, witCollection)
		// when/then
		ifNoneMatch := generateWorkItemTypesTag(*witCollection)
		test.ListWorkitemtypesNotModified(t, nil, nil, s.typeCtrl, fxt.Spaces[0].ID, &page, nil, &ifNoneMatch)
	})
}
