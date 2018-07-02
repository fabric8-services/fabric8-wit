package controller_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSuiteWorkItemLinkTypes(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemLinkTypesSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func TestNewWorkItemLinkTypesControllerDBNull(t *testing.T) {
	require.Panics(t, func() {
		NewWorkItemLinkTypesController(nil, nil, nil)
	})
}

type workItemLinkTypesSuite struct {
	gormtestsupport.DBTestSuite
	testDir string
}

func (s *workItemLinkTypesSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.testDir = filepath.Join("test-files", "work_item_link_type")
}

func (s *workItemLinkTypesSuite) UnSecuredController() (*goa.Service, *WorkItemLinkTypesController) {
	svc := goa.New("WorkItemLinkTypes-Service")
	return svc, NewWorkItemLinkTypesController(svc, s.GormDB, s.Configuration)
}

func (s *workItemLinkTypesSuite) TestList() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemLinkTypes(2))
	_, ctrl := s.UnSecuredController()

	s.T().Run("ok", func(t *testing.T) {
		// when
		res, wilts := test.ListWorkItemLinkTypesOK(t, nil, nil, ctrl, fxt.SpaceTemplates[0].ID, nil, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok.res.payload.golden.json"), wilts)
		safeOverriteHeader(t, res, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("not found for non-existing-spacetemplate", func(t *testing.T) {
		// given
		spaceTemplateID := uuid.NewV4()
		// when
		res, wilts := test.ListWorkItemLinkTypesNotFound(t, nil, nil, ctrl, spaceTemplateID, nil, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_found_for_non_existing_space_id.res.payload.golden.json"), wilts)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_found_for_non_existing_space_id.res.headers.golden.json"), res.Header())
	})

	s.T().Run("ok using expired IfModifiedSince header", func(t *testing.T) {
		// given
		ifModifiedSinceHeader := app.ToHTTPTime(fxt.WorkItemLinkTypes[1].UpdatedAt.Add(-1 * time.Hour))
		// when
		res, wilts := test.ListWorkItemLinkTypesOK(t, nil, nil, ctrl, fxt.SpaceTemplates[0].ID, &ifModifiedSinceHeader, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifmodifiedsince_header.res.payload.golden.json"), wilts)
		safeOverriteHeader(t, res, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifmodifiedsince_header.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok using expired IfNoneMatch header", func(t *testing.T) {
		// given
		ifNoneMatch := "foo"
		// when
		res, wilts := test.ListWorkItemLinkTypesOK(t, nil, nil, ctrl, fxt.SpaceTemplates[0].ID, nil, &ifNoneMatch)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifnonematch_header.res.payload.golden.json"), wilts)
		safeOverriteHeader(t, res, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifnonematch_header.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified using IfModifiedSince header", func(t *testing.T) {
		// given
		ifModifiedSinceHeader := app.ToHTTPTime(fxt.WorkItemLinkTypes[1].UpdatedAt)
		// when
		res := test.ListWorkItemLinkTypesNotModified(t, nil, nil, ctrl, fxt.SpaceTemplates[0].ID, &ifModifiedSinceHeader, nil)
		// then
		safeOverriteHeader(t, res, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_modified_using_ifmodifiedsince_header.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified using IfNoneMatch header", func(t *testing.T) {
		// given
		res, _ := test.ListWorkItemLinkTypesOK(t, nil, nil, ctrl, fxt.SpaceTemplates[0].ID, nil, nil)
		ifNoneMatch := res.Header().Get(app.ETag)
		// when
		res = test.ListWorkItemLinkTypesNotModified(t, nil, nil, ctrl, fxt.SpaceTemplates[0].ID, nil, &ifNoneMatch)
		// then
		safeOverriteHeader(t, res, app.ETag, "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "not_modified_using_ifnonematch_header.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})
}
