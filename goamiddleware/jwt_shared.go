// The MIT License (MIT)

// Copyright (c) 2015 Raphael Simon and goa Contributors

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package goamiddleware

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"net/http"
	"sort"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

func extractTokenFromHeader(schemeName string, req *http.Request) (string, error) {
	val := req.Header.Get(schemeName)
	if val == "" {
		return "", goajwt.ErrJWTError(fmt.Sprintf("missing header %q", schemeName))
	}

	if !strings.HasPrefix(strings.ToLower(val), "bearer ") {
		return "", goajwt.ErrJWTError(fmt.Sprintf("invalid or malformed %q header, expected 'Bearer JWT-token...'", val))
	}

	incomingToken := strings.Split(val, " ")[1]

	return incomingToken, nil
}

func extractTokenFromQueryParam(schemeName string, req *http.Request) (string, error) {
	incomingToken := req.URL.Query().Get(schemeName)
	if incomingToken == "" {
		return "", goajwt.ErrJWTError(fmt.Sprintf("missing parameter %q", schemeName))
	}

	return incomingToken, nil
}

// validScopeClaimKeys are the claims under which scopes may be found in a token
var validScopeClaimKeys = []string{"scope", "scopes"}

// parseClaimScopes parses the "scope" or "scopes" parameter in the Claims. It
// supports two formats:
//
// * a list of strings
//
// * a single string with space-separated scopes (akin to OAuth2's "scope").
//
// An empty string is an explicit claim of no scopes.
func parseClaimScopes(token *jwt.Token) (map[string]bool, []string, error) {
	scopesInClaim := make(map[string]bool)
	var scopesInClaimList []string
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, nil, fmt.Errorf("unsupport claims shape")
	}
	for _, k := range validScopeClaimKeys {
		if rawscopes, ok := claims[k]; ok && rawscopes != nil {
			switch scopes := rawscopes.(type) {
			case string:
				for _, scope := range strings.Split(scopes, " ") {
					scopesInClaim[scope] = true
					scopesInClaimList = append(scopesInClaimList, scope)
				}
			case []interface{}:
				for _, scope := range scopes {
					if val, ok := scope.(string); ok {
						scopesInClaim[val] = true
						scopesInClaimList = append(scopesInClaimList, val)
					}
				}
			default:
				return nil, nil, fmt.Errorf("unsupported scope format in incoming JWT claim, was type %T", scopes)
			}
			break
		}
	}
	sort.Strings(scopesInClaimList)
	return scopesInClaim, scopesInClaimList, nil
}

// partitionKeys sorts keys by their type.
func partitionKeys(k interface{}) ([]*rsa.PublicKey, []*ecdsa.PublicKey, [][]byte) {
	var (
		rsaKeys   []*rsa.PublicKey
		ecdsaKeys []*ecdsa.PublicKey
		hmacKeys  [][]byte
	)

	switch typed := k.(type) {
	case []byte:
		hmacKeys = append(hmacKeys, typed)
	case [][]byte:
		hmacKeys = typed
	case string:
		hmacKeys = append(hmacKeys, []byte(typed))
	case []string:
		for _, s := range typed {
			hmacKeys = append(hmacKeys, []byte(s))
		}
	case *rsa.PublicKey:
		rsaKeys = append(rsaKeys, typed)
	case []*rsa.PublicKey:
		rsaKeys = typed
	case *ecdsa.PublicKey:
		ecdsaKeys = append(ecdsaKeys, typed)
	case []*ecdsa.PublicKey:
		ecdsaKeys = typed
	}

	return rsaKeys, ecdsaKeys, hmacKeys
}

func validateRSAKeys(rsaKeys []*rsa.PublicKey, algo, incomingToken string) (token *jwt.Token, err error) {
	for _, pubkey := range rsaKeys {
		token, err = jwt.Parse(incomingToken, func(token *jwt.Token) (interface{}, error) {
			if !strings.HasPrefix(token.Method.Alg(), algo) {
				return nil, goajwt.ErrJWTError(fmt.Sprintf("Unexpected signing method: %v", token.Header["alg"]))
			}
			return pubkey, nil
		})
		if err == nil {
			return
		}
	}
	return
}

func validateECDSAKeys(ecdsaKeys []*ecdsa.PublicKey, algo, incomingToken string) (token *jwt.Token, err error) {
	for _, pubkey := range ecdsaKeys {
		token, err = jwt.Parse(incomingToken, func(token *jwt.Token) (interface{}, error) {
			if !strings.HasPrefix(token.Method.Alg(), algo) {
				return nil, goajwt.ErrJWTError(fmt.Sprintf("Unexpected signing method: %v", token.Header["alg"]))
			}
			return pubkey, nil
		})
		if err == nil {
			return
		}
	}
	return
}

func validateHMACKeys(hmacKeys [][]byte, algo, incomingToken string) (token *jwt.Token, err error) {
	for _, key := range hmacKeys {
		token, err = jwt.Parse(incomingToken, func(token *jwt.Token) (interface{}, error) {
			if !strings.HasPrefix(token.Method.Alg(), algo) {
				return nil, goajwt.ErrJWTError(fmt.Sprintf("Unexpected signing method: %v", token.Header["alg"]))
			}
			return key, nil
		})
		if err == nil {
			return
		}
	}
	return
}
