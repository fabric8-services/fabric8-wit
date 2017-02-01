package remoteworkitem_test

import (
	"testing"

	"github.com/almighty/almighty-core/application"
	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type trackerRepoBlackBoxTest struct {
	gormsupport.DBTestSuite
	repo application.TrackerRepository
}

func TestRunTrackerRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &trackerRepoBlackBoxTest{DBTestSuite: gormsupport.NewDBTestSuite("../" + config.GetDefaultConfigurationFile())})
}

func (s *trackerRepoBlackBoxTest) SetupTest() {
	s.repo = remoteworkitem.NewTrackerRepository(s.DB)
}

func (s *trackerRepoBlackBoxTest) TestFailDeleteZeroID() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	// Create at least 1 item to avoid RowsEffectedCheck
	_, err := s.repo.Create(
		context.Background(),
		"http://api.github.com",
		remoteworkitem.ProviderGithub)

	if err != nil {
		s.T().Error("Could not create tracker", err)
	}

	err = s.repo.Delete(context.Background(), "0")
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}

func (s *trackerRepoBlackBoxTest) TestFailSaveZeroID() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	// Create at least 1 item to avoid RowsEffectedCheck
	tr, err := s.repo.Create(
		context.Background(),
		"http://api.github.com",
		remoteworkitem.ProviderGithub)

	if err != nil {
		s.T().Error("Could not create tracker", err)
	}
	tr.ID = "0"

	_, err = s.repo.Save(context.Background(), *tr)
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}

func (s *trackerRepoBlackBoxTest) TestFaiLoadZeroID() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	// Create at least 1 item to avoid RowsEffectedCheck
	_, err := s.repo.Create(
		context.Background(),
		"http://api.github.com",
		remoteworkitem.ProviderGithub)

	if err != nil {
		s.T().Error("Could not create tracker", err)
	}

	_, err = s.repo.Load(context.Background(), "0")
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}
