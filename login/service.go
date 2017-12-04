package login

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login/tokencontext"
	"github.com/fabric8-services/fabric8-wit/token"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

// NewKeycloakOAuthProvider creates a new login.Service capable of using keycloak for authorization
func NewKeycloakOAuthProvider(identities account.IdentityRepository, users account.UserRepository, tokenManager token.Manager, db application.DB) *KeycloakOAuthProvider {
	return &KeycloakOAuthProvider{
		Identities:   identities,
		Users:        users,
		TokenManager: tokenManager,
		db:           db,
	}
}

// KeycloakOAuthProvider represents a keycloak IDP
type KeycloakOAuthProvider struct {
	Identities   account.IdentityRepository
	Users        account.UserRepository
	TokenManager token.Manager
	db           application.DB
}

// KeycloakOAuthService represents keycloak OAuth service interface
type KeycloakOAuthService interface {
	CreateOrUpdateKeycloakUser(accessToken string, ctx context.Context) (*account.Identity, *account.User, error)
}

// CreateOrUpdateKeycloakUser creates a user and a keycloak identity. If the user and identity already exist then update them.
func (keycloak *KeycloakOAuthProvider) CreateOrUpdateKeycloakUser(accessToken string, ctx context.Context) (*account.Identity, *account.User, error) {
	var identity *account.Identity
	var user *account.User

	claims, err := keycloak.TokenManager.ParseToken(ctx, accessToken)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"token": accessToken,
			"err":   err,
		}, "unable to parse the token")
		return nil, nil, errors.New("unable to parse the token " + err.Error())
	}

	if err := token.CheckClaims(claims); err != nil {
		log.Error(ctx, map[string]interface{}{
			"token": accessToken,
			"err":   err,
		}, "invalid keycloak token claims")
		return nil, nil, errors.New("invalid keycloak token claims " + err.Error())
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

	if len(identities) == 0 {
		// No Identity found, create a new Identity and User
		user = new(account.User)
		identity = &account.Identity{}
		_, err = fillUser(claims, user, identity)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"keycloak_identity_id": keycloakIdentityID,
				"err": err,
			}, "unable to create user/identity")
			return nil, nil, errors.New("failed to update user/identity from claims" + err.Error())
		}
		err = application.Transactional(keycloak.db, func(appl application.Application) error {
			err := appl.Users().Create(ctx, user)
			if err != nil {
				return err
			}

			identity.ID = keycloakIdentityID
			identity.ProviderType = account.KeycloakIDP
			identity.UserID = account.NullUUID{UUID: user.ID, Valid: true}
			identity.User = *user

			err = appl.Identities().Create(ctx, identity)
			return err
		})
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"keycloak_identity_id": keycloakIdentityID,
				"username":             claims.Username,
				"err":                  err,
			}, "unable to create user/identity")
			return nil, nil, errors.New("failed to create user/identity " + err.Error())
		}

	} else {
		identity = &identities[0]
		user = &identity.User
		if user.ID == uuid.Nil {
			log.Error(ctx, map[string]interface{}{
				"identity_id": keycloakIdentityID,
			}, "Found Keycloak identity is not linked to any User")
			return nil, nil, errors.New("found Keycloak identity is not linked to any User")
		}
		// let's update the existing user with the fullname, email and avatar from Keycloak,
		// in case the user changed them since the last time he/she logged in
		isChanged, err := fillUser(claims, user, identity)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"keycloak_identity_id": keycloakIdentityID,
				"err": err,
			}, "unable to create user/identity")
			return nil, nil, errors.New("failed to update user/identity from claims" + err.Error())
		} else if isChanged {
			err = application.Transactional(keycloak.db, func(appl application.Application) error {
				err = appl.Users().Save(ctx, user)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"user_id": user.ID,
						"err":     err,
					}, "unable to update user")
					return errors.New("failed to update user " + err.Error())
				}
				err = appl.Identities().Save(ctx, identity)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"user_id": identity.ID,
						"err":     err,
					}, "unable to update identity")
					return errors.New("failed to update identity " + err.Error())
				}
				return err
			})
			if err != nil {
				log.Error(ctx, map[string]interface{}{
					"keycloak_identity_id": keycloakIdentityID,
					"username":             claims.Username,
					"err":                  err,
				}, "unable to update user/identity")
				return nil, nil, errors.New("failed to update user/identity " + err.Error())
			}
		}
	}
	return identity, user, nil
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

func fillUser(claims *token.TokenClaims, user *account.User, identity *account.Identity) (bool, error) {
	isChanged := false
	if user.FullName != claims.Name || user.Email != claims.Email || user.Company != claims.Company || identity.Username != claims.Username || user.ImageURL == "" {
		isChanged = true
	} else {
		return isChanged, nil
	}
	user.FullName = claims.Name
	user.Email = claims.Email
	user.Company = claims.Company
	identity.Username = claims.Username
	if user.ImageURL == "" {
		image, err := generateGravatarURL(claims.Email)
		if err != nil {
			log.Warn(nil, map[string]interface{}{
				"user_full_name": user.FullName,
				"err":            err,
			}, "error when generating gravatar")
			// if there is an error, we will qualify the identity/user as unchanged.
			return false, errors.New("Error when generating gravatar " + err.Error())
		}
		user.ImageURL = image
	}
	return isChanged, nil
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
			"uuid":          uuid,
			"token_manager": manager,
			"err":           err,
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
