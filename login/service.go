package login

import (
	"crypto/md5"
	"crypto/rsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	errs "github.com/pkg/errors"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/token"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

const (
	// InvalidCodeError could occure when the OAuth Exchange with Keycloak returns no valid AccessToken
	InvalidCodeError string = "Invalid OAuth2.0 code"
)

// Service defines the basic entrypoint required to perform a remote oauth login
type Service interface {
	Perform(ctx *app.AuthorizeLoginContext) error
}

// NewKeycloakOAuthProvider creates a new login.Service capable of using keycloak for authorization
func NewKeycloakOAuthProvider(config *oauth2.Config, identities account.IdentityRepository, users account.UserRepository, tokenManager token.Manager) Service {
	return &keycloakOAuthProvider{
		config:       config,
		identities:   identities,
		users:        users,
		tokenManager: tokenManager,
	}
}

type keycloakOAuthProvider struct {
	config       *oauth2.Config
	identities   account.IdentityRepository
	users        account.UserRepository
	tokenManager token.Manager
}

// keycloakTokenClaims represents standard Keycloak token claims
type keycloakTokenClaims struct {
	Name       string `json:"name"`
	Username   string `json:"preferred_username"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Email      string `json:"email"`
	jwt.StandardClaims
}

// TEMP: This will leak memory in the long run with many 'failed' redirect attempts
var stateReferer = map[string]string{}
var mapLock sync.RWMutex

func (keycloak *keycloakOAuthProvider) Perform(ctx *app.AuthorizeLoginContext) error {
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

		keycloakToken, err := keycloak.config.Exchange(ctx, code)

		if err != nil || keycloakToken.AccessToken == "" {
			log.Println(err)
			return redirectWithError(ctx, knownReferer, InvalidCodeError)
		}

		claims, err := parseToken(keycloakToken.AccessToken, keycloak.tokenManager.PublicKey())
		if err != nil || checkClaims(claims) != nil {
			return redirectWithError(ctx, knownReferer, "Error when parsing token "+err.Error())
		}

		email := claims.Email
		users, err := keycloak.users.Query(account.UserByEmails([]string{email}), account.UserWithIdentity())
		if err != nil {
			return redirectWithError(ctx, knownReferer, "Error during querying for a user "+err.Error())
		}
		var identity *account.Identity

		if len(users) == 0 {
			// No User found, create a new User and Identity
			identity = new(account.Identity)
			fillIdentity(claims, identity)
			keycloak.identities.Create(ctx, identity)
			keycloak.users.Create(ctx, &account.User{Email: email, Identity: *identity})
		} else {
			identity = &users[0].Identity
			// let's update the current identity with the fullname and avatar from Keycloak,
			// in case the user changed them since the last time he/she logged in here
			fillIdentity(claims, identity)
			keycloak.identities.Save(ctx, identity)
		}

		ctx.ResponseData.Header().Set("Location", knownReferer+"?token="+keycloakToken.AccessToken)
		return ctx.TemporaryRedirect()
	}

	// First time access, redirect to oauth provider

	// store referer id to state for redirect later
	log.Println("Got Request from: ", referer)
	state = uuid.NewV4().String()

	mapLock.Lock()
	defer mapLock.Unlock()

	stateReferer[state] = referer

	keycloak.config.RedirectURL = rest.AbsoluteURL(ctx.RequestData, "/api/login/authorize")

	redirectURL := keycloak.config.AuthCodeURL(state, oauth2.AccessTypeOnline)

	ctx.ResponseData.Header().Set("Location", redirectURL)
	return ctx.TemporaryRedirect()
}

func redirectWithError(ctx *app.AuthorizeLoginContext, knownReferer string, errorString string) error {
	ctx.ResponseData.Header().Set("Location", knownReferer+"?error="+errorString)
	return ctx.TemporaryRedirect()
}

func parseToken(tokenString string, publicKey *rsa.PublicKey) (*keycloakTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &keycloakTokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims := token.Claims.(*keycloakTokenClaims)
	if token.Valid {
		return claims, nil
	}
	return nil, errs.WithStack(errors.New("Token is not valid"))
}

func generateGravatarURL(email string) (string, error) {
	if email == "" {
		return "", nil
	}
	grURL, err := url.Parse("https://www.gravatar.com/avatar/")
	if err != nil {
		return "", errs.WithStack(err)
	}
	hash := md5.New()
	hash.Write([]byte(email))
	grURL.Path += fmt.Sprintf("%v", hex.EncodeToString(hash.Sum(nil))) + ".jpg"

	// We can use our own default image if there is no gravatar available for this email
	// defaultImage := "someDefaultImageURL.jpg"
	// parameters := url.Values{}
	// parameters.Add("d", fmt.Sprintf("%v", defaultImage))
	// grURL.RawQuery = parameters.Encode()

	urlStr := grURL.String()
	return urlStr, nil
}

func checkClaims(claims *keycloakTokenClaims) error {
	if claims.Subject == "" {
		return errors.New("Subject claim not found in token")
	}
	_, err := uuid.FromString(claims.Subject)
	if err != nil {
		return errors.New("Subject claim from token is not UUID " + err.Error())
	}
	if claims.Username == "" {
		return errors.New("Username claim not found in token")
	}
	if claims.Email == "" {
		return errors.New("Email claim not found in token")
	}
	return nil
}

func fillIdentity(claims *keycloakTokenClaims, identity *account.Identity) error {
	identity.FullName = claims.Name
	id, _ := uuid.FromString(claims.Subject)
	identity.ID = id
	image, err := generateGravatarURL(claims.Email)
	if err != nil {
		return errors.New("Error when generating gravatar " + err.Error())
	}
	identity.ImageURL = image
	return nil
}

// ContextIdentity returns the identity's ID found in given context
// Uses tokenManager.Locate to fetch the identity of currently logged in user
func ContextIdentity(ctx context.Context) (string, error) {
	tm := ReadTokenManagerFromContext(ctx)
	if tm == nil {
		return "", errs.New("Missing token manager")
	}
	uuid, err := tm.Locate(ctx)
	if err != nil {
		// TODO : need a way to define user as Guest
		fmt.Println("Guest User")
		return "", errs.WithStack(err)
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
