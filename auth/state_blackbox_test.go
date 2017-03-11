package auth_test

import (
	"os"
	"testing"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/workitem"

	"github.com/almighty/almighty-core/auth"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type stateBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo  auth.OauthStateReferenceRepository
	clean func()
	ctx   context.Context
}

func TestRunStateBlackBoxTest(t *testing.T) {
	suite.Run(t, &stateBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *stateBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if _, c := os.LookupEnv(resource.Database); c != false {
		if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			s.ctx = migration.NewMigrationContext(context.Background())
			return migration.PopulateCommonTypes(s.ctx, tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
}

func (s *stateBlackBoxTest) SetupTest() {
	s.repo = auth.NewOauthStateReferenceRepository(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *stateBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *stateBlackBoxTest) TestCreateDeleteLoad() {
	// given
	state := &auth.OauthStateReference{
		ID:       uuid.NewV4(),
		Referrer: "domain.org"}

	state2 := &auth.OauthStateReference{
		ID:       uuid.NewV4(),
		Referrer: "anotherdomain.com"}

	_, err := s.repo.Create(s.ctx, state)
	require.Nil(s.T(), err, "Could not create state reference")
	_, err = s.repo.Create(s.ctx, state2)
	require.Nil(s.T(), err, "Could not create state reference")
	// when
	err = s.repo.Delete(s.ctx, state.ID)
	// then
	assert.Nil(s.T(), err)
	_, err = s.repo.Load(s.ctx, state.ID)
	require.NotNil(s.T(), err)
	require.IsType(s.T(), errors.NotFoundError{}, err)

	foundState, err := s.repo.Load(s.ctx, state2.ID)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), foundState)
	require.True(s.T(), state2.Equal(*foundState))
}
