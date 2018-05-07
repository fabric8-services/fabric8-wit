package jwt

import (
	"crypto/rsa"
	"io/ioutil"

	jwt "github.com/dgrijalva/jwt-go"
)

// PrivateKey returns the PrivateKey from the given filename
func PrivateKey(filename string) (*rsa.PrivateKey, error) {
	rsaPrivateKey, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return jwt.ParseRSAPrivateKeyFromPEM(rsaPrivateKey)
}

// PublicKey returns the PublicKey from the given filename
func PublicKey(filename string) (*rsa.PublicKey, error) {
	rsaPublicKey, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return jwt.ParseRSAPublicKeyFromPEM(rsaPublicKey)
}
