package account_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type identityBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo account.IdentityRepository
}

func TestRunIdentityBlackBoxTest(t *testing.T) {
	suite.Run(t, &identityBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *identityBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = account.NewIdentityRepository(s.DB)
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

func (s *identityBlackBoxTest) TestOKToLoad() {
	createAndLoad(s)
}

func (s *identityBlackBoxTest) TestExistsIdentity() {
	t := s.T()
	resource.Require(t, resource.Database)

	t.Run("identity exists", func(t *testing.T) {
		//t.Parallel()
		// given
		identity := createAndLoad(s)
		// when
		err := s.repo.CheckExists(s.Ctx, identity.ID.String())
		// then
		require.NoError(t, err, "Could not check if identity exists")
	})

	t.Run("identity doesn't exist", func(t *testing.T) {
		//t.Parallel()
		err := s.repo.CheckExists(s.Ctx, uuid.NewV4().String())
		// then

		require.IsType(t, errors.NotFoundError{}, err)
	})

}

func (s *identityBlackBoxTest) TestOKToSave() {
	// given
	identity := createAndLoad(s)
	// when
	identity.Username = "newusernameTestIdentity"
	err := s.repo.Save(s.Ctx, identity)
	// then
	require.NoError(s.T(), err, "Could not update identity")
}

func createAndLoad(s *identityBlackBoxTest) *account.Identity {
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
