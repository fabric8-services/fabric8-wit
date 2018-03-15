package controller_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSuiteWorkItemLinkTypes(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(workItemLinkTypesSuite))
}

type workItemLinkTypesSuite struct {
	gormtestsupport.DBTestSuite
	linkTypeCtrl *WorkItemLinkTypesController
	svc          *goa.Service
	testDir      string
}

func (s *workItemLinkTypesSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	svc := goa.New("workItemLinkTypesSuite-Service")
	require.NotNil(s.T(), svc)
	s.linkTypeCtrl = NewWorkItemLinkTypesController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	require.NotNil(s.T(), s.linkTypeCtrl)
	s.testDir = filepath.Join("test-files", "work_item_link_types")
}

func TestNewWorkItemLinkTypesControllerDBNull(t *testing.T) {
	require.Panics(t, func() {
		NewWorkItemLinkTypeController(nil, nil, nil)
	})
}

func (s *workItemLinkTypesSuite) TestList() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemLinkTypes(2))
	wilt1 := fxt.WorkItemLinkTypes[0]
	wilt2 := fxt.WorkItemLinkTypes[0]
	requireWILTsIncluded := func(t *testing.T, linkTypes *app.WorkItemLinkTypeList) {
		var found1, found2 bool
		for _, wilt := range linkTypes.Data {
			if *wilt.ID == wilt1.ID {
				found1 = true
			}
			if *wilt.ID == wilt2.ID {
				found2 = true
			}
		}
		require.True(t, found1, "failed to find work item link type 1: %+v", wilt1)
		require.True(t, found2, "failed to find work item link type 2: %+v", wilt2)
	}

	s.T().Run("ok", func(t *testing.T) {
		// when
		res, linkTypes := test.ListWorkItemLinkTypesOK(t, nil, nil, s.linkTypeCtrl, fxt.SpaceTemplates[0].ID, nil, nil)
		// then
		safeOverriteHeader(t, res, app.ETag, "EZUYNwJobqN2yZeWw7GuZw==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok.payload.golden.json"), linkTypes)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok.headers.golden.json"), res.Header())
		requireWILTsIncluded(t, linkTypes)
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok using expired IfModifiedSince header", func(t *testing.T) {
		// when fetching all work item link type in a give space
		ifModifiedSinceHeader := app.ToHTTPTime(wilt1.UpdatedAt.Add(-1 * time.Hour))
		res, linkTypes := test.ListWorkItemLinkTypesOK(t, nil, nil, s.linkTypeCtrl, fxt.SpaceTemplates[0].ID, &ifModifiedSinceHeader, nil)
		// then
		safeOverriteHeader(t, res, app.ETag, "EZUYNwJobqN2yZeWw7GuZw==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifmodifiedsince_header.headers.golden.json"), res.Header())
		requireWILTsIncluded(t, linkTypes)
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok using expired IfNoneMatch header", func(t *testing.T) {
		// when fetching all work item link type in a give space
		ifNoneMatch := "foo"
		res, linkTypes := test.ListWorkItemLinkTypesOK(t, nil, nil, s.linkTypeCtrl, fxt.SpaceTemplates[0].ID, nil, &ifNoneMatch)
		// then
		safeOverriteHeader(t, res, app.ETag, "IGos54TQC8+mZ70zZAWQQg==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifnonematch_header.headers.golden.json"), res.Header())
		requireWILTsIncluded(t, linkTypes)
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified using IfModifiedSince header", func(t *testing.T) {
		// when fetching all work item link type in a give space
		ifModifiedSinceHeader := app.ToHTTPTime(wilt1.UpdatedAt)
		res := test.ListWorkItemLinkTypesNotModified(t, nil, nil, s.linkTypeCtrl, fxt.SpaceTemplates[0].ID, &ifModifiedSinceHeader, nil)
		// then
		safeOverriteHeader(t, res, app.ETag, "IGos54TQC8+mZ70zZAWQQg==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_modified_using_ifmodifiedsince_header.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified using IfNoneMatch header", func(t *testing.T) {
		// when
		_, existingLinkTypes := test.ListWorkItemLinkTypesOK(t, nil, nil, s.linkTypeCtrl, fxt.SpaceTemplates[0].ID, nil, nil)
		// when fetching all work item link type in a give space
		createdWorkItemLinkTypeModels := make([]app.ConditionalRequestEntity, len(existingLinkTypes.Data))
		for i, linkTypeData := range existingLinkTypes.Data {
			createdWorkItemLinkTypeModel, err := ConvertWorkItemLinkTypeToModel(
				app.WorkItemLinkTypeSingle{
					Data: linkTypeData,
				},
			)
			require.Nil(t, err)
			createdWorkItemLinkTypeModels[i] = *createdWorkItemLinkTypeModel
		}
		ifNoneMatch := app.GenerateEntitiesTag(createdWorkItemLinkTypeModels)
		res := test.ListWorkItemLinkTypesNotModified(t, nil, nil, s.linkTypeCtrl, fxt.SpaceTemplates[0].ID, nil, &ifNoneMatch)
		safeOverriteHeader(t, res, app.ETag, "IGos54TQC8+mZ70zZAWQQg==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_modified_using_ifnonematch_header.headers.golden.json"), res.Header())
		// then
		assertResponseHeaders(t, res)
	})
}
