package login

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

const (
	// InvalidCodeError could occure when the OAuth Exchange with GitHub return no valid AccessToken
	InvalidCodeError string = "Invalid OAuth2.0 code"
	// PrimaryEmailNotFoundError could occure if no primary email was returned by GitHub
	PrimaryEmailNotFoundError string = "Primary email not found"
)

// Service defines the basic entrypoint required to perform a remote oauth login
type Service interface {
	Perform(ctx *app.AuthorizeLoginContext) error
}

// NewGitHubOAuth creates a new login.Service capable of using GitHub for authorization
func NewGitHubOAuth(config *oauth2.Config, identities account.IdentityRepository, users account.UserRepository, tokenManager token.Manager) Service {
	return &gitHubOAuth{
		config:       config,
		identities:   identities,
		users:        users,
		tokenManager: tokenManager,
	}
}

type gitHubOAuth struct {
	config       *oauth2.Config
	identities   account.IdentityRepository
	users        account.UserRepository
	tokenManager token.Manager
}

// TEMP: This will leak memory in the long run with many 'failed' redirect attemts
var stateReferer = map[string]string{}
var mapLock sync.RWMutex

func (gh *gitHubOAuth) Perform(ctx *app.AuthorizeLoginContext) error {
	state := ctx.Params.Get("state")
	code := ctx.Params.Get("code")
	referer := ctx.RequestData.Header.Get("Referer")

	if code != "" {
		// After redirect from oauth provider

		// validate known state
		var knownReferer string
		defer func() {
			delete(stateReferer, state)
		}()

		knownReferer = stateReferer[state]
		if state == "" || knownReferer == "" {
			return ctx.Unauthorized()
		}

		ghtoken, err := gh.config.Exchange(ctx, code)

		/*

			In case of invalid code, this is what we get in the ghtoken object

			&oauth2.Token{AccessToken:"", TokenType:"", RefreshToken:"", Expiry:time.Time{sec:0, nsec:0, loc:(*time.Location)(nil)}, raw:url.Values{"error":[]string{"bad_verification_code"}, "error_description":[]string{"The code passed is incorrect or expired."}, "error_uri":[]string{"https://developer.github.com/v3/oauth/#bad-verification-code"}}}

		*/

		if err != nil || ghtoken.AccessToken == "" {
			fmt.Println(err)
			ctx.ResponseData.Header().Set("Location", knownReferer+"?error="+InvalidCodeError)
			return ctx.TemporaryRedirect()
		}

		emails, err := gh.getUserEmails(ctx, ghtoken)
		fmt.Println(emails)

		primaryEmail := filterPrimaryEmail(emails)
		if primaryEmail == "" {
			fmt.Println("No primary email found?! ", emails)
			ctx.ResponseData.Header().Set("Location", knownReferer+"?error="+PrimaryEmailNotFoundError)
			return ctx.TemporaryRedirect()
		}
		users, err := gh.users.Query(account.UserByEmails([]string{primaryEmail}), account.UserWithIdentity())
		if err != nil {
			ctx.ResponseData.Header().Set("Location", knownReferer+"?error=Associated user not found "+err.Error())
			return ctx.TemporaryRedirect()
		}
		var identity account.Identity
		if len(users) == 0 {
			// No User found, create new User and Identity
			ghUser, err := gh.getUser(ctx, ghtoken)
			if err != nil {
				fmt.Println(err)
				return ctx.Unauthorized()
			}
			fmt.Println(ghUser)

			identity = createIdentity(*ghUser)
			gh.identities.Create(ctx, &identity)
			gh.users.Create(ctx, &account.User{Email: primaryEmail, Identity: identity})
		} else {
			identity = users[0].Identity
		}

		fmt.Println("Identity: ", identity)

		// register other emails in User table?

		// generate token
		almtoken, err := gh.tokenManager.Generate(identity)
		if err != nil {
			fmt.Println("Failed to generate token", err)
			return ctx.Unauthorized()
		}

		ctx.ResponseData.Header().Set("Location", knownReferer+"?token="+almtoken)
		return ctx.TemporaryRedirect()
	}

	// First time access, redirect to oauth provider

	// store referer id to state for redirect later
	fmt.Println("Got Request from: ", referer)
	state = uuid.NewV4().String()

	mapLock.Lock()
	defer mapLock.Unlock()

	stateReferer[state] = referer

	redirectURL := gh.config.AuthCodeURL(state, oauth2.AccessTypeOnline)
	ctx.ResponseData.Header().Set("Location", redirectURL)
	return ctx.TemporaryRedirect()
}

func (gh gitHubOAuth) getUserEmails(ctx context.Context, token *oauth2.Token) ([]ghEmail, error) {
	client := gh.config.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return nil, err
	}

	var emails []ghEmail
	json.NewDecoder(resp.Body).Decode(&emails)
	return emails, nil
}

func (gh gitHubOAuth) getUser(ctx context.Context, token *oauth2.Token) (*ghUser, error) {
	client := gh.config.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}

	var user ghUser
	json.NewDecoder(resp.Body).Decode(&user)
	return &user, nil
}

func createIdentity(ghUser ghUser) account.Identity {
	// Use login as name if 'name' is not set #391
	name := ghUser.Name
	if name == "" {
		name = ghUser.Login
	}
	return account.Identity{
		FullName: name,
		ImageURL: ghUser.AvatarURL,
	}
}

func filterPrimaryEmail(emails []ghEmail) string {
	for _, email := range emails {
		if email.Primary {
			return email.Email
		}
	}
	return ""
}

// ghEmail represents the needed response from api.github.com/user/emails
type ghEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// ghUser represents the needed response from api.github.com/user
type ghUser struct {
	Name      string `json:"name"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

type contextKey int

const (
	//currentUserInContextKey is a key that will be used to put and to get `currentUser` from goa.context
	currentUserInContextKey contextKey = iota + 1
)

// WithIdentity fills the context with input id string
func WithIdentity(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, currentUserInContextKey, id)
}

// ContextIdentity reads the input context then extracts and returns logged in user from that
func ContextIdentity(ctx context.Context) string {
	val := ctx.Value(currentUserInContextKey)
	if val != nil {
		return val.(string)
	}
	return ""
}

// ConfigureExtractUser is a configuration for middleware,
// Accepts PublicKey and PrivateKey to create a token manager
// Using token manager, it is responsible for extracting the token from context if present
// Update the context with uuid found in token
func ConfigureExtractUser(manager token.Manager) goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			// for now ignore the error becasue still test for logged in user is not done.
			uuid, err := manager.Locate(ctx)
			if err != nil {
				// TODO : need a way to define user as Guest
				fmt.Println("Geust User")
			}
			ctxWithUser := WithIdentity(ctx, uuid.String())
			return h(ctxWithUser, rw, req)
		}
	}
}
