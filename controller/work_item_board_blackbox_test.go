package controller_test

import (
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/suite"
)

type workItemBoardSuite struct {
	gormtestsupport.DBTestSuite
	svc     *goa.Service
	ctrl    *WorkItemBoardController
	testDir string
}

func TestWorkItemBoardSuite(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemBoardSuite{
		DBTestSuite: gormtestsupport.NewDBTestSuite(),
	})
}

// The SetupTest method will be run before every test in the suite.
func (s *workItemBoardSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.svc = testsupport.ServiceAsUser("Board-Service", testsupport.TestIdentity)
	s.ctrl = NewWorkItemBoardController(s.svc, gormapplication.NewGormDB(s.DB))
	s.testDir = filepath.Join("test-files", "work_item_board")
}

func (s *workItemBoardSuite) TestShow() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemBoards(1))
		// when
		res, group := test.ShowWorkItemBoardOK(t, nil, s.svc, s.ctrl, fxt.WorkItemBoards[0].ID)
		// then
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.board.golden.json"), group)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "ok.headers.golden.json"), res.Header())
	})
	s.T().Run("not found", func(t *testing.T) {
		// given
		boardID := uuid.NewV4()
		// when
		res, jerrs := test.ShowWorkItemBoardNotFound(t, nil, s.svc, s.ctrl, boardID)
		// then
		ignoreMe := "IGNOREME"
		jerrs.Errors[0].ID = &ignoreMe
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_found.errors.golden.json"), jerrs)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "show", "not_found.headers.golden.json"), res.Header())
	})
}
