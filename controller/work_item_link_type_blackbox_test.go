package controller_test

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	. "github.com/fabric8-services/fabric8-wit/controller"
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
	suite.Run(t, &workItemLinkTypeSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func TestNewWorkItemLinkTypeControllerDBNull(t *testing.T) {
	require.Panics(t, func() {
		NewWorkItemLinkTypeController(nil, nil, nil)
	})
}

type workItemLinkTypeSuite struct {
	gormtestsupport.DBTestSuite
	testDir string
}

func (s *workItemLinkTypeSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.testDir = filepath.Join("test-files", "work_item_link_type")
}

func (s *workItemLinkTypeSuite) UnSecuredController() (*goa.Service, *WorkItemLinkTypeController) {
	svc := goa.New("WorkItemLinkType-Service")
	return svc, NewWorkItemLinkTypeController(svc, s.GormDB, s.Configuration)
}

func createWorkItemLinkTypeInRepo(t *testing.T, db application.DB, ctx context.Context, payload *app.CreateWorkItemLinkTypePayload) *app.WorkItemLinkTypeSingle {
	appLinkType := app.WorkItemLinkTypeSingle{
		Data: payload.Data,
	}
	modelLinkType, err := ConvertWorkItemLinkTypeToModel(appLinkType)
	require.NoError(t, err)
	var appLinkTypeResult app.WorkItemLinkTypeSingle
	err = application.Transactional(db, func(appl application.Application) error {
		createdModelLinkType, err := appl.WorkItemLinkTypes().Create(ctx, modelLinkType)
		if err != nil {
			return err
		}
		r := &http.Request{Host: "domain.io"}
		appLinkTypeResult = ConvertWorkItemLinkTypeFromModel(r, *createdModelLinkType)
		return nil
	})
	require.NoError(t, err)
	return &appLinkTypeResult
}

func (s *workItemLinkTypeSuite) TestShow() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemLinkTypes(1))
	_, ctrl := s.UnSecuredController()

	s.T().Run("ok", func(t *testing.T) {
		// when
		res, wilt := test.ShowWorkItemLinkTypeOK(s.T(), nil, nil, ctrl, fxt.WorkItemLinkTypes[0].ID, nil, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.res.payload.golden.json"), wilt)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("not found", func(t *testing.T) {
		// given
		id := uuid.NewV4()
		// when
		res, jerrs := test.ShowWorkItemLinkTypeNotFound(s.T(), nil, nil, ctrl, id, nil, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_found.res.payload.golden.json"), jerrs)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_found.res.headers.golden.json"), res.Header())
		// assertResponseHeaders(t, res)
	})

	s.T().Run("ok using expired IfModifiedSince header", func(t *testing.T) {
		// given
		ifModifiedSinceHeader := app.ToHTTPTime(fxt.WorkItemLinkTypes[0].UpdatedAt.Add(-1 * time.Hour))
		// when
		res, wilt := test.ShowWorkItemLinkTypeOK(s.T(), nil, nil, ctrl, fxt.WorkItemLinkTypes[0].ID, &ifModifiedSinceHeader, nil)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_ifmodifiedsince_header.res.payload.golden.json"), wilt)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_ifmodifiedsince_header.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok using expired IfNoneMatch header", func(t *testing.T) {
		// given
		ifNoneMatch := "foo"
		// when
		res, wilt := test.ShowWorkItemLinkTypeOK(s.T(), nil, nil, ctrl, fxt.WorkItemLinkTypes[0].ID, nil, &ifNoneMatch)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_ifnonematch_header.res.payload.golden.json"), wilt)
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok_using_expired_ifnonematch_header.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified using IfModifiedSince header", func(t *testing.T) {
		// given
		ifModifiedSinceHeader := app.ToHTTPTime(fxt.WorkItemLinkTypes[0].UpdatedAt)
		// when
		res := test.ShowWorkItemLinkTypeNotModified(s.T(), nil, nil, ctrl, fxt.WorkItemLinkTypes[0].ID, &ifModifiedSinceHeader, nil)
		// then
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_modified_using_ifmodifiedsince_header.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified using IfNoneMatch header", func(t *testing.T) {
		// given
		ifNoneMatch := app.GenerateEntityTag(fxt.WorkItemLinkTypes[0])
		// when
		res := test.ShowWorkItemLinkTypeNotModified(s.T(), nil, nil, ctrl, fxt.WorkItemLinkTypes[0].ID, nil, &ifNoneMatch)
		// then
		safeOverriteHeader(t, res, "Etag", "0icd7ov5CqwDXN6Fx9z18g==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_modified_using_ifnonematch_header.res.headers.golden.json"), res.Header())
		assertResponseHeaders(t, res)
	})
}
