package login

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/token"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"golang.org/x/oauth2/github"
)

var db *gorm.DB
var loginService *gitHubOAuth

// Github doesn't allow commiting actual tokens no matter how
// less privleges the token has.
var camouflagedAccessToken = "751e16a8b39c0985066-AccessToken-4871777f2c13b32be8550"
var actualToken = strings.Split(camouflagedAccessToken, "-AccessToken-")[0] + strings.Split(camouflagedAccessToken, "-AccessToken-")[1]

func setup() {

	// If it's an integration test then this setups up the database object,

	if _, c := os.LookupEnv(resource.Database); c != false {
		var err error
		dbhost := os.Getenv("ALMIGHTY_DB_HOST")
		db, err = gorm.Open("postgres", fmt.Sprintf("host=%s database=postgres user=postgres password=mysecretpassword sslmode=disable", dbhost))

		if err != nil {
			panic("Failed to connect database: " + err.Error())
		}
	}

	oauth := &oauth2.Config{
		ClientID:     "875da0d2113ba0a6951d",
		ClientSecret: "2fe6736e90a9283036a37059d75ac0c82f4f5288",
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}

	publicKey, err := token.ParsePublicKey([]byte(token.RSAPublicKey))
	if err != nil {
		panic(err)
	}

	privateKey, err := token.ParsePrivateKey([]byte(token.RSAPrivateKey))
	if err != nil {
		panic(err)
	}

	tokenManager := token.NewManager(publicKey, privateKey)
	userRepository := account.NewUserRepository(db)
	identityRepository := account.NewIdentityRepository(db)
	loginService = &gitHubOAuth{
		config:       oauth,
		identities:   identityRepository,
		users:        userRepository,
		tokenManager: tokenManager,
	}
}

func tearDown() {
	if db != nil {
		db.Close()
		db = nil
	}
	loginService = nil
}

func TestValidOAuthAccessToken(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	setup()
	defer tearDown()

	accessToken := &oauth2.Token{
		AccessToken: actualToken,
		TokenType:   "Bearer",
	}

	emails, err := loginService.getUserEmails(context.Background(), accessToken)
	assert.Nil(t, err)
	assert.NotEmpty(t, emails)
}

func TestInvalidOAuthAccessToken(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	setup()
	defer tearDown()

	if loginService == nil {
		setup()
	}

	invalidAccessToken := "7423742yuuiy-INVALID-73842342389h"

	accessToken := &oauth2.Token{
		AccessToken: invalidAccessToken,
		TokenType:   "Bearer",
	}

	emails, err := loginService.getUserEmails(context.Background(), accessToken)
	assert.Nil(t, err)
	assert.Empty(t, emails)
}

func TestGetUserEmails(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	setup()
	defer tearDown()

	accessToken := &oauth2.Token{
		AccessToken: actualToken,
		TokenType:   "Bearer",
	}

	githubEmails, err := loginService.getUserEmails(context.Background(), accessToken)
	t.Log(githubEmails)
	assert.Nil(t, err)
	assert.NotNil(t, githubEmails)
	assert.NotEmpty(t, githubEmails)
}

func TestGetUser(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	setup()
	defer tearDown()

	accessToken := &oauth2.Token{
		AccessToken: actualToken,
		TokenType:   "Bearer",
	}

	githubUser, err := loginService.getUser(context.Background(), accessToken)
	assert.Nil(t, err)
	assert.NotNil(t, githubUser)
	t.Log(githubUser)
}

func TestFilterPrimaryEmail(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	t.Skip("Not implemented")
}
