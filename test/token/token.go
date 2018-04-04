package token

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/token"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

var (
	TokenManager token.Manager
)

func init() {
	TokenManager = NewManager()
}

// GenerateToken generates a JWT token and signs it using the given private key
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

func PrivateKey() *rsa.PrivateKey {
	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(configuration.DevModeRsaPrivateKey))
	if err != nil {
		panic("Failed: " + err.Error())
	}
	return rsaKey
}
