package remoteworkitem_test

import (
	"os"
	"testing"

	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/test/resource"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type trackerQueryRepoBlackBoxTest struct {
	gormsupport.DBTestSuite
	repo   application.TrackerQueryRepository
	trRepo application.TrackerRepository
	clean  func()
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
func (s *trackerQueryRepoBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if _, c := os.LookupEnv(resource.Database); c {
		if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
}

func TestRunTrackerQueryRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &trackerQueryRepoBlackBoxTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

func (s *trackerQueryRepoBlackBoxTest) SetupTest() {
	s.repo = remoteworkitem.NewTrackerQueryRepository(s.DB)
	s.trRepo = remoteworkitem.NewTrackerRepository(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *trackerQueryRepoBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *trackerQueryRepoBlackBoxTest) TestFailDeleteZeroID() {
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
