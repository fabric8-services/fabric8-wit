package login

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/token"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/github"
)

type serviceWhiteBoxTest struct {
	gormsupport.DBTestSuite
	loginService       *gitHubOAuth
	identityRepository account.IdentityRepository
	userRepository     account.UserRepository
}

func TestRunServiceWhiteBoxTest(t *testing.T) {
	suite.Run(t, &serviceWhiteBoxTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

func (s *serviceWhiteBoxTest) SetupTest() {
	var err error
	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	oauth := &oauth2.Config{
		ClientID:     configuration.GetGithubClientID(),
		ClientSecret: configuration.GetGithubSecret(),
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
	publicKey, err := token.ParsePublicKey([]byte(configuration.GetTokenPublicKey()))
	if err != nil {
		panic(err)
	}
	privateKey, err := token.ParsePrivateKey([]byte(configuration.GetTokenPrivateKey()))
	if err != nil {
		panic(err)
	}
	tokenManager := token.NewManager(publicKey, privateKey)
	s.userRepository = account.NewUserRepository(s.DB)
	s.identityRepository = account.NewIdentityRepository(s.DB)
	s.loginService = &gitHubOAuth{
		config:       oauth,
		identities:   s.identityRepository,
		users:        s.userRepository,
		tokenManager: tokenManager,
	}
	s.DB.LogMode(true)

}

func (s *serviceWhiteBoxTest) TestValidOAuthAccessToken() {
	resource.Require(s.T(), resource.UnitTest)
	accessToken := &oauth2.Token{
		AccessToken: configuration.GetGithubAuthToken(),
		TokenType:   "Bearer",
	}
	emails, err := s.loginService.getUserEmails(context.Background(), accessToken)
	var trials int // Number of tries to reach GitHub
	if err != nil {
		for trials = 0; trials < 10; trials++ {
			emails, err = s.loginService.getUserEmails(context.Background(), accessToken)
			if err == nil {
				assert.NotEmpty(s.T(), emails)
				break
			}
			time.Sleep(5 * time.Second) // Pause before the next retry
		}
		if trials == 10 {
			s.T().Error("Test failed, Maximum Retry limit reached", err) // Test failed after trial for 10 times
		}
	} else {
		assert.NotEmpty(s.T(), emails)
	}
}

func (s *serviceWhiteBoxTest) TestInvalidOAuthAccessToken() {
	resource.Require(s.T(), resource.UnitTest)
	invalidAccessToken := "7423742yuuiy-INVALID-73842342389h"
	accessToken := &oauth2.Token{
		AccessToken: invalidAccessToken,
		TokenType:   "Bearer",
	}
	emails, err := s.loginService.getUserEmails(context.Background(), accessToken)
	assert.Nil(s.T(), err)
	assert.Empty(s.T(), emails)
}

func (s *serviceWhiteBoxTest) TestGetUserEmails() {
	resource.Require(s.T(), resource.UnitTest)
	accessToken := &oauth2.Token{
		AccessToken: configuration.GetGithubAuthToken(),
		TokenType:   "Bearer",
	}
	githubEmails, err := s.loginService.getUserEmails(context.Background(), accessToken)
	s.T().Log(githubEmails)
	require.Nil(s.T(), err)
	assert.NotNil(s.T(), githubEmails)
	assert.NotEmpty(s.T(), githubEmails)
}

func (s *serviceWhiteBoxTest) TestGetUser() {
	resource.Require(s.T(), resource.UnitTest)
	accessToken := &oauth2.Token{
		AccessToken: configuration.GetGithubAuthToken(),
		TokenType:   "Bearer",
	}
	githubUser, err := s.loginService.getUser(context.Background(), accessToken)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), githubUser)
	s.T().Log(githubUser)
}

func (s *serviceWhiteBoxTest) TestFilterPrimaryEmail() {
	resource.Require(s.T(), resource.UnitTest)
	s.T().Skip("Not implemented")
}

func (s *serviceWhiteBoxTest) TestCreateUser() {
	resource.Require(s.T(), resource.Database)
	defer gormsupport.DeleteCreatedEntities(s.DB)()
	s.removeUserIfExists("sbose78@gmail.com")

	accessToken := &oauth2.Token{
		AccessToken: configuration.GetGithubAuthToken(),
		TokenType:   "Bearer",
	}
	identity, err := s.loginService.retrieveUserIdentity(context.Background(), accessToken)
	require.Nil(s.T(), err)
	assert.NotNil(s.T(), identity.FullName)
	assert.Regexp(s.T(), "https://avatars.githubusercontent.com/u/...", identity.ImageURL)
	storedIdentity, _ := s.identityRepository.Load(context.Background(), identity.ID)
	require.NotNil(s.T(), storedIdentity)
	assert.NotNil(s.T(), storedIdentity.FullName)
	assert.NotNil(s.T(), storedIdentity.ImageURL)
}

func (s *serviceWhiteBoxTest) TestUpdateUser() {
	resource.Require(s.T(), resource.Database)
	defer gormsupport.DeleteCreatedEntities(s.DB)()
	// preload db with an identity
	s.removeUserIfExists("sbose78@gmail.com")
	existingUser := account.User{Email: "sbose78@gmail.com"}
	existingIdentity := account.Identity{Emails: []account.User{existingUser}}
	s.identityRepository.Create(context.Background(), &existingIdentity)
	// verify that the identify exists, but without any fullname and avatar URL
	storedIdentity, _ := s.identityRepository.Load(context.Background(), existingIdentity.ID)
	require.NotNil(s.T(), storedIdentity)
	require.Equal(s.T(), "", storedIdentity.FullName)
	require.Equal(s.T(), "", storedIdentity.ImageURL)
	accessToken := &oauth2.Token{
		AccessToken: configuration.GetGithubAuthToken(),
		TokenType:   "Bearer",
	}
	identity, err := s.loginService.retrieveUserIdentity(context.Background(), accessToken)
	require.Nil(s.T(), err)
	assert.Equal(s.T(), identity.ID, storedIdentity.ID)
	assert.NotNil(s.T(), identity.FullName)
	assert.Regexp(s.T(), "https://avatars.githubusercontent.com/u/...", identity.ImageURL)
	storedIdentity, _ = s.identityRepository.Load(context.Background(), identity.ID)
	require.NotNil(s.T(), storedIdentity)
	assert.NotNil(s.T(), storedIdentity.FullName)
	assert.NotNil(s.T(), storedIdentity.ImageURL)

}

func (s *serviceWhiteBoxTest) removeUserIfExists(email string) {
	// remove identity if it already exists
	storedUsers, _ := s.userRepository.Query(account.UserWithIdentity(), func(db *gorm.DB) *gorm.DB {
		return db.Where("email = ?", email)
	})
	if len(storedUsers) > 0 {
		for i, _ := range storedUsers {
			log.Println("Removing identify " + storedUsers[i].Identity.FullName + " (" + storedUsers[i].Identity.ID.String() + ")")
			s.DB.Unscoped().Delete(storedUsers[i])
			s.DB.Unscoped().Delete(storedUsers[i].Identity)
		}
	}
}
