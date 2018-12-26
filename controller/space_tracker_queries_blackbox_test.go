package controller_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestSpaceTrackerQueries struct {
	gormtestsupport.DBTestSuite
	svcSpaceTrackerQueries  *goa.Service
	ctrlSpaceTrackerQueries *SpaceTrackerQueriesController
	db                      *gormapplication.GormDB
	RwiScheduler            *remoteworkitem.Scheduler
}

func TestRunSpaceTrackerQueries(t *testing.T) {
	suite.Run(t, &TestSpaceTrackerQueries{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *TestSpaceTrackerQueries) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.RwiScheduler = remoteworkitem.NewScheduler(s.DB)
	s.db = gormapplication.NewGormDB(s.DB)
}

func (s *TestSpaceTrackerQueries) SecuredController() (*goa.Service, *SpaceTrackerQueriesController, *TrackerqueryController) {
	svc := testsupport.ServiceAsUser("SpaceTrackerQuery-Service", testsupport.TestIdentity)
	return svc, NewSpaceTrackerQueriesController(svc, s.db, s.Configuration), NewTrackerqueryController(svc, s.db, s.RwiScheduler, s.Configuration)
}

func (s *TestSpaceTrackerQueries) TestListSpaceTrackerQueriesOK() {
	resource.Require(s.T(), resource.Database)
	svc, spaceTrackerQueriesCtrl, trackerQueryCtrl := s.SecuredController()
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(2), tf.Trackers(1), tf.WorkItemTypes(1))
	require.NotNil(s.T(), fxt.Spaces[0], fxt.Trackers[0])

	// Create two TrackerQueries from first space
	tqpayload1 := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID, fxt.WorkItemTypes[0].ID)
	_, tq1 := test.CreateTrackerqueryCreated(s.T(), svc.Context, svc, trackerQueryCtrl, &tqpayload1)
	require.NotNil(s.T(), tq1)

	tqpayload2 := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID, fxt.WorkItemTypes[0].ID)
	_, tq2 := test.CreateTrackerqueryCreated(s.T(), svc.Context, svc, trackerQueryCtrl, &tqpayload2)
	require.NotNil(s.T(), tq2)

	// Create one TrackerQuery from second space
	tqpayload3 := newCreateTrackerQueryPayload(fxt.Spaces[1].ID, fxt.Trackers[0].ID, fxt.WorkItemTypes[0].ID)
	_, tq3 := test.CreateTrackerqueryCreated(s.T(), svc.Context, svc, trackerQueryCtrl, &tqpayload3)
	assert.NotNil(s.T(), tq3)

	// list TrackerQueries
	_, list := test.ListSpaceTrackerQueriesOK(s.T(), svc.Context, svc, spaceTrackerQueriesCtrl, fxt.Spaces[0].ID, nil, nil)
	require.NotNil(s.T(), list.Data)
	require.Len(s.T(), list.Data, 2)
	require.Equal(s.T(), list.Data[0].ID, tq1.Data.ID)
	require.Equal(s.T(), list.Data[1].ID, tq2.Data.ID)
}
