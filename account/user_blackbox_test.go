package account_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type userBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo account.UserRepository
}

func TestRunUserBlackBoxTest(t *testing.T) {
	suite.Run(t, &userBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *userBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = account.NewUserRepository(s.DB)
}

func (s *userBlackBoxTest) TestOKToDelete() {
	t := s.T()
	resource.Require(t, resource.Database)

	// create 2 users, where the first one would be deleted.
	user := createAndLoadUser(s, t)
	createAndLoadUser(s, t)

	err := s.repo.Delete(s.Ctx, user.ID)
	require.NoError(t, err)

	// lets see how many are present.
	users, err := s.repo.List(s.Ctx)
	require.NoError(t, err, "Could not list users")
	require.True(t, len(users) > 0)

	for _, data := range users {
		// The user 'user' was deleted and rest were not deleted, hence we check
		// that none of the user objects returned include the one deleted.
		require.NotEqual(t, user.ID.String(), data.ID.String())
	}
}

func (s *userBlackBoxTest) TestOKToLoad() {
	t := s.T()
	resource.Require(t, resource.Database)

	createAndLoadUser(s, t) // this function does the needful already
}

func (s *userBlackBoxTest) TestLoadByUsername() {
	t := s.T()
	resource.Require(t, resource.Database)
	t.Run("load ok", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.Identities(1, func(fixture *tf.TestFixture, idx int) error {
			fixture.Identities[idx].Username = "myusername"
			return nil
		}), tf.Users(1))
		loadedUser, err := s.repo.LoadByUsername(s.Ctx, "myusername")
		require.NoError(t, err, "Could not load user")
		require.Equal(t, loadedUser[0].Email, fxt.Users[0].Email)
		require.Equal(t, loadedUser[0].ID, fxt.Users[0].ID)
	})
	t.Run("load one user with a list of identities associated to a this user", func(t *testing.T) {
		random := uuid.NewV4()
		myusername := "myusername" + random.String()
		fxt := tf.NewTestFixture(t, s.DB, tf.Users(1), tf.Identities(5, func(fixture *tf.TestFixture, idx int) error {
			fixture.Identities[idx].Username = myusername
			return nil
		}))
		loadedUser, err := s.repo.LoadByUsername(s.Ctx, myusername)
		require.NoError(t, err, "Could not load user")
		require.Equal(t, len(loadedUser), 1)
		require.Equal(t, loadedUser[0].ID, fxt.Users[0].ID)
	})
	t.Run("load users with a list of identities associated to a those users", func(t *testing.T) {
		random := uuid.NewV4()
		myusername := "myusername" + random.String()
		numUsers := 3
		tf.NewTestFixture(t, s.DB,
			tf.Users(numUsers),
			tf.Identities(5, func(fixture *tf.TestFixture, idx int) error {
				fixture.Identities[idx].Username = myusername
				fixture.Identities[idx].User = *fixture.Users[idx%numUsers]
				return nil
			}),
		)
		loadedUser, err := s.repo.LoadByUsername(s.Ctx, myusername)
		require.NoError(t, err, "Could not load user")
		require.Len(t, loadedUser, numUsers)
	})
}

func (s *userBlackBoxTest) TestExistsUser() {
	t := s.T()
	resource.Require(t, resource.Database)

	t.Run("user exists", func(t *testing.T) {
		//t.Parallel()
		user := createAndLoadUser(s, t)
		// when
		err := s.repo.CheckExists(s.Ctx, user.ID)
		// then
		require.NoError(t, err)
	})

	t.Run("user doesn't exist", func(t *testing.T) {
		//t.Parallel()
		// Check not existing
		err := s.repo.CheckExists(s.Ctx, uuid.NewV4())
		// then
		//
		require.IsType(t, errors.NotFoundError{}, err)
	})
}

func (s *userBlackBoxTest) TestSave() {
	t := s.T()
	resource.Require(t, resource.Database)
	user := createAndLoadUser(s, t)
	t.Run("save is ok", func(t *testing.T) {
		user.FullName = "newusernameTestUser"
		err := s.repo.Save(s.Ctx, user)
		require.NoError(t, err, "Could not update user")

		updatedUser, err := s.repo.Load(s.Ctx, user.ID)
		require.NoError(t, err, "Could not load user")
		assert.Equal(t, user.FullName, updatedUser.FullName)
		fields := user.ContextInformation
		assert.Equal(t, fields["last_visited"], "http://www.google.com")
		assert.Equal(t, fields["myid"], "XXX71f343e3-2bfa-4ec6-86d4-79b91476acfc")
	})
	t.Run("update empty string", func(t *testing.T) {
		err := s.repo.Save(s.Ctx, user)
		require.NoError(t, err)
		user.Bio = ""
		err = s.repo.Save(s.Ctx, user)
		require.NoError(t, err)
		u, err := s.repo.Load(s.Ctx, user.ID)
		require.NoError(t, err)
		require.Empty(t, u.Bio)
	})
}

func createAndLoadUser(s *userBlackBoxTest, t *testing.T) *account.User {
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
	require.NoError(t, err, "Could not create user")

	createdUser, err := s.repo.Load(s.Ctx, user.ID)
	require.NoError(t, err, "Could not load user")
	require.Equal(t, user.Email, createdUser.Email)
	require.Equal(t, user.ID, createdUser.ID)
	require.Equal(t, user.ContextInformation["last_visited"], createdUser.ContextInformation["last_visited"])

	return createdUser
}
