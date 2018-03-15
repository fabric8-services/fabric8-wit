package controller_test

import (
	"net/http"
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
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSuiteWorkItemLinkType(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(workItemLinkTypeSuite))
}

type workItemLinkTypeSuite struct {
	gormtestsupport.DBTestSuite
	linkTypeCtrl *WorkItemLinkTypeController
	svc          *goa.Service
	testDir      string
}

func (s *workItemLinkTypeSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	svc := goa.New("workItemLinkTypeSuite-Service")
	require.NotNil(s.T(), svc)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	require.NotNil(s.T(), s.linkTypeCtrl)
	s.testDir = filepath.Join("test-files", "work_item_link_type")
}

func TestNewWorkItemLinkTypeControllerDBNull(t *testing.T) {
	require.Panics(t, func() {
		NewWorkItemLinkTypeController(nil, nil, nil)
	})
}

// safeOverriteHeader checks if an header entry with the given key is present
// and only then sets it to the given value
func safeOverriteHeader(t *testing.T, res http.ResponseWriter, key string, val string) {
	obj := res.Header()[key]
	require.NotEmpty(t, obj, `response header entry "%s" is empty or not set`, key)
	res.Header().Set(key, val)
}

func (s *workItemLinkTypeSuite) TestShow() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemLinkTypes(1))
	wilt := fxt.WorkItemLinkTypes[0]

	s.T().Run("ok", func(t *testing.T) {
		// when
		res, shownWilt := test.ShowWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, wilt.ID, nil, nil)
		// then
		safeOverriteHeader(t, res, app.ETag, "IGos54TQC8+mZ70zZAWQQg==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.golden.json"), shownWilt)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.headers.golden.json"), res.Header())
		// spew.Dump(res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok using expired IfModifiedSince header", func(t *testing.T) {
		// when
		ifModifiedSinceHeader := app.ToHTTPTime(wilt.UpdatedAt.Add(-1 * time.Hour))
		res, shownWilt := test.ShowWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, wilt.ID, &ifModifiedSinceHeader, nil)
		// then
		safeOverriteHeader(t, res, app.ETag, "IGos54TQC8+mZ70zZAWQQg==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_ifmodifiedsince_header.golden.json"), shownWilt)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_ifmodifiedsince_header.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok using expired IfNoneMatch header", func(t *testing.T) {
		// when
		ifNoneMatch := "foo"
		res, shownWilt := test.ShowWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, wilt.ID, nil, &ifNoneMatch)
		// then
		safeOverriteHeader(t, res, app.ETag, "IGos54TQC8+mZ70zZAWQQg==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_ifnonematch_header.golden.json"), shownWilt)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_ifnonematch_header.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified using IfModifiedSince header", func(t *testing.T) {
		// when
		ifModifiedSinceHeader := app.ToHTTPTime(wilt.UpdatedAt)
		res := test.ShowWorkItemLinkTypeNotModified(t, nil, nil, s.linkTypeCtrl, wilt.ID, &ifModifiedSinceHeader, nil)
		// then
		safeOverriteHeader(t, res, app.ETag, "IGos54TQC8+mZ70zZAWQQg==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_modified_using_ifmodifiedsince_header.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified using IfNoneMatch header", func(t *testing.T) {
		// when
		ifNoneMatch := app.GenerateEntityTag(wilt)
		res := test.ShowWorkItemLinkTypeNotModified(t, nil, nil, s.linkTypeCtrl, wilt.ID, nil, &ifNoneMatch)
		// then
		safeOverriteHeader(t, res, app.ETag, "IGos54TQC8+mZ70zZAWQQg==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_modified_using_ifnonematch_header.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("not found", func(t *testing.T) {
		res, jerrs := test.ShowWorkItemLinkTypeNotFound(t, nil, nil, s.linkTypeCtrl, uuid.FromStringOrNil("303badca-ea6b-440d-a964-1c925a174969"), nil, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_found.headers.golden.json"), res.Header())
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_found.golden.json"), jerrs)
	})
}
