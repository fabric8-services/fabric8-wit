package token

import (
	"context"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/login/tokencontext"
	"github.com/fabric8-services/fabric8-wit/token"

	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa/client"
	jwtgoa "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

var (
	TokenManager token.Manager
)

func init() {
	TokenManager = NewManager()
}

// GenerateTokenObject generates a JWT token and signs it using the given private key
func GenerateTokenObject(identityID string, identityUsername string, privateKey *rsa.PrivateKey) (*jwt.Token, error) {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims.(jwt.MapClaims)["uuid"] = identityID
	token.Claims.(jwt.MapClaims)["preferred_username"] = identityUsername
	token.Claims.(jwt.MapClaims)["sub"] = identityID

	token.Claims.(jwt.MapClaims)["jti"] = uuid.NewV4().String()
	token.Claims.(jwt.MapClaims)["session_state"] = uuid.NewV4().String()
	token.Claims.(jwt.MapClaims)["iat"] = time.Now().Unix()
	token.Claims.(jwt.MapClaims)["exp"] = time.Now().Unix() + 60*60*24*30
	token.Claims.(jwt.MapClaims)["nbf"] = 0
	token.Claims.(jwt.MapClaims)["iss"] = "wit"
	token.Claims.(jwt.MapClaims)["typ"] = "Bearer"

	token.Claims.(jwt.MapClaims)["approved"] = true
	token.Claims.(jwt.MapClaims)["name"] = identityUsername
	token.Claims.(jwt.MapClaims)["company"] = "Company Inc."
	token.Claims.(jwt.MapClaims)["given_name"] = identityUsername
	token.Claims.(jwt.MapClaims)["family_name"] = identityUsername
	token.Claims.(jwt.MapClaims)["email"] = fmt.Sprintf("%s@email.com", identityUsername)

	token.Header["kid"] = "test-key"
	var err error
	token.Raw, err = token.SignedString(privateKey)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return token, nil
}

// GenerateToken generates a JWT token and signs it using the given private key
func GenerateToken(identityID string, identityUsername string, privateKey *rsa.PrivateKey) (string, error) {
	token, err := GenerateTokenObject(identityID, identityUsername, privateKey)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return token.Raw, nil
}

// NewManager returns a new token Manager for handling tokens
func NewManager() token.Manager {
	return token.NewManagerWithPublicKey("test-key", &PrivateKey().PublicKey)
}

// EmbedTokenInContext generates a token and embed it into the context
func EmbedTokenInContext(t *testing.T, sub, username string) (context.Context, string, error) {
	// Generate Token with an identity that doesn't exist in the database
	tokenString, err := GenerateToken(sub, username, PrivateKey())
	if err != nil {
		return nil, "", err
	}

	extracted, err := parse(t, tokenString)
	if err != nil {
		return nil, "", err
	}

	// Embed Token in the context
	ctx := jwtgoa.WithJWT(context.Background(), extracted)
	ctx = tokencontext.ContextWithTokenManager(ctx, TokenManager)
	return ctx, tokenString, nil
}

func parse(t *testing.T, tokenString string) (*jwt.Token, error) {
	keyFunc := keyFunction()
	jwtToken, err := jwt.Parse(tokenString, keyFunc)
	require.NoError(t, err)
	return jwtToken, nil
}

func keyFunction() jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		kid := token.Header["kid"]
		if kid == nil {
			return nil, errors.New("there is no 'kid' header in the token")
		}
		key := TokenManager.PublicKey(fmt.Sprintf("%s", kid))
		if key == nil {
			return nil, errors.New(fmt.Sprintf("there is no public key with such ID: %s", kid))
		}
		return key, nil
	}
}

func ContextWithTokenAndRequestID(t *testing.T) (context.Context, uuid.UUID, string, string) {
	identityID := uuid.NewV4()
	ctx, ctxToken, err := EmbedTokenInContext(t, identityID.String(), uuid.NewV4().String())
	require.NoError(t, err)

	reqID := uuid.NewV4().String()
	ctx = client.SetContextRequestID(ctx, reqID)

	return ctx, identityID, ctxToken, reqID
}

func PrivateKey() *rsa.PrivateKey {
	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(configuration.DevModeRsaPrivateKey))
	if err != nil {
		panic("Failed: " + err.Error())
	}
	return rsaKey
}
