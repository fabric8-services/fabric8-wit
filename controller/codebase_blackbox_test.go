package controller_test

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/dnaeon/go-vcr/recorder"
	"github.com/fabric8-services/fabric8-wit/account/tenant"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/codebase/che"
	"github.com/fabric8-services/fabric8-wit/configuration"

	"github.com/fabric8-services/fabric8-wit/account"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/test/http_monitor"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// a normal test function that will kick off TestSuiteCodebases
func TestCodebaseController(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &CodebaseControllerTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// ========== CodebaseControllerTestSuite struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
type CodebaseControllerTestSuite struct {
	gormtestsupport.DBTestSuite
	db      *gormapplication.GormDB
	testDir string
}

func (s *CodebaseControllerTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.db = gormapplication.NewGormDB(s.DB)
	s.testDir = filepath.Join("test-files", "codebase")
}

type ConfigureCodebaseController func(codebaseCtrl *CodebaseController)

func withCheClient(f CodebaseCheClientProvider) ConfigureCodebaseController {
	return func(codebaseCtrl *CodebaseController) {
		codebaseCtrl.NewCheClient = f
	}
}

func withShowTenant(f account.CodebaseInitTenantProvider) ConfigureCodebaseController {
	return func(codebaseCtrl *CodebaseController) {
		codebaseCtrl.ShowTenant = f
	}
}

func (s *CodebaseControllerTestSuite) UnsecuredController(settings ...ConfigureCodebaseController) (*goa.Service, *CodebaseController) {
	svc := goa.New("Codebases-service")
	codebaseCtrl := NewCodebaseController(svc, s.db, s.Configuration)
	for _, set := range settings {
		set(codebaseCtrl)
	}
	return svc, codebaseCtrl
}

func (s *CodebaseControllerTestSuite) SecuredControllers(identity account.Identity, settings ...ConfigureCodebaseController) (*goa.Service, *CodebaseController) {
	svc := testsupport.ServiceAsUser("Codebase-Service", identity)
	codebaseCtrl := NewCodebaseController(svc, s.db, s.Configuration)
	for _, set := range settings {
		set(codebaseCtrl)
	}
	return svc, codebaseCtrl
}

func NewMockCheClient(r http.RoundTripper, config *configuration.Registry) CodebaseCheClientProvider {
	return func(ctx context.Context, ns string) (che.Client, error) {
		h := &http.Client{
			Timeout:   1 * time.Second,
			Transport: r,
		}
		cheClient := che.NewStarterClient(config.GetCheStarterURL(), config.GetOpenshiftTenantMasterURL(), ns, h)
		return cheClient, nil
	}
}

func MockShowTenant() func(context.Context) (*tenant.TenantSingle, error) {
	return func(context.Context) (*tenant.TenantSingle, error) {
		// return a predefined response for the Tenant
		tenantType := "che"
		tenantNamespace := "foo"
		return &tenant.TenantSingle{
				Data: &tenant.Tenant{
					Attributes: &tenant.TenantAttributes{
						Namespaces: []*tenant.NamespaceAttributes{
							{
								Type: &tenantType,
								Name: &tenantNamespace,
							},
						},
					},
				},
			},
			nil
	}
}

func (s *CodebaseControllerTestSuite) TestShowCodebase() {
	resetFn := s.DisableGormCallbacks()
	defer resetFn()

	s.T().Run("success without stackId", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.Codebases(1))
		svc, ctrl := s.UnsecuredController()
		// when
		_, result := test.ShowCodebaseOK(t, svc.Context, svc, ctrl, fxt.Codebases[0].ID)
		// then
		require.NotNil(t, result)
		compareWithGoldenUUIDAgnostic(t, filepath.Join(s.testDir, "show", "ok_without_stackId.golden.json"), result)
	})

	s.T().Run("success with stackId", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.Codebases(1, func(fxt *tf.TestFixture, idx int) error {
			stackID := "golang-default"
			fxt.Codebases[idx].StackID = &stackID
			return nil
		}))
		svc, ctrl := s.UnsecuredController()
		// when
		_, result := test.ShowCodebaseOK(t, svc.Context, svc, ctrl, fxt.Codebases[0].ID)
		// then
		require.NotNil(t, result)
		compareWithGoldenUUIDAgnostic(t, filepath.Join(s.testDir, "show", "ok_with_stackId.golden.json"), result)
	})
}

func (s *CodebaseControllerTestSuite) TestDeleteCodebase() {
	resetFn := s.DisableGormCallbacks()
	defer resetFn()

	s.T().Run("OK", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.Spaces(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.Spaces[idx].OwnerID = testsupport.TestIdentity.ID
				return nil
			}),
			tf.Codebases(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.Codebases[idx].URL = "git@github.com:bar/foo"
				return nil
			}))
		// setup the mock client for Che
		r, err := recorder.New("../test/data/che/che_delete_codebase_workspaces.ok")
		require.NoError(t, err)
		defer r.Stop()
		m := httpmonitor.NewTransportMonitor(r.Transport)
		svc, ctrl := s.SecuredControllers(testsupport.TestIdentity, withCheClient(NewMockCheClient(m, s.Configuration)), withShowTenant(MockShowTenant()))
		// when
		test.DeleteCodebaseNoContent(t, svc.Context, svc, ctrl, fxt.Codebases[0].ID)
		// verify that a `DELETE workspace` request was sent by the Che client
		err = m.ValidateExchanges(
			httpmonitor.Exchange{
				RequestMethod: "GET",
				RequestURL:    "che-server/workspace?masterUrl=https://tsrv.devshift.net:8443&namespace=foo&repository=git@github.com:bar/foo",
				StatusCode:    200,
			},
			httpmonitor.Exchange{
				RequestMethod: "DELETE",
				RequestURL:    "che-server/workspace/string?masterUrl=https://tsrv.devshift.net:8443&namespace=foo",
				StatusCode:    200,
			})
		require.NoError(t, err)

	})

	s.T().Run("OK with workspace deletion failure", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.Spaces(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.Spaces[idx].OwnerID = testsupport.TestIdentity.ID
				return nil
			}),
			tf.Codebases(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.Codebases[idx].URL = "git@github.com:bar/foo"
				return nil
			}))
		// setup the mock client for Che
		r, err := recorder.New("../test/data/che/che_delete_codebase_workspaces.failure")
		require.NoError(t, err)
		defer r.Stop()
		m := httpmonitor.NewTransportMonitor(r.Transport)
		svc, ctrl := s.SecuredControllers(testsupport.TestIdentity, withCheClient(NewMockCheClient(m, s.Configuration)), withShowTenant(MockShowTenant()))
		// when
		test.DeleteCodebaseNoContent(t, svc.Context, svc, ctrl, fxt.Codebases[0].ID)
		// then verify that the Che client emitted the expected requests
		err = m.ValidateExchanges(
			httpmonitor.Exchange{
				RequestMethod: "GET",
				RequestURL:    "che-server/workspace?masterUrl=https://tsrv.devshift.net:8443&namespace=foo&repository=git@github.com:bar/foo",
				StatusCode:    200,
			},
			httpmonitor.Exchange{
				RequestMethod: "DELETE",
				RequestURL:    "che-server/workspace/string?masterUrl=https://tsrv.devshift.net:8443&namespace=foo",
				StatusCode:    500,
			})
		require.NoError(t, err)
	})

	s.T().Run("NotFound", func(t *testing.T) {
		// given
		codebaseID := uuid.NewV4()
		r, err := recorder.New("")
		require.NoError(t, err)
		defer r.Stop()
		m := httpmonitor.NewTransportMonitor(r.Transport)
		svc, ctrl := s.SecuredControllers(testsupport.TestIdentity, withCheClient(NewMockCheClient(m, s.Configuration)), withShowTenant(MockShowTenant()))
		// when (codebase does not exist)
		test.DeleteCodebaseNotFound(t, svc.Context, svc, ctrl, codebaseID)
		// then nothing should be sent to Che
		err = m.ValidateNoExchanges()
		require.NoError(t, err)
	})

	s.T().Run("Unauthorized on non-existing codebase", func(t *testing.T) {
		// given
		codebaseID := uuid.NewV4()
		r, err := recorder.New("")
		require.NoError(t, err)
		defer r.Stop()
		m := httpmonitor.NewTransportMonitor(r.Transport)
		svc, ctrl := s.UnsecuredController(withCheClient(NewMockCheClient(m, s.Configuration)), withShowTenant(MockShowTenant()))
		// when (user is not authenticated)
		test.DeleteCodebaseUnauthorized(t, svc.Context, svc, ctrl, codebaseID)
		// then nothing should be sent to Che
		err = m.ValidateNoExchanges()
		require.NoError(t, err)
	})

	s.T().Run("Unauthorized on existing codebase", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.Spaces(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.Spaces[idx].OwnerID = testsupport.TestIdentity2.ID
				return nil
			}),
			tf.Codebases(1))
		// when/then (user is not authenticated)
		r, err := recorder.New("")
		require.NoError(t, err)
		defer r.Stop()
		m := httpmonitor.NewTransportMonitor(r.Transport)
		svc, ctrl := s.UnsecuredController(withCheClient(NewMockCheClient(m, s.Configuration)), withShowTenant(MockShowTenant()))
		test.DeleteCodebaseUnauthorized(t, svc.Context, svc, ctrl, fxt.Codebases[0].ID)
		// then nothing should be sent to Che
		err = m.ValidateNoExchanges()
		require.NoError(t, err)
	})

	s.T().Run("Forbidden", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.Spaces(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.Spaces[idx].OwnerID = testsupport.TestIdentity.ID
				return nil
			}),
			tf.Codebases(1))
		r, err := recorder.New("")
		require.NoError(t, err)
		defer r.Stop()
		m := httpmonitor.NewTransportMonitor(r.Transport)
		// when (user is not space owner)
		svc, ctrl := s.SecuredControllers(testsupport.TestIdentity2, withCheClient(NewMockCheClient(m, s.Configuration)), withShowTenant(MockShowTenant()))
		test.DeleteCodebaseForbidden(t, svc.Context, svc, ctrl, fxt.Codebases[0].ID)
		// then nothing should be sent to Che
		err = m.ValidateNoExchanges()
		require.NoError(t, err)
	})

}
