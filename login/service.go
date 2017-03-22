package login

import (
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	errs "github.com/pkg/errors"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/auth"
	er "github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	tokencontext "github.com/almighty/almighty-core/login/token_context"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/token"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

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
	Perform(ctx *app.AuthorizeLoginContext, authEndpoint string, tokenEndpoint string, brokerEndpoint string) error
	CreateOrUpdateKeycloakUser(accessToken string, ctx context.Context) (*account.Identity, *account.User, error)
	Link(ctx *app.LinkLoginContext, brokerEndpoint string, clientID string) error
	LinkSession(ctx *app.LinksessionLoginContext, brokerEndpoint string, clientID string) error
	LinkCallback(ctx *app.LinkcallbackLoginContext, brokerEndpoint string, clientID string) error
}

type linkInterface interface {
	context.Context
	jsonapi.InternalServerError
	TemporaryRedirect() error
}

// keycloakTokenClaims represents standard Keycloak token claims
type keycloakTokenClaims struct {
	Name          string `json:"name"`
	Username      string `json:"preferred_username"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Email         string `json:"email"`
	SessionState  string `json:"session_state"`
	ClientSession string `json:"client_session"`
	jwt.StandardClaims
}

var allProvidersToLink = []string{"github", "openshift-v3"}

// Perform performs authenticatin
func (keycloak *KeycloakOAuthProvider) Perform(ctx *app.AuthorizeLoginContext, authEndpoint string, tokenEndpoint string, brokerEndpoint string) error {
	state := ctx.Params.Get("state")
	code := ctx.Params.Get("code")
	referrer := ctx.RequestData.Header.Get("Referer")

	if code != "" {
		// After redirect from oauth provider

		// validate known state
		knownReferrer, err := keycloak.getReferrer(ctx, state)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"state": state,
				"err":   err,
			}, "uknown state")
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized("uknown state. " + err.Error()))
			return ctx.Unauthorized(jerrors)
		}

		keycloakToken, err := keycloak.config.Exchange(ctx, code)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"code": code,
				"err":  err,
			}, "keycloak exchange operation failed")
			return redirectWithError(ctx, knownReferrer, err.Error())
		}

		_, _, err = keycloak.CreateOrUpdateKeycloakUser(keycloakToken.AccessToken, ctx)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"token": keycloakToken.AccessToken,
				"err":   err,
			}, "failed to create a user and KeyCloak identity using the access token")
			return redirectWithError(ctx, knownReferrer, err.Error())
		}

		// redirect back to original referrel
		referrerURL, err := url.Parse(knownReferrer)
		if err != nil {
			return redirectWithError(ctx, knownReferrer, err.Error())
		}

		err = encodeToken(referrerURL, keycloakToken)
		if err != nil {
			return redirectWithError(ctx, knownReferrer, err.Error())
		}
		referrerStr := referrerURL.String()

		// Check if federated identities are not likned yet
		// TODO we probably won't want to check it for the existing users.
		// But we need it for now because old users still may not be linked.
		linked, err := keycloak.checkAllFederatedIdentities(ctx, keycloakToken.AccessToken, brokerEndpoint)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}
		// Return linked=true param if account has been linked to all IdPs or linked=false if not.
		if linked {
			referrerStr = referrerStr + "&linked=true"
			ctx.ResponseData.Header().Set("Location", referrerStr)
			return ctx.TemporaryRedirect()
		}

		// TODO
		// ---- Autolinking enabled regardless of the "link" param. Remove when UI adds support of account linking
		link := true
		// ----

		if !link && (ctx.Link == nil || !*ctx.Link) {
			referrerStr = referrerStr + "&linked=false"
			ctx.ResponseData.Header().Set("Location", referrerStr)
			return ctx.TemporaryRedirect()
		}

		referrerStr = referrerStr + "&linked=true"
		return keycloak.autoLinkProvidersDuringLogin(ctx, keycloakToken.AccessToken, referrerStr)
	}

	// First time access, redirect to oauth provider

	// store referrer in a state reference to redirect later
	log.Info(ctx, map[string]interface{}{
		"referrer": referrer,
	}, "Got Request from!")

	stateID := uuid.NewV4()
	err := keycloak.saveReferrer(ctx, stateID, referrer)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"state":    stateID,
			"referrer": referrer,
			"err":      err,
		}, "unable to save the state")
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized("Unable to save the state. " + err.Error()))
		return ctx.InternalServerError(jerrors)
	}

	keycloak.config.Endpoint.AuthURL = authEndpoint
	keycloak.config.Endpoint.TokenURL = tokenEndpoint
	keycloak.config.RedirectURL = rest.AbsoluteURL(ctx.RequestData, "/api/login/authorize")

	redirectURL := keycloak.config.AuthCodeURL(stateID.String(), oauth2.AccessTypeOnline)

	ctx.ResponseData.Header().Set("Location", redirectURL)
	return ctx.TemporaryRedirect()
}

func (keycloak *KeycloakOAuthProvider) autoLinkProvidersDuringLogin(ctx *app.AuthorizeLoginContext, token string, referrerURL string) error {
	// Link all available Identity Providers
	linkURL, err := url.Parse(rest.AbsoluteURL(ctx.RequestData, "/api/login/linksession"))
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	claims, err := parseToken(token, keycloak.TokenManager.PublicKey())
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to parse token")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	parameters := url.Values{}
	parameters.Add("redirect", referrerURL)
	parameters.Add("sessionState", fmt.Sprintf("%v", claims.SessionState))
	parameters.Add("clientSession", fmt.Sprintf("%v", claims.ClientSession))
	linkURL.RawQuery = parameters.Encode()
	ctx.ResponseData.Header().Set("Location", linkURL.String())
	return ctx.TemporaryRedirect()
}

// checkAllFederatedIdentities returns false if there is at least one federated identity not linked to the account
func (keycloak *KeycloakOAuthProvider) checkAllFederatedIdentities(ctx context.Context, token string, brokerEndpoint string) (bool, error) {
	for _, provider := range allProvidersToLink {
		linked, err := keycloak.checkFederatedIdentity(ctx, token, brokerEndpoint, provider)
		if err != nil {
			return false, err
		}
		if !linked {
			return false, nil
		}
	}
	return true, nil
}

// checkFederatedIdentity returns true if the account is already linked to the identity provider
func (keycloak *KeycloakOAuthProvider) checkFederatedIdentity(ctx context.Context, token string, brokerEndpoint string, provider string) (bool, error) {
	req, err := http.NewRequest("GET", brokerEndpoint+"/"+provider+"/token", nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "Unable to crete http request")
		return false, er.NewInternalError("unable to crete http request " + err.Error())
	}
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"provider": provider,
			"err":      err.Error(),
		}, "Unable to obtain a federated identity token")
		return false, er.NewInternalError("Unable to obtain a federated identity token " + err.Error())
	}
	return res.StatusCode == http.StatusOK, nil
}

// Link links identity provider(s) to the user's account using user's access token
func (keycloak *KeycloakOAuthProvider) Link(ctx *app.LinkLoginContext, brokerEndpoint string, clientID string) error {
	token := goajwt.ContextJWT(ctx)
	claims := token.Claims.(jwt.MapClaims)
	sessionState := claims["session_state"]
	clientSession := claims["client_session"]
	if sessionState == nil || clientSession == nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal("Session state or client session are missing in token"))
	}
	ss := sessionState.(*string)
	cs := clientSession.(*string)
	return keycloak.linkAccountToProviders(ctx, ctx.RequestData, ctx.ResponseData, ctx.Redirect, ctx.Provider, *ss, *cs, brokerEndpoint, clientID)
}

// LinkSession links identity provider(s) to the user's account using session state
func (keycloak *KeycloakOAuthProvider) LinkSession(ctx *app.LinksessionLoginContext, brokerEndpoint string, clientID string) error {
	if ctx.SessionState == nil || ctx.ClientSession == nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest("Authorization header or session state and client session params are required"))
	}
	return keycloak.linkAccountToProviders(ctx, ctx.RequestData, ctx.ResponseData, ctx.Redirect, ctx.Provider, *ctx.SessionState, *ctx.ClientSession, brokerEndpoint, clientID)
}

func (keycloak *KeycloakOAuthProvider) linkAccountToProviders(ctx linkInterface, req *goa.RequestData, res *goa.ResponseData, redirect *string, provider *string, sessionState string, clientSession, brokerEndpoint string, clientID string) error {
	referrer := req.Header.Get("Referer")

	rdr := redirect
	if rdr == nil {
		rdr = &referrer
	}

	state := uuid.NewV4()
	keycloak.saveReferrer(ctx, state, *rdr)

	if provider != nil {
		return keycloak.linkProvider(ctx, req, res, state.String(), sessionState, clientSession, *provider, nil, brokerEndpoint, clientID)
	}

	return keycloak.linkProvider(ctx, req, res, state.String(), sessionState, clientSession, allProvidersToLink[0], &allProvidersToLink[1], brokerEndpoint, clientID)
}

// LinkCallback redirects to original referrer when Identity Provider account are linked to the user account
func (keycloak *KeycloakOAuthProvider) LinkCallback(ctx *app.LinkcallbackLoginContext, brokerEndpoint string, clientID string) error {
	state := ctx.State
	errorMessage := ctx.Params.Get("error")
	if state == nil {
		jsonapi.JSONErrorResponse(ctx, goa.ErrInternal("State is empty. "+errorMessage))
	}
	if errorMessage != "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(errorMessage))
	}

	next := ctx.Next
	if next != nil {
		// Link the next provider
		sessionState := ctx.SessionState
		clientSession := ctx.ClientSession
		if sessionState == nil || clientSession == nil {
			log.Error(ctx, map[string]interface{}{
				"state": state,
			}, "Session state or client session state is empty")
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest("Session state or client session state is empty"))
			return ctx.Unauthorized(jerrors)
		}
		providerURL, err := getProviderURL(ctx.RequestData, *state, *sessionState, *clientSession, *next, nextProvider(*next), brokerEndpoint, clientID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}
		ctx.ResponseData.Header().Set("Location", providerURL)
		return ctx.TemporaryRedirect()
	}

	// No more providers to link. Redirect back to the original referrer
	originalReferrer, err := keycloak.getReferrer(ctx, *state)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"state": state,
			"err":   err,
		}, "uknown state")
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized("uknown state. " + err.Error()))
		return ctx.Unauthorized(jerrors)
	}

	ctx.ResponseData.Header().Set("Location", originalReferrer)
	return ctx.TemporaryRedirect()
}

func nextProvider(currentProvider string) *string {
	for i, provider := range allProvidersToLink {
		if provider == currentProvider {
			if i+1 < len(allProvidersToLink) {
				return &allProvidersToLink[i+1]
			}
			return nil
		}
	}
	return nil
}

func (keycloak *KeycloakOAuthProvider) linkProvider(ctx linkInterface, req *goa.RequestData, res *goa.ResponseData, state string, sessionState string, clientSession string, provider string, nextProvider *string, brokerEndpoint string, clientID string) error {
	providerURL, err := getProviderURL(req, state, sessionState, clientSession, provider, nextProvider, brokerEndpoint, clientID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	res.Header().Set("Location", providerURL)
	return ctx.TemporaryRedirect()
}

func (keycloak *KeycloakOAuthProvider) saveReferrer(ctx context.Context, state uuid.UUID, referrer string) error {
	// TODO The state reference table will be collecting dead states left from some failed login attempts.
	// We need to clean up the old states from time to time.
	ref := auth.OauthStateReference{
		ID:       state,
		Referrer: referrer,
	}
	err := application.Transactional(keycloak.db, func(appl application.Application) error {
		_, err := appl.OauthStates().Create(ctx, &ref)
		return err
	})
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"state":    state,
			"referrer": referrer,
			"err":      err,
		}, "unable to create oauth state reference")
		return errors.New("Unable to create oauth state reference " + err.Error())
	}
	return nil
}

func (keycloak *KeycloakOAuthProvider) getReferrer(ctx context.Context, state string) (string, error) {
	var referrer string
	stateID, err := uuid.FromString(state)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"state": state,
			"err":   err,
		}, "unable to convert oauth state to uuid")
		return "", errors.New("Unable to convert oauth state to uuid. " + err.Error())
	}
	err = application.Transactional(keycloak.db, func(appl application.Application) error {
		ref, err := appl.OauthStates().Load(ctx, stateID)
		if err != nil {
			return err
		}
		referrer = ref.Referrer
		err = appl.OauthStates().Delete(ctx, stateID)
		return err
	})
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"state": state,
			"err":   err,
		}, "unable to delete oauth state reference")
		return "", errors.New("Unable to delete oauth state reference " + err.Error())
	}
	return referrer, nil
}

func getProviderURL(req *goa.RequestData, state string, sessionState string, clientSession string, provider string, nextProvider *string, brokerEndpoint string, clientID string) (string, error) {
	var nextParam string
	if nextProvider != nil {
		nextParam = "&next=" + *nextProvider
	}
	callbackURL := rest.AbsoluteURL(req, "/api/login/linkcallback?provider="+provider+nextParam+"&sessionState="+sessionState+"&clientSession="+clientSession+"&state="+state)

	nonce := uuid.NewV4().String()

	s := nonce + sessionState + clientSession + provider
	h := sha256.New()
	h.Write([]byte(s))
	hash := base64.StdEncoding.EncodeToString(h.Sum(nil))

	linkingURL, err := url.Parse(brokerEndpoint + "/" + provider + "/link")
	if err != nil {
		return "", err
	}

	parameters := url.Values{}
	parameters.Add("client_id", clientID)
	parameters.Add("redirect_uri", callbackURL)
	parameters.Add("nonce", nonce)
	parameters.Add("hash", hash)
	linkingURL.RawQuery = parameters.Encode()

	return linkingURL.String(), nil
}

func encodeToken(referrer *url.URL, outhToken *oauth2.Token) error {
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
	referrer.RawQuery = parameters.Encode()

	return nil
}

// CreateOrUpdateKeycloakUser creates a user and a keyclaok identity. If the user and identity already exist then update them.
func (keycloak *KeycloakOAuthProvider) CreateOrUpdateKeycloakUser(accessToken string, ctx context.Context) (*account.Identity, *account.User, error) {
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
			"keycloak_identity_id": keycloakIdentityID,
			"err": err,
		}, "unable to  query for an identity by ID")
		return nil, nil, errors.New("Error during querying for an identity by ID " + err.Error())
	}

	// TODO REMOVE THIS WORKAROUND
	// ----------------- BEGIN WORKAROUND -----------------
	if len(identities) == 0 {
		// This is not what actaully should happen.
		// This is a workaround for Keyclaok and DB unsynchronization.
		// The old identity will be removed. The new one with proper ID will be created.
		// All links to the old identities (in Work Items for example) will still point to the deleted identity.
		// No Idenity with the keycloak user ID is found, try to search by the username
		identities, err = keycloak.Identities.Query(account.IdentityFilterByUsername(claims.Username), account.IdentityWithUser())
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"keycloakIdentityUsername": claims.Username,
				"err": err,
			}, "unable to  query for an identity by username")
			return nil, nil, errors.New("Error during querying for an identity by username " + err.Error())
		}
		if len(identities) != 0 {
			idn := identities[0]
			if idn.ProviderType == account.KeycloakIDP {
				log.Warn(ctx, map[string]interface{}{
					"keycloak_identity_id":       keycloakIdentityID,
					"core_identity_id":           idn.ID,
					"keycloak_identity_username": claims.Username,
				}, "the identity ID fetched from Keycloak and the identity ID from the core DB for the same username don't match. The identity will be re-created.")

				err = application.Transactional(keycloak.db, func(appl application.Application) error {
					user = &idn.User
					identity = &account.Identity{
						ID:           keycloakIdentityID,
						Username:     claims.Username,
						ProviderType: account.KeycloakIDP,
						UserID:       account.NullUUID{UUID: user.ID, Valid: true},
						User:         *user}
					err := appl.Identities().Delete(ctx, idn.ID)
					if err != nil {
						return err
					}
					err = appl.Identities().Create(ctx, identity)
					return err
				})
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"keycloak_identity_id":       keycloakIdentityID,
						"core_identity_id":           idn.ID,
						"keycloak_identity_username": claims.Username,
						"err": err,
					}, "unable to update identity")
					return nil, nil, errors.New("Cant' create user/identity " + err.Error())
				}
				identities[0] = identity
			} else {
				// The found identity is not a KC identity, ignore it
				// TODO we also should make sure that the email used by this Identity is not the same.
				// It may happen if the found identity was imported from a remote issue tracker and has the same email
				identities = []*account.Identity{}
			}
		}
	}
	// ----------------- END WORKAROUND -----------------

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
				"keycloak_identity_id": keycloakIdentityID,
				"username":             claims.Username,
				"err":                  err,
			}, "unable to create user/identity")
			return nil, nil, errors.New("Cant' create user/identity " + err.Error())
		}
	} else {
		user = &identities[0].User
		if user.ID == uuid.Nil {
			log.Error(ctx, map[string]interface{}{
				"identity_id": keycloakIdentityID,
			}, "Found Keycloak identity is not linked to any User")
			return nil, nil, errors.New("found Keycloak identity is not linked to any User")
		}
		// let's update the existing user with the fullname, email and avatar from Keycloak,
		// in case the user changed them since the last time he/she logged in
		fillUser(claims, user)
		err = keycloak.Users.Save(ctx, user)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"user_id": user.ID,
				"err":     err,
			}, "unable to update user")
			return nil, nil, errors.New("Cant' update user " + err.Error())
		}
	}
	return identity, user, nil
}

func redirectWithError(ctx *app.AuthorizeLoginContext, knownReferrer string, errorString string) error {
	ctx.ResponseData.Header().Set("Location", knownReferrer+"?error="+errorString)
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
