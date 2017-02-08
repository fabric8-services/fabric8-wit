package login_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/configuration"
	. "github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var db *gorm.DB
var loginService Service

func TestMain(m *testing.M) {
	if _, c := os.LookupEnv(resource.Database); c != false {
		var err error
		if err = configuration.Setup(""); err != nil {
			panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
		}

		db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())
		if err != nil {
			panic("Failed to connect database: " + err.Error())
		}
		defer db.Close()

		// Migrate the schema
		err = migration.Migrate(db.DB())
		if err != nil {
			panic(err.Error())
		}

	}

	oauth := &oauth2.Config{
		ClientID:     configuration.GetKeycloakClientID(),
		ClientSecret: configuration.GetKeycloakSecret(),
		Scopes:       []string{"user:email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://sso.demo.almighty.io/auth/realms/demo/protocol/openid-connect/auth",
			TokenURL: "http://sso.demo.almighty.io/auth/realms/demo/protocol/openid-connect/token",
		},
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
	loginService = NewKeycloakOAuthProvider(oauth, identityRepository, userRepository, tokenManager)

	os.Exit(m.Run())
}

func TestKeycloakAuthorizationRedirect(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	rw := httptest.NewRecorder()
	u := &url.URL{
		Path: fmt.Sprintf("/api/login/authorize"),
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}

	// The user clicks login while on ALM UI.
	// Therefore the referer would be an ALM URL.
	refererUrl := "https://alm-url.example.org/path"
	req.Header.Add("referer", refererUrl)

	prms := url.Values{}
	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	authorizeCtx, err := app.NewAuthorizeLoginContext(goaCtx, goa.New("LoginService"))
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = loginService.Perform(authorizeCtx)

	assert.Equal(t, 307, rw.Code)
	assert.Contains(t, rw.Header().Get("Location"), configuration.GetKeycloakEndpointAuth())
}

func TestValidOAuthAuthorizationCode(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// Current the OAuth code is generated as part of a UI workflow.
	// Yet to figure out how to mock.
	t.Skip("Authorization Code not available")

}

func TestValidState(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// We do not have a test for a valid
	// authorization code because it needs a
	// user UI workflow. Furthermore, the code can be used
	// only once. https://tools.ietf.org/html/rfc6749#section-4.1.2
	t.Skip("Authorization Code not available")
}

func TestInvalidState(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	// Setup request context
	rw := httptest.NewRecorder()
	u := &url.URL{
		Path: fmt.Sprintf("/api/login/authorize"),
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}

	// The OAuth 'state' is sent as a query parameter by calling /api/login/authorize?code=_SOME_CODE_&state=_SOME_STATE_
	// The request originates from Keycloak after a valid authorization by the end user.
	// This is not where the redirection should happen on failure.
	refererKeyclaokUrl := "https://keycloak-url.example.org/path-of-login"
	req.Header.Add("referer", refererKeyclaokUrl)

	prms := url.Values{
		"state": {},
		"code":  {"doesnt_matter_what_is_here"},
	}
	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	authorizeCtx, err := app.NewAuthorizeLoginContext(goaCtx, goa.New("LoginService"))
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = loginService.Perform(authorizeCtx)
	assert.Equal(t, 401, rw.Code)
}

func TestInvalidOAuthAuthorizationCode(t *testing.T) {

	// When a valid referrer talks to our system and provides
	// an invalid OAuth2.0 code, the access token exchange
	// fails. In such a scenario, there is response redirection
	// to the valid referer, ie, the URL where the request originated from.
	// Currently, this should be something like https://demo.almighty.org/somepage/

	resource.Require(t, resource.UnitTest)

	// Setup request context
	rw := httptest.NewRecorder()
	u := &url.URL{
		Path: fmt.Sprintf("/api/login/authorize"),
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}

	// The user clicks login while on ALM UI.
	// Therefore the referer would be an ALM URL.
	refererUrl := "https://alm-url.example.org/path"
	req.Header.Add("referer", refererUrl)

	prms := url.Values{}
	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	authorizeCtx, err := app.NewAuthorizeLoginContext(goaCtx, goa.New("LoginService"))
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = loginService.Perform(authorizeCtx)

	assert.Equal(t, 307, rw.Code) // redirect to github login page.

	locationString := rw.HeaderMap["Location"][0]
	locationUrl, err := url.Parse(locationString)
	if err != nil {
		t.Fatal("Redirect URL is in a wrong format ", err)
	}

	t.Log(locationString)
	allQueryParameters := locationUrl.Query()

	// Avoiding panics.
	assert.NotNil(t, allQueryParameters)
	assert.NotNil(t, allQueryParameters["state"][0])

	returnedState := allQueryParameters["state"][0]

	prms = url.Values{
		"state": {returnedState},
		"code":  {"INVALID_OAUTH2.0_CODE"},
	}
	ctx = context.Background()
	rw = httptest.NewRecorder()

	req, err = http.NewRequest("GET", u.String(), nil)

	// The OAuth code is sent as a query parameter by calling /api/login/authorize?code=_SOME_CODE_&state=_SOME_STATE_
	// The request originates from Keycloak after a valid authorization by the end user.
	// This is not where the redirection should happen on failure.
	refererKeycloakUrl := "https://keycloak-url.example.org/path-of-login"
	req.Header.Add("referer", refererKeycloakUrl)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}

	goaCtx = goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	authorizeCtx, err = app.NewAuthorizeLoginContext(goaCtx, goa.New("LoginService"))

	err = loginService.Perform(authorizeCtx)

	locationString = rw.HeaderMap["Location"][0]
	locationUrl, err = url.Parse(locationString)
	if err != nil {
		t.Fatal("Redirect URL is in a wrong format ", err)
	}

	t.Log(locationString)
	allQueryParameters = locationUrl.Query()
	assert.Equal(t, 307, rw.Code) // redirect to ALM page where login was clicked.
	// Avoiding panics.
	assert.NotNil(t, allQueryParameters)
	assert.NotNil(t, allQueryParameters["error"])
	assert.Equal(t, allQueryParameters["error"][0], InvalidCodeError)

	returnedErrorReason := allQueryParameters["error"][0]
	assert.NotEmpty(t, returnedErrorReason)
	assert.NotContains(t, locationString, refererKeycloakUrl)
	assert.Contains(t, locationString, refererUrl)
}
