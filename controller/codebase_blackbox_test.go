package controller_test

import (
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/controller"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
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

func (s *CodebaseControllerTestSuite) UnsecuredController() (*goa.Service, *CodebaseController) {
	svc := goa.New("Codebases-service")
	return svc, NewCodebaseController(svc, s.db, s.Configuration)
}

func (s *CodebaseControllerTestSuite) SecuredControllers(identity account.Identity) (*goa.Service, *CodebaseController) {
	svc := testsupport.ServiceAsUser("Codebase-Service", identity)
	return svc, controller.NewCodebaseController(svc, s.db, s.Configuration)
}

func (s *CodebaseControllerTestSuite) TestShowCodebase() {
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
			tf.Codebases(1))
		// when/then
		svc, ctrl := s.SecuredControllers(testsupport.TestIdentity)
		test.DeleteCodebaseNoContent(t, svc.Context, svc, ctrl, fxt.Codebases[0].ID)
	})

	s.T().Run("NotFound", func(t *testing.T) {
		// given
		codebaseID := uuid.NewV4()
		// when/then (codebase does not exist)
		svc, ctrl := s.SecuredControllers(testsupport.TestIdentity)
		test.DeleteCodebaseNotFound(t, svc.Context, svc, ctrl, codebaseID)
	})

	s.T().Run("Unauthorized on non-existing codebase", func(t *testing.T) {
		// given
		codebaseID := uuid.NewV4()
		// when/then (user is not authenticated)
		svc, ctrl := s.UnsecuredController()
		test.DeleteCodebaseUnauthorized(t, svc.Context, svc, ctrl, codebaseID)
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
		svc, ctrl := s.UnsecuredController()
		test.DeleteCodebaseUnauthorized(t, svc.Context, svc, ctrl, fxt.Codebases[0].ID)
	})

	s.T().Run("Forbidden", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.Spaces(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.Spaces[idx].OwnerID = testsupport.TestIdentity.ID
				return nil
			}),
			tf.Codebases(1))
		// when/then (user is not space owner)
		svc, ctrl := s.SecuredControllers(testsupport.TestIdentity2)
		test.DeleteCodebaseForbidden(t, svc.Context, svc, ctrl, fxt.Codebases[0].ID)
	})

}
