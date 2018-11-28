package account_test

import (
	"math/rand"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IdentityRepositoryTestSuite struct {
	gormtestsupport.DBTestSuite
	repo account.IdentityRepository
}

func TestIdentityRepository(t *testing.T) {
	suite.Run(t, &IdentityRepositoryTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *IdentityRepositoryTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = account.NewIdentityRepository(s.DB)
}

func (s *IdentityRepositoryTestSuite) TestQuery() {

	s.T().Run("without preload of user", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.Identities(1))
		id := fxt.Identities[0].ID
		// when
		ids, err := s.repo.Query(account.IdentityFilterByID(id))
		// then
		require.NoError(t, err)
		require.Len(t, ids, 1)
		assert.Equal(t, ids[0].ID, id)
		assert.Equal(t, uuid.Nil, ids[0].User.ID) // UUID will be `0000000-0000-0000-000000000000`
	})

	s.T().Run("with preload of user", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.Identities(1))
		id := fxt.Identities[0].ID
		// when
		ids, err := s.repo.Query(account.IdentityFilterByID(id), account.IdentityWithUser())
		// then
		require.NoError(t, err)
		require.Len(t, ids, 1)
		assert.Equal(t, ids[0].ID, id)
		assert.NotEqual(t, uuid.Nil, ids[0].User.ID)     // UUID will be set with the ID of the user
		assert.Equal(t, fxt.Users[0].ID, ids[0].User.ID) // UUID will be set with the ID of the user
	})
}

func (s *IdentityRepositoryTestSuite) TestOKToDelete() {
	// given
	identity := &account.Identity{
		ID:           uuid.NewV4(),
		Username:     "someuserTestIdentity",
		ProviderType: account.KeycloakIDP}

	identity2 := &account.Identity{
		ID:           uuid.NewV4(),
		Username:     "onemoreuserTestIdentity",
		ProviderType: account.KeycloakIDP}

	err := s.repo.Create(s.Ctx, identity)
	require.NoError(s.T(), err, "Could not create identity")
	err = s.repo.Create(s.Ctx, identity2)
	require.NoError(s.T(), err, "Could not create identity")
	// when
	err = s.repo.Delete(s.Ctx, identity.ID)
	// then
	require.NoError(s.T(), err)
	identities, err := s.repo.List(s.Ctx)
	require.NoError(s.T(), err, "Could not list identities")
	require.True(s.T(), len(identities) > 0)
	for _, ident := range identities {
		require.NotEqual(s.T(), "someuserTestIdentity", ident.Username)
	}
}

func randString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63() % int64(len(letterBytes))]
	}
	return string(b)
}

func (s *IdentityRepositoryTestSuite) TestOKToObfuscate() {
	// given
	identity := createAndLoad(s)
	obfStr := randString(6)
	// when
	err := s.repo.Obfuscate(s.Ctx, identity.ID, obfStr)
	// then
	require.NoError(s.T(), err, "Could not obfuscate identity")
	newIdentity, err := s.repo.Load(s.Ctx, identity.ID)
	require.NoError(s.T(), err, "Could not retrieve identity")
	require.Equal(s.T(), obfStr, newIdentity.Username)
}

func (s *IdentityRepositoryTestSuite) TestOKToLoad() {
	createAndLoad(s)
}

func (s *IdentityRepositoryTestSuite) TestExistsIdentity() {
	t := s.T()
	resource.Require(t, resource.Database)

	t.Run("identity exists", func(t *testing.T) {
		//t.Parallel()
		// given
		identity := createAndLoad(s)
		// when
		err := s.repo.CheckExists(s.Ctx, identity.ID)
		// then
		require.NoError(t, err, "Could not check if identity exists")
	})

	t.Run("identity doesn't exist", func(t *testing.T) {
		//t.Parallel()
		err := s.repo.CheckExists(s.Ctx, uuid.NewV4())
		// then

		require.IsType(t, errors.NotFoundError{}, err)
	})

}

func (s *IdentityRepositoryTestSuite) TestOKToSave() {
	// given
	identity := createAndLoad(s)
	// when
	identity.Username = "newusernameTestIdentity"
	err := s.repo.Save(s.Ctx, identity)
	// then
	require.NoError(s.T(), err, "Could not update identity")
}

func createAndLoad(s *IdentityRepositoryTestSuite) *account.Identity {
	identity := &account.Identity{
		ID:           uuid.NewV4(),
		Username:     "someuserTestIdentity2",
		ProviderType: account.KeycloakIDP}

	err := s.repo.Create(s.Ctx, identity)
	require.NoError(s.T(), err, "Could not create identity")
	// when
	idnt, err := s.repo.Load(s.Ctx, identity.ID)
	// then
	require.NoError(s.T(), err, "Could not load identity")
	require.Equal(s.T(), "someuserTestIdentity2", idnt.Username)
	return idnt
}
