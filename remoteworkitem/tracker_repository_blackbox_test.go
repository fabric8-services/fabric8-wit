package remoteworkitem_test

import (
	"os"
	"testing"

	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type trackerRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo application.TrackerRepository
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
func (s *trackerRepoBlackBoxTest) SetupSuite() {
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

func TestRunTrackerRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &trackerRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
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
