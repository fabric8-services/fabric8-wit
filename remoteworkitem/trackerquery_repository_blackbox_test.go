package remoteworkitem_test

import (
	"testing"

	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type trackerQueryRepoBlackBoxTest struct {
	gormsupport.DBTestSuite
	repo   application.TrackerQueryRepository
	trRepo application.TrackerRepository
}

func TestRunTrackerQueryRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &trackerQueryRepoBlackBoxTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

func (s *trackerQueryRepoBlackBoxTest) SetupTest() {
	s.repo = remoteworkitem.NewTrackerQueryRepository(s.DB)
	s.trRepo = remoteworkitem.NewTrackerRepository(s.DB)
}

func (s *trackerQueryRepoBlackBoxTest) TestFailDeleteZeroID() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	// Create at least 1 item to avoid RowsEffectedCheck
	tr, err := s.trRepo.Create(
		context.Background(),
		"http://api.github.com",
		remoteworkitem.ProviderGithub)
	if err != nil {
		s.T().Error("Could not create tracker", err)
	}

	_, err = s.repo.Create(
		context.Background(),
		"project = ARQ AND text ~ 'arquillian'",
		"15 * * * * *",
		tr.ID)
	if err != nil {
		s.T().Error("Could not create tracker query", err)
	}

	err = s.repo.Delete(context.Background(), "0")
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}

func (s *trackerQueryRepoBlackBoxTest) TestFailSaveZeroID() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	// Create at least 1 item to avoid RowsEffectedCheck
	tr, err := s.trRepo.Create(
		context.Background(),
		"http://api.github.com",
		remoteworkitem.ProviderGithub)
	if err != nil {
		s.T().Error("Could not create tracker", err)
	}

	tq, err := s.repo.Create(
		context.Background(),
		"project = ARQ AND text ~ 'arquillian'",
		"15 * * * * *",
		tr.ID)
	if err != nil {
		s.T().Error("Could not create tracker query", err)
	}
	tq.ID = "0"

	_, err = s.repo.Save(context.Background(), *tq)
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}

func (s *trackerQueryRepoBlackBoxTest) TestFaiLoadZeroID() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	// Create at least 1 item to avoid RowsEffectedCheck
	tr, err := s.trRepo.Create(
		context.Background(),
		"http://api.github.com",
		remoteworkitem.ProviderGithub)
	if err != nil {
		s.T().Error("Could not create tracker", err)
	}

	_, err = s.repo.Create(
		context.Background(),
		"project = ARQ AND text ~ 'arquillian'",
		"15 * * * * *",
		tr.ID)
	if err != nil {
		s.T().Error("Could not create tracker query", err)
	}

	_, err = s.repo.Load(context.Background(), "0")
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}
