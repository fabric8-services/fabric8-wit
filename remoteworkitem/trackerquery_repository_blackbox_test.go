package remoteworkitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type trackerQueryRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo   remoteworkitem.TrackerQueryRepository
	trRepo remoteworkitem.TrackerRepository
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
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Trackers(1), tf.Spaces(1))
	_, err := s.repo.Create(
		s.Ctx,
		"project = ARQ AND text ~ 'arquillian'",
		"15 * * * * *",
		fxt.Trackers[0].ID, fxt.Spaces[0].ID)
	require.NoError(s.T(), err)

	err = s.repo.Delete(s.Ctx, "0")
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}

func (s *trackerQueryRepoBlackBoxTest) TestFailSaveZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Trackers(1), tf.Spaces(1))

	tq, err := s.repo.Create(
		s.Ctx,
		"project = ARQ AND text ~ 'arquillian'",
		"15 * * * * *",
		fxt.Trackers[0].ID, fxt.Spaces[0].ID)
	require.NoError(s.T(), err)
	tq.ID = "0"

	_, err = s.repo.Save(s.Ctx, *tq)
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}

func (s *trackerQueryRepoBlackBoxTest) TestFaiLoadZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Trackers(1), tf.Spaces(1))

	_, err := s.repo.Create(
		s.Ctx,
		"project = ARQ AND text ~ 'arquillian'",
		"15 * * * * *",
		fxt.Trackers[0].ID, fxt.Spaces[0].ID)
	require.NoError(s.T(), err)

	_, err = s.repo.Load(s.Ctx, "0")
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}
