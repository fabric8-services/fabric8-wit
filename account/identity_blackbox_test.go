package account_test

import (
	"context"
	"os"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type identityBlackBoxTest struct {
	gormsupport.DBTestSuite
	repo  account.IdentityRepository
	clean func()
}

func TestRunIdentityBlackBoxTest(t *testing.T) {
	suite.Run(t, &identityBlackBoxTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *identityBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if _, c := os.LookupEnv(resource.Database); c != false {
		if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
}

func (s *identityBlackBoxTest) SetupTest() {
	s.repo = account.NewIdentityRepository(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *identityBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *identityBlackBoxTest) TestOKToDelete() {
	// given
	identity := &account.Identity{
		ID:           uuid.NewV4(),
		Username:     "someuserTestIdentity",
		ProviderType: account.KeycloakIDP}

	identity2 := &account.Identity{
		ID:           uuid.NewV4(),
		Username:     "onemoreuserTestIdentity",
		ProviderType: account.KeycloakIDP}

	err := s.repo.Create(context.Background(), identity)
	require.Nil(s.T(), err, "Could not create identity")
	err = s.repo.Create(context.Background(), identity2)
	require.Nil(s.T(), err, "Could not create identity")
	// when
	err = s.repo.Delete(context.Background(), identity.ID)
	// then
	assert.Nil(s.T(), err)
	identities, err := s.repo.List(context.Background())
	require.Nil(s.T(), err, "Could not list identities")
	require.True(s.T(), len(identities.Data) > 0)
	for _, data := range identities.Data {
		require.NotEqual(s.T(), "someuserTestIdentity", data.Attributes.Username)
	}
}

func (s *identityBlackBoxTest) TestOKToLoad() {
	createAndLoad(s)
}

func (s *identityBlackBoxTest) TestOKToSave() {
	// given
	identity := createAndLoad(s)
	// when
	identity.Username = "newusernameTestIdentity"
	err := s.repo.Save(context.Background(), identity)
	// then
	require.Nil(s.T(), err, "Could not update identity")
}

func createAndLoad(s *identityBlackBoxTest) *account.Identity {
	identity := &account.Identity{
		ID:           uuid.NewV4(),
		Username:     "someuserTestIdentity2",
		ProviderType: account.KeycloakIDP}

	err := s.repo.Create(context.Background(), identity)
	require.Nil(s.T(), err, "Could not create identity")
	// when
	idnt, err := s.repo.Load(context.Background(), identity.ID)
	// then
	require.Nil(s.T(), err, "Could not load identity")
	require.Equal(s.T(), "someuserTestIdentity2", idnt.Username)
	return idnt
}
