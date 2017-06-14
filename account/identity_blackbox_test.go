package account_test

import (
	"context"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type identityBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo  account.IdentityRepository
	clean func()
	ctx   context.Context
}

func TestRunIdentityBlackBoxTest(t *testing.T) {
	suite.Run(t, &identityBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *identityBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)
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

	err := s.repo.Create(s.ctx, identity)
	require.Nil(s.T(), err, "Could not create identity")
	err = s.repo.Create(s.ctx, identity2)
	require.Nil(s.T(), err, "Could not create identity")
	// when
	err = s.repo.Delete(s.ctx, identity.ID)
	// then
	assert.Nil(s.T(), err)
	identities, err := s.repo.List(s.ctx)
	require.Nil(s.T(), err, "Could not list identities")
	require.True(s.T(), len(identities) > 0)
	for _, ident := range identities {
		require.NotEqual(s.T(), "someuserTestIdentity", ident.Username)
	}
}

func (s *identityBlackBoxTest) TestOKToLoad() {
	createAndLoad(s)
}

func (s *identityBlackBoxTest) TestExistsIdentity() {
	t := s.T()
	resource.Require(t, resource.Database)

	t.Run("identity exists", func(t *testing.T) {
		t.Parallel()
		// given
		identity := createAndLoad(s)
		// when
		exists, err := s.repo.Exists(s.ctx, identity.ID.String())
		// then
		require.Nil(t, err, "Could not check if identity exists")
		require.True(t, exists)
	})

	t.Run("identity doesn't exists", func(t *testing.T) {
		t.Parallel()
		exists, err := s.repo.Exists(s.ctx, uuid.NewV4().String())
		// then
		require.IsType(t, errors.NotFoundError{}, err)
		require.False(t, exists)
	})

}

func (s *identityBlackBoxTest) TestOKToSave() {
	// given
	identity := createAndLoad(s)
	// when
	identity.Username = "newusernameTestIdentity"
	err := s.repo.Save(s.ctx, identity)
	// then
	require.Nil(s.T(), err, "Could not update identity")
}

func createAndLoad(s *identityBlackBoxTest) *account.Identity {
	identity := &account.Identity{
		ID:           uuid.NewV4(),
		Username:     "someuserTestIdentity2",
		ProviderType: account.KeycloakIDP}

	err := s.repo.Create(s.ctx, identity)
	require.Nil(s.T(), err, "Could not create identity")
	// when
	idnt, err := s.repo.Load(s.ctx, identity.ID)
	// then
	require.Nil(s.T(), err, "Could not load identity")
	require.Equal(s.T(), "someuserTestIdentity2", idnt.Username)
	return idnt
}
