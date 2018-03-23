package controller_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestEvent struct {
	gormtestsupport.DBTestSuite
	db      *gormapplication.GormDB
	testDir string
}

func TestRunEvent(t *testing.T) {
	resource.Require(t, resource.Database)
	pwd, err := os.Getwd()
	require.NoError(t, err)
	suite.Run(t, &TestEvent{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (s *TestEvent) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.db = gormapplication.NewGormDB(s.DB)
	s.testDir = filepath.Join("test-files", "event")
}

func (s *TestEvent) TestListEvent() {

	s.T().Run("event list ok - assigned", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewWorkItemEventsController(svc, s.db, s.Configuration)
		assignee := []string{fxt.Identities[0].ID.String()}
		fxt.WorkItems[0].Fields[workitem.SystemAssignees] = assignee
		err := application.Transactional(s.db, func(app application.Application) error {
			_, err := app.WorkItems().Save(context.Background(), fxt.Spaces[0].ID, *fxt.WorkItems[0], fxt.Identities[0].ID)
			return err
		})
		require.NoError(t, err)
		_, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-assignees.res.golden.json"), eventList)
	})

	s.T().Run("event list ok - label", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewWorkItemEventsController(svc, s.db, s.Configuration)
		label := []string{"label1"}
		fxt.WorkItems[0].Fields[workitem.SystemLabels] = label
		err := application.Transactional(s.db, func(app application.Application) error {
			_, err := app.WorkItems().Save(context.Background(), fxt.Spaces[0].ID, *fxt.WorkItems[0], fxt.Identities[0].ID)
			return err
		})
		require.NoError(t, err)
		_, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-labels.res.golden.json"), eventList)
	})

	s.T().Run("event list ok - iteration", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewWorkItemEventsController(svc, s.db, s.Configuration)
		fxt.WorkItems[0].Fields[workitem.SystemIteration] = fxt.Iterations[0].ID.String()
		err := application.Transactional(s.db, func(app application.Application) error {
			_, err := app.WorkItems().Save(context.Background(), fxt.Spaces[0].ID, *fxt.WorkItems[0], fxt.Identities[0].ID)
			return err
		})
		require.NoError(t, err)
		_, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-iteration.res.golden.json"), eventList)
	})

	s.T().Run("event list - empty", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewWorkItemEventsController(svc, s.db, s.Configuration)
		_, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-no-event.res.errors.golden.json"), eventList)
	})
}
