package jwt

import (
	"context"

	jwt "github.com/dgrijalva/jwt-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

// NewJWTContext creates a context with a JWT having the given `subject`
// using the private key that is located in the 'privateKeyPath'
func NewJWTContext(subject, privateKeyPath string) (context.Context, error) {
	claims := jwt.MapClaims{}
	claims["sub"] = subject
	token := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	if privateKeyPath == "" {
		privateKeyPath = "../test/jwt/private_key.pem"
	}
	key, err := PrivateKey(privateKeyPath)
	if err != nil {
		return nil, err
	}
	signed, err := token.SignedString(key)
	if err != nil {
		return nil, err
	}
	token.Raw = signed

	return goajwt.WithJWT(context.Background(), token), nil
}
