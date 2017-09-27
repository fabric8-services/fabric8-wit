package remoteworkitem_test

import (
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type trackerQueryRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo   application.TrackerQueryRepository
	trRepo application.TrackerRepository
}

func TestRunTrackerQueryRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &trackerQueryRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *trackerQueryRepoBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = remoteworkitem.NewTrackerQueryRepository(s.DB)
	s.trRepo = remoteworkitem.NewTrackerRepository(s.DB)
}

func (s *trackerQueryRepoBlackBoxTest) TestFailDeleteZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Trackers(1), tf.Spaces(1))
	trackerID := fmt.Sprintf("%s", testFxt.Trackers[0].ID)
	_, err := s.repo.Create(
		s.Ctx,
		"project = ARQ AND text ~ 'arquillian'",
		"15 * * * * *",
		trackerID, testFxt.Spaces[0].ID)
	if err != nil {
		s.T().Error("Could not create tracker query", err)
	}

	err = s.repo.Delete(s.Ctx, "0")
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}

func (s *trackerQueryRepoBlackBoxTest) TestFailSaveZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Trackers(1), tf.Spaces(1))
	trackerID := fmt.Sprintf("%s", testFxt.Trackers[0].ID)

	tq, err := s.repo.Create(
		s.Ctx,
		"project = ARQ AND text ~ 'arquillian'",
		"15 * * * * *",
		trackerID, testFxt.Spaces[0].ID)
	if err != nil {
		s.T().Error("Could not create tracker query", err)
	}
	tq.ID = "0"

	_, err = s.repo.Save(s.Ctx, *tq)
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}

func (s *trackerQueryRepoBlackBoxTest) TestFaiLoadZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	testFxt := tf.NewTestFixture(s.T(), s.DB, tf.Trackers(1), tf.Spaces(1))
	trackerID := fmt.Sprintf("%s", testFxt.Trackers[0].ID)

	_, err := s.repo.Create(
		s.Ctx,
		"project = ARQ AND text ~ 'arquillian'",
		"15 * * * * *",
		trackerID, testFxt.Spaces[0].ID)
	if err != nil {
		s.T().Error("Could not create tracker query", err)
	}

	_, err = s.repo.Load(s.Ctx, "0")
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}
