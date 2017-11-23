package token

import (
	"context"
	"crypto/rsa"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	authclient "github.com/fabric8-services/fabric8-auth/token"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login/tokencontext"
	"github.com/satori/go.uuid"

	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/pkg/errors"
)

// tokenManagerConfiguration represents configuration needed to construct a token manager
type tokenManagerConfiguration interface {
	GetAuthServiceURL() string
	GetKeycloakDevModeURL() string
}

// TokenClaims represents access token claims
type TokenClaims struct {
	Name          string                `json:"name"`
	Username      string                `json:"preferred_username"`
	GivenName     string                `json:"given_name"`
	FamilyName    string                `json:"family_name"`
	Email         string                `json:"email"`
	Company       string                `json:"company"`
	SessionState  string                `json:"session_state"`
	Authorization *AuthorizationPayload `json:"authorization"`
	jwt.StandardClaims
}

// AuthorizationPayload represents an authz payload in the rpt token
type AuthorizationPayload struct {
	Permissions []Permissions `json:"permissions"`
}

// Permissions represents a "permissions" in the AuthorizationPayload
type Permissions struct {
	ResourceSetName *string `json:"resource_set_name"`
	ResourceSetID   *string `json:"resource_set_id"`
}

type PublicKey struct {
	KeyID string
	Key   *rsa.PublicKey
}

// Manager generate and find auth token information
type Manager interface {
	Locate(ctx context.Context) (uuid.UUID, error)
	ParseToken(ctx context.Context, tokenString string) (*TokenClaims, error)
	PublicKey(kid string) *rsa.PublicKey
	PublicKeys() []*rsa.PublicKey
	IsServiceAccount(ctx context.Context, serviceName string) bool
}

type tokenManager struct {
	publicKeysMap map[string]*rsa.PublicKey
	publicKeys    []*PublicKey
}

// NewManager returns a new token Manager for handling tokens
func NewManager(config tokenManagerConfiguration) (Manager, error) {
	// Load public keys from Auth service and add them to the manager
	tm := &tokenManager{
		publicKeysMap: map[string]*rsa.PublicKey{},
	}

	keysEndpoint := fmt.Sprintf("%s%s", config.GetAuthServiceURL(), authservice.KeysTokenPath())
	remoteKeys, err := authclient.FetchKeys(keysEndpoint)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":      err,
			"keys_url": keysEndpoint,
		}, "unable to load public keys from remote service")
		return nil, errors.New("unable to load public keys from remote service")
	}
	for _, remoteKey := range remoteKeys {
		tm.publicKeysMap[remoteKey.KeyID] = remoteKey.Key
		tm.publicKeys = append(tm.publicKeys, &PublicKey{KeyID: remoteKey.KeyID, Key: remoteKey.Key})
		log.Info(nil, map[string]interface{}{
			"kid": remoteKey.KeyID,
		}, "Public key added")
	}

	devModeURL := config.GetKeycloakDevModeURL()
	if devModeURL != "" {
		remoteKeys, err = authclient.FetchKeys(fmt.Sprintf("%s/protocol/openid-connect/certs", devModeURL))
		if err != nil {
			log.Error(nil, map[string]interface{}{
				"err":      err,
				"keys_url": devModeURL,
			}, "unable to load public keys from remote service in Dev Mode")
			return nil, errors.New("unable to load public keys from remote service in Dev Mode")
		}
		for _, remoteKey := range remoteKeys {
			tm.publicKeysMap[remoteKey.KeyID] = remoteKey.Key
			tm.publicKeys = append(tm.publicKeys, &PublicKey{KeyID: remoteKey.KeyID, Key: remoteKey.Key})
			log.Info(nil, map[string]interface{}{
				"kid": remoteKey.KeyID,
			}, "Public key added")
		}
		// Add the public key which will be used to verify tokens generated in dev mode
		rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(configuration.DevModeRsaPrivateKey))
		if err != nil {
			return nil, err
		}
		tm.publicKeysMap["test-key"] = &rsaKey.PublicKey
		tm.publicKeys = append(tm.publicKeys, &PublicKey{KeyID: "test-key", Key: &rsaKey.PublicKey})
		log.Info(nil, map[string]interface{}{
			"kid": "test-key",
		}, "Public key added")
	}

	return tm, nil
}

// NewManagerWithPublicKey returns a new token Manager for handling tokens with the only public key
func NewManagerWithPublicKey(id string, key *rsa.PublicKey) Manager {
	return &tokenManager{
		publicKeysMap: map[string]*rsa.PublicKey{id: key},
		publicKeys:    []*PublicKey{{KeyID: id, Key: key}},
	}
}

func (mgm *tokenManager) IsServiceAccount(ctx context.Context, serviceName string) bool {
	token := goajwt.ContextJWT(ctx)
	if token == nil {
		return false
	}
	accountName := token.Claims.(jwt.MapClaims)["service_accountname"]
	if accountName == nil {
		return false
	}
	accountNameTyped, isString := accountName.(string)

	// https://github.com/fabric8-services/fabric8-auth/commit/8d7f5a3646974ae8820893d75c29f3f5e9b1ff66#diff-6b1a7621961d1f6fe7463db59c5afef5R379
	return isString && (accountNameTyped == serviceName)
}

// ParseToken parses token claims
func (mgm *tokenManager) ParseToken(ctx context.Context, tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		kid, ok := token.Header["kid"]
		if !ok {
			log.Error(ctx, map[string]interface{}{}, "There is no 'kid' header in the token")
			return nil, errors.New("there is no 'kid' header in the token")
		}
		key := mgm.PublicKey(fmt.Sprintf("%s", kid))
		if key == nil {
			log.Error(ctx, map[string]interface{}{
				"kid": kid,
			}, "There is no public key with such ID")
			return nil, errors.Errorf("there is no public key with such ID: %s", kid)
		}
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	claims := token.Claims.(*TokenClaims)
	if token.Valid {
		return claims, nil
	}
	return nil, errors.WithStack(errors.New("token is not valid"))
}

func (mgm *tokenManager) Locate(ctx context.Context) (uuid.UUID, error) {
	token := goajwt.ContextJWT(ctx)
	if token == nil {
		return uuid.UUID{}, errors.New("Missing token") // TODO, make specific tokenErrors
	}
	id := token.Claims.(jwt.MapClaims)["sub"]
	if id == nil {
		return uuid.UUID{}, errors.New("Missing sub")
	}
	idTyped, err := uuid.FromString(id.(string))
	if err != nil {
		return uuid.UUID{}, errors.New("uuid not of type string")
	}
	return idTyped, nil
}

// PublicKey returns the public key by the ID
func (mgm *tokenManager) PublicKey(kid string) *rsa.PublicKey {
	return mgm.publicKeysMap[kid]
}

// PublicKeys returns all the public keys
func (mgm *tokenManager) PublicKeys() []*rsa.PublicKey {
	keys := make([]*rsa.PublicKey, 0, len(mgm.publicKeysMap))
	for _, key := range mgm.publicKeys {
		keys = append(keys, key.Key)
	}
	return keys
}

// CheckClaims checks if all the required claims are present in the access token
func CheckClaims(claims *TokenClaims) error {
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

// ReadManagerFromContext extracts the token manager from the context
func ReadManagerFromContext(ctx context.Context) (*Manager, error) {
	tm := tokencontext.ReadTokenManagerFromContext(ctx)
	if tm == nil {
		log.Error(ctx, map[string]interface{}{
			"token": tm,
		}, "missing token manager")

		return nil, errors.New("Missing token manager")
	}
	tokenManager := tm.(Manager)
	return &tokenManager, nil
}
