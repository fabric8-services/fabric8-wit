package remoteworkitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	uuid "github.com/satori/go.uuid"

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
	tq := remoteworkitem.TrackerQuery{
		Query:     "project=ARQ AND test ~ 'arquillian'",
		Schedule:  "15 * * * * *",
		TrackerID: fxt.Trackers[0].ID,
		SpaceID:   fxt.Spaces[0].ID,
	}
	err := s.repo.Create(
		s.Ctx,
		&tq)
	require.Nil(s.T(), err)

	err = s.repo.Delete(s.Ctx, uuid.NewV4())
	require.IsType(s.T(), errors.NotFoundError{}, err)
}

func (s *trackerQueryRepoBlackBoxTest) TestFailSaveZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Trackers(1), tf.Spaces(1))

	tq := remoteworkitem.TrackerQuery{
		Query:     "project=ARQ AND test ~ 'arquillian'",
		Schedule:  "15 * * * * *",
		TrackerID: fxt.Trackers[0].ID,
		SpaceID:   fxt.Spaces[0].ID,
	}
	err := s.repo.Create(
		s.Ctx,
		&tq)
	require.Nil(s.T(), err)
	tq.ID = uuid.NewV4()

	_, err = s.repo.Save(s.Ctx, tq)
	require.IsType(s.T(), errors.NotFoundError{}, err)
}

func (s *trackerQueryRepoBlackBoxTest) TestFaiLoadZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Trackers(1), tf.Spaces(1))

	tq := remoteworkitem.TrackerQuery{
		Query:     "project=ARQ AND test ~ 'arquillian'",
		Schedule:  "15 * * * * *",
		TrackerID: fxt.Trackers[0].ID,
		SpaceID:   fxt.Spaces[0].ID,
	}
	err := s.repo.Create(
		s.Ctx,
		&tq)
	require.Nil(s.T(), err)

	_, err = s.repo.Load(s.Ctx, uuid.NewV4())
	require.IsType(s.T(), errors.NotFoundError{}, err)
}
