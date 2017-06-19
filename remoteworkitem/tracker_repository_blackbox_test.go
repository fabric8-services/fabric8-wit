package remoteworkitem_test

import (
	"testing"

	"context"

	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/remoteworkitem"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type trackerRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo application.TrackerRepository

	clean func()
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
func (s *trackerRepoBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)
}

func TestRunTrackerRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &trackerRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *trackerRepoBlackBoxTest) SetupTest() {
	s.repo = remoteworkitem.NewTrackerRepository(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (test *trackerRepoBlackBoxTest) TearDownTest() {
	test.clean()
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

	exists, err := s.repo.Exists(context.Background(), "0")
	require.False(s.T(), exists)
	require.IsType(s.T(), errors.NotFoundError{}, err)
}
