package account_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type userBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo account.UserRepository
}

func TestRunUserBlackBoxTest(t *testing.T) {
	suite.Run(t, &userBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *userBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = account.NewUserRepository(s.DB)
}

func (s *userBlackBoxTest) TestOKToDelete() {
	t := s.T()
	resource.Require(t, resource.Database)

	// create 2 users, where the first one would be deleted.
	user := createAndLoadUser(s)
	createAndLoadUser(s)

	err := s.repo.Delete(s.Ctx, user.ID)
	require.NoError(s.T(), err)

	// lets see how many are present.
	users, err := s.repo.List(s.Ctx)
	require.NoError(s.T(), err, "Could not list users")
	require.True(s.T(), len(users) > 0)

	for _, data := range users {
		// The user 'user' was deleted and rest were not deleted, hence we check
		// that none of the user objects returned include the one deleted.
		require.NotEqual(s.T(), user.ID.String(), data.ID.String())
	}
}

func (s *userBlackBoxTest) TestOKToLoad() {
	t := s.T()
	resource.Require(t, resource.Database)

	createAndLoadUser(s) // this function does the needful already
}

func (s *userBlackBoxTest) TestExistsUser() {
	t := s.T()
	resource.Require(t, resource.Database)

	t.Run("user exists", func(t *testing.T) {
		//t.Parallel()
		user := createAndLoadUser(s)
		// when
		err := s.repo.CheckExists(s.Ctx, user.ID.String())
		// then
		require.NoError(t, err)
	})

	t.Run("user doesn't exist", func(t *testing.T) {
		//t.Parallel()
		// Check not existing
		err := s.repo.CheckExists(s.Ctx, uuid.NewV4().String())
		// then
		//
		require.IsType(s.T(), errors.NotFoundError{}, err)
	})
}

func (s *userBlackBoxTest) TestOKToSave() {
	t := s.T()
	resource.Require(t, resource.Database)

	user := createAndLoadUser(s)

	user.FullName = "newusernameTestUser"
	err := s.repo.Save(s.Ctx, user)
	require.NoError(s.T(), err, "Could not update user")

	updatedUser, err := s.repo.Load(s.Ctx, user.ID)
	require.NoError(s.T(), err, "Could not load user")
	assert.Equal(s.T(), user.FullName, updatedUser.FullName)
	fields := user.ContextInformation
	assert.Equal(s.T(), fields["last_visited"], "http://www.google.com")
	assert.Equal(s.T(), fields["myid"], "71f343e3-2bfa-4ec6-86d4-79b91476acfc")

}

func (s *userBlackBoxTest) TestUpdateToEmptyString() {
	t := s.T()
	resource.Require(t, resource.Database)
	user := createAndLoadUser(s)

	err := s.repo.Save(s.Ctx, user)
	require.NoError(t, err)
	user.Bio = ""
	err = s.repo.Save(s.Ctx, user)
	require.NoError(t, err)
	u, err := s.repo.Load(s.Ctx, user.ID)
	require.NoError(t, err)
	require.Empty(t, u.Bio)
}

func createAndLoadUser(s *userBlackBoxTest) *account.User {
	user := &account.User{
		ID:       uuid.NewV4(),
		Email:    "someuser@TestUser" + uuid.NewV4().String(),
		FullName: "someuserTestUser" + uuid.NewV4().String(),
		ImageURL: "someImageUrl" + uuid.NewV4().String(),
		Bio:      "somebio" + uuid.NewV4().String(),
		URL:      "someurl" + uuid.NewV4().String(),
		ContextInformation: account.ContextInformation{
			"space":        uuid.NewV4(),
			"last_visited": "http://www.google.com",
			"myid":         "71f343e3-2bfa-4ec6-86d4-79b91476acfc",
		},
	}

	err := s.repo.Create(s.Ctx, user)
	require.NoError(s.T(), err, "Could not create user")

	createdUser, err := s.repo.Load(s.Ctx, user.ID)
	require.NoError(s.T(), err, "Could not load user")
	require.Equal(s.T(), user.Email, createdUser.Email)
	require.Equal(s.T(), user.ID, createdUser.ID)
	require.Equal(s.T(), user.ContextInformation["last_visited"], createdUser.ContextInformation["last_visited"])

	return createdUser
}
