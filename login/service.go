package login

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	errs "github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

const (
	// InvalidCodeError could occure when the OAuth Exchange with GitHub return no valid AccessToken
	InvalidCodeError string = "Invalid OAuth2.0 code"
	// PrimaryEmailNotFoundError could occur if no primary email was returned by GitHub
	PrimaryEmailNotFoundError string = "Primary email not found"
	AssociatedUserNotFound    string = "Associated user not found"
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
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized("State or known referer was empty"))
			return ctx.Unauthorized(jerrors)
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

		identity, err := gh.retrieveUserIdentity(ctx, ghtoken)
		if err != nil {
			switch err.(type) {
			case errs.IdentityError:
				identityErr := err.(errs.IdentityError)
				switch identityErr.Code {
				case noPrimaryEmail:
					ctx.ResponseData.Header().Set("Location", knownReferer+"?error="+identityErr.Message.(string))
					return ctx.TemporaryRedirect()
				case unauthorizedGitHubRequest:
					return ctx.Unauthorized(identityErr.Message.(*app.JSONAPIErrors))
				case remoteAPIError:
					ctx.ResponseData.Header().Set("Location", knownReferer+"?error="+identityErr.Message.(string))
					fmt.Println("Failed to retrieve user identity:", err.Error())
					return ctx.InternalServerError()
				}
			default:
				ctx.ResponseData.Header().Set("Location", knownReferer+"?error="+AssociatedUserNotFound+": "+err.Error())
			}
			return ctx.TemporaryRedirect()
		}
		// register other emails in User table?

		// generate token
		almtoken, err := gh.tokenManager.Generate(*identity)
		if err != nil {
			fmt.Println("Failed to generate token", err)
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
			return ctx.Unauthorized(jerrors)
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

const (
	noPrimaryEmail = iota
	unauthorizedGitHubRequest
	remoteAPIError
)

// retrieveUserIdentity retrieves the user identity on GitHub, then creates or updates the corresponding identity in the DB.
func (gh *gitHubOAuth) retrieveUserIdentity(ctx context.Context, ghtoken *oauth2.Token) (*account.Identity, error) {
	emails, err := gh.getUserEmails(ctx, ghtoken)
	if err != nil {
		return nil, errs.NewIdentityError(remoteAPIError, fmt.Sprintf("Failed to retrieve user emails: %v", err))
	}
	primaryEmail := filterPrimaryEmail(emails)
	if primaryEmail == "" {
		fmt.Println("No primary email found?! ", emails)
		return nil, errs.NewIdentityError(noPrimaryEmail, PrimaryEmailNotFoundError)
	}
	users, err := gh.users.Query(account.UserByEmails([]string{primaryEmail}), account.UserWithIdentity())
	if err != nil {
		return nil, errs.NewIdentityError(remoteAPIError, fmt.Sprintf("Failed to retrieve user by primary email: %v", err))
	}
	ghUser, err := gh.getUser(ctx, ghtoken)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
		return nil, errs.NewIdentityError(unauthorizedGitHubRequest, jerrors)
	}
	var identity account.Identity
	if len(users) == 0 {
		// No User found, create new User and Identity
		identity := new(account.Identity)
		fillIdentity(ghUser, identity)
		gh.identities.Create(ctx, identity)
		gh.users.Create(ctx, &account.User{Email: primaryEmail, Identity: *identity})
	} else {
		identity = users[0].Identity
		// let's update the current identity with the fullname and avatar from GitHub,
		// in case the user changed them since the last time he logged in here
		fillIdentity(ghUser, &identity)
		gh.identities.Save(ctx, &identity)
	}
	return &identity, nil
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

func fillIdentity(ghUser *ghUser, identity *account.Identity) {
	// Use login as name if 'name' is not set #391
	if ghUser.Name == "" {
		identity.FullName = ghUser.Login
	} else {
		identity.FullName = ghUser.Name
	}
	identity.ImageURL = ghUser.AvatarURL
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

// ContextIdentity returns the identity's ID found in given context
// Uses tokenManager.Locate to fetch the identity of currently logged in user
func ContextIdentity(ctx context.Context) (string, error) {
	tm := ReadTokenManagerFromContext(ctx)
	if tm == nil {
		return "", errors.New("Missing token manager")
	}
	uuid, err := tm.Locate(ctx)
	if err != nil {
		// TODO : need a way to define user as Guest
		fmt.Println("Geust User")
		return "", err
	}
	return uuid.String(), nil
}

type contextTMKey int

const (
	//contextTokenManagerKey is a key that will be used to put and to get `tokenManager` from goa.context
	contextTokenManagerKey contextTMKey = iota + 1
)

//ReadTokenManagerFromContext returns tokenManager from context.
// Must have been set by ContextWithTokenManager ONLY
func ReadTokenManagerFromContext(ctx context.Context) token.Manager {
	tm := ctx.Value(contextTokenManagerKey)
	if tm != nil {
		return tm.(token.Manager)
	}
	return nil
}

// ContextWithTokenManager injects tokenManager in the context for every incoming request
// Accepts Token.Manager in order to make sure that correct object is set in the context.
// Only other possible value is nil
func ContextWithTokenManager(ctx context.Context, tm token.Manager) context.Context {
	return context.WithValue(ctx, contextTokenManagerKey, tm)
}

// InjectTokenManager is a middleware responsible for setting up tokenManager in the context for every request.
func InjectTokenManager(tokenManager token.Manager) goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			ctxWithTM := ContextWithTokenManager(ctx, tokenManager)
			return h(ctxWithTM, rw, req)
		}
	}
}
