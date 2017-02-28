package login

import (
	"crypto/md5"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	errs "github.com/pkg/errors"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	tokencontext "github.com/almighty/almighty-core/login/token_context"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/token"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

// Service defines the basic entrypoint required to perform a remote oauth login
type Service interface {
	Perform(ctx *app.AuthorizeLoginContext, authEndpoint string, tokenEndpoint string) error
}

// NewKeycloakOAuthProvider creates a new login.Service capable of using keycloak for authorization
func NewKeycloakOAuthProvider(config *oauth2.Config, identities account.IdentityRepository, users account.UserRepository, tokenManager token.Manager, db application.DB) *KeycloakOAuthProvider {
	return &KeycloakOAuthProvider{
		config:       config,
		Identities:   identities,
		Users:        users,
		TokenManager: tokenManager,
		db:           db,
	}
}

// KeycloakOAuthProvider represents a keyclaok IDP
type KeycloakOAuthProvider struct {
	config       *oauth2.Config
	Identities   account.IdentityRepository
	Users        account.UserRepository
	TokenManager token.Manager
	db           application.DB
}

// KeycloakOAuthService represents keycloak OAuth service interface
type KeycloakOAuthService interface {
	Perform(ctx *app.AuthorizeLoginContext, authEndpoint string, tokenEndpoint string) error
	CreateKeycloakUser(accessToken string, ctx context.Context) (*account.Identity, *account.User, error)
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

// Perform performs authenticatin
func (keycloak *KeycloakOAuthProvider) Perform(ctx *app.AuthorizeLoginContext, authEndpoint string, tokenEndpoint string) error {
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
			log.Error(ctx, map[string]interface{}{
				"state":   state,
				"referer": knownReferer,
			}, "state or known referer was empty")

			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized("State or known referer was empty"))
			return ctx.Unauthorized(jerrors)
		}

		keycloakToken, err := keycloak.config.Exchange(ctx, code)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"code": code,
				"err":  err,
			}, "keycloak exchange operation failed")
			return redirectWithError(ctx, knownReferer, err.Error())
		}

		_, _, err = keycloak.CreateKeycloakUser(keycloakToken.AccessToken, ctx)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"token": keycloakToken.AccessToken,
				"err":   err,
			}, "failed to create a user and KeyCloak identity using the access token")
			return redirectWithError(ctx, knownReferer, err.Error())

		}

		referelURL, err := url.Parse(knownReferer)
		if err != nil {
			return redirectWithError(ctx, knownReferer, err.Error())
		}

		err = encodeToken(referelURL, keycloakToken)
		if err != nil {
			return redirectWithError(ctx, knownReferer, err.Error())
		}
		ctx.ResponseData.Header().Set("Location", referelURL.String())
		return ctx.TemporaryRedirect()
	}

	// First time access, redirect to oauth provider

	// store referer id to state for redirect later
	log.Info(ctx, map[string]interface{}{
		"referer": referer,
	}, "Got Request from!")

	state = uuid.NewV4().String()

	mapLock.Lock()
	defer mapLock.Unlock()

	stateReferer[state] = referer
	keycloak.config.Endpoint.AuthURL = authEndpoint
	keycloak.config.Endpoint.TokenURL = tokenEndpoint
	keycloak.config.RedirectURL = rest.AbsoluteURL(ctx.RequestData, "/api/login/authorize")

	redirectURL := keycloak.config.AuthCodeURL(state, oauth2.AccessTypeOnline)

	ctx.ResponseData.Header().Set("Location", redirectURL)
	return ctx.TemporaryRedirect()
}

func encodeToken(referal *url.URL, outhToken *oauth2.Token) error {
	str := outhToken.Extra("expires_in")
	expiresIn, err := strconv.Atoi(fmt.Sprintf("%v", str))
	if err != nil {
		return errs.WithStack(errors.New("cant convert expires_in to integer " + err.Error()))
	}
	str = outhToken.Extra("refresh_expires_in")
	refreshExpiresIn, err := strconv.Atoi(fmt.Sprintf("%v", str))
	if err != nil {
		return errs.WithStack(errors.New("cant convert refresh_expires_in to integer " + err.Error()))
	}
	tokenData := &app.TokenData{
		AccessToken:      &outhToken.AccessToken,
		RefreshToken:     &outhToken.RefreshToken,
		TokenType:        &outhToken.TokenType,
		ExpiresIn:        &expiresIn,
		RefreshExpiresIn: &refreshExpiresIn,
	}
	b, err := json.Marshal(tokenData)
	if err != nil {
		return errs.WithStack(errors.New("cant marshal token data struct " + err.Error()))
	}

	parameters := url.Values{}
	parameters.Add("token", outhToken.AccessToken) // Temporary keep the old "token" param. We will drop this param as soon as UI adopt the new json param.
	parameters.Add("token_json", string(b))
	referal.RawQuery = parameters.Encode()

	return nil
}

// CreateKeycloakUser creates a user and a keyclaok identity
func (keycloak *KeycloakOAuthProvider) CreateKeycloakUser(accessToken string, ctx context.Context) (*account.Identity, *account.User, error) {
	var identity *account.Identity
	var user *account.User

	claims, err := parseToken(accessToken, keycloak.TokenManager.PublicKey())
	if err != nil || checkClaims(claims) != nil {
		log.Error(ctx, map[string]interface{}{
			"token": accessToken,
			"err":   err,
		}, "unable to parse the token")
		return nil, nil, errors.New("Error when parsing token " + err.Error())
	}

	keycloakIdentityID, _ := uuid.FromString(claims.Subject)
	identities, err := keycloak.Identities.Query(account.IdentityFilterByID(keycloakIdentityID), account.IdentityWithUser())
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"keycloakIdentityID": keycloakIdentityID,
			"err":                err,
		}, "unable to  query for an identity by ID")
		return nil, nil, errors.New("Error during querying for an identity by ID " + err.Error())
	}

	if len(identities) == 0 {
		// No Idenity found, create a new Identity and User
		user = new(account.User)
		fillUser(claims, user)
		err = application.Transactional(keycloak.db, func(appl application.Application) error {
			err := appl.Users().Create(ctx, user)
			if err != nil {
				return err
			}
			identity = &account.Identity{
				ID:           keycloakIdentityID,
				Username:     claims.Username,
				ProviderType: account.KeycloakIDP,
				UserID:       account.NullUUID{UUID: user.ID, Valid: true},
				User:         *user}
			err = appl.Identities().Create(ctx, identity)
			return err
		})
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"keyCloakIdentityID": keycloakIdentityID,
				"username":           claims.Username,
				"err":                err,
			}, "unable to create user/identity")
			return nil, nil, errors.New("Cant' create user/identity " + err.Error())
		}
	} else {
		user = &identities[0].User
		// let's update the existing user with the fullname, email and avatar from Keycloak,
		// in case the user changed them since the last time he/she logged in
		fillUser(claims, user)
		err = keycloak.Users.Save(ctx, user)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"userID": user.ID,
				"err":    err,
			}, "unable to update user")
			return nil, nil, errors.New("Cant' update user " + err.Error())
		}
	}
	return identity, user, nil
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
	return nil, errs.WithStack(errors.New("token is not valid"))
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
		return errors.New("subject claim not found in token")
	}
	_, err := uuid.FromString(claims.Subject)
	if err != nil {
		return errors.New("subject claim from token is not UUID " + err.Error())
	}
	if claims.Username == "" {
		return errors.New("username claim not found in token")
	}
	if claims.Email == "" {
		return errors.New("email claim not found in token")
	}
	return nil
}

func fillUser(claims *keycloakTokenClaims, user *account.User) error {
	user.FullName = claims.Name
	user.Email = claims.Email
	image, err := generateGravatarURL(claims.Email)
	if err != nil {
		log.Warn(nil, map[string]interface{}{
			"userFullName": user.FullName,
			"err":          err,
		}, "error when generating gravatar")
		return errors.New("Error when generating gravatar " + err.Error())
	}
	user.ImageURL = image
	return nil
}

// ContextIdentity returns the identity's ID found in given context
// Uses tokenManager.Locate to fetch the identity of currently logged in user
func ContextIdentity(ctx context.Context) (*uuid.UUID, error) {
	tm := tokencontext.ReadTokenManagerFromContext(ctx)
	if tm == nil {
		log.Error(ctx, map[string]interface{}{
			"token": tm,
		}, "missing token manager")

		return nil, errs.New("Missing token manager")
	}
	// As mentioned in token.go, we can now safely convert tm to a token.Manager
	manager := tm.(token.Manager)
	uuid, err := manager.Locate(ctx)
	if err != nil {
		// TODO : need a way to define user as Guest
		log.Error(ctx, map[string]interface{}{
			"uuid":         uuid,
			"tokenManager": manager,
			"err":          err,
		}, "identity belongs to a Guest User")

		return nil, errs.WithStack(err)
	}
	return &uuid, nil
}

// InjectTokenManager is a middleware responsible for setting up tokenManager in the context for every request.
func InjectTokenManager(tokenManager token.Manager) goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			ctxWithTM := tokencontext.ContextWithTokenManager(ctx, tokenManager)
			return h(ctxWithTM, rw, req)
		}
	}
}
