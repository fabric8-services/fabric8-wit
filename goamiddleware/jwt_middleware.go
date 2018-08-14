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
	"crypto/rsa"
	"fmt"
	"net/http"

	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

// New returns a middleware to be used with the JWTSecurity DSL definitions of goa.  It supports the
// scopes claim in the JWT and ensures goa-defined Security DSLs are properly validated.
//
// The steps taken by the middleware are:
//
//     1. Extract the "Bearer" token from the Authorization header or query parameter
//     2. Validate the "Bearer" token against the key(s)
//        given to New
//     3. If scopes are defined in the design for the action, validate them
//        against the scopes presented by the JWT in the claim "scope", or if
//        that's not defined, "scopes".
//
// The `exp` (expiration) and `nbf` (not before) date checks are validated by the JWT library.
//
// validationKeys can be one of these:
//
//     * a string (for HMAC)
//     * a []byte (for HMAC)
//     * an rsa.PublicKey
//     * an ecdsa.PublicKey
//     * a slice of any of the above
//
// The type of the keys determine the algorithm that will be used to do the check.  The goal of
// having lists of keys is to allow for key rotation, still check the previous keys until rotation
// has been completed.
//
// You can define an optional function to do additional validations on the token once the signature
// and the claims requirements are proven to be valid.  Example:
//
//    validationHandler, _ := goa.NewMiddleware(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
//        token := jwt.ContextJWT(ctx)
//        if val, ok := token.Claims["is_uncle"].(string); !ok || val != "ben" {
//            return jwt.ErrJWTError("you are not uncle ben's")
//        }
//    })
//
// Mount the middleware with the generated UseXX function where XX is the name of the scheme as
// defined in the design, e.g.:
//
//    app.UseJWT(jwt.New("secret", validationHandler, app.NewJWTSecurity()))
//
func New(validationKeys interface{}, validationFunc goa.Middleware, scheme *goa.JWTSecurity) goa.Middleware {
	var rsaKeys []*rsa.PublicKey
	var hmacKeys [][]byte

	rsaKeys, ecdsaKeys, hmacKeys := partitionKeys(validationKeys)

	return func(nextHandler goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			var (
				incomingToken string
				err           error
			)

			if scheme.In == goa.LocHeader {
				if incomingToken, err = extractTokenFromHeader(scheme.Name, req); err != nil {
					return err
				}
			} else if scheme.In == goa.LocQuery {
				if incomingToken, err = extractTokenFromQueryParam(scheme.Name, req); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("whoops, security scheme with location (in) %q not supported", scheme.In)
			}

			var (
				token     *jwt.Token
				validated = false
			)

			if len(rsaKeys) > 0 {
				token, err = validateRSAKeys(rsaKeys, "RS", incomingToken)
				validated = err == nil
			}

			if !validated && len(ecdsaKeys) > 0 {
				token, err = validateECDSAKeys(ecdsaKeys, "ES", incomingToken)
				validated = err == nil
			}

			if !validated && len(hmacKeys) > 0 {
				token, err = validateHMACKeys(hmacKeys, "HS", incomingToken)
				//validated = err == nil
			}

			if err != nil {
				return goajwt.ErrJWTError(fmt.Sprintf("JWT validation failed: %s", err))
			}

			scopesInClaim, scopesInClaimList, err := parseClaimScopes(token)
			if err != nil {
				goa.LogError(ctx, err.Error())
				return goajwt.ErrJWTError(err)
			}

			requiredScopes := goa.ContextRequiredScopes(ctx)

			for _, scope := range requiredScopes {
				if !scopesInClaim[scope] {
					msg := "authorization failed: required 'scope' or 'scopes' not present in JWT claim"
					return goajwt.ErrJWTError(msg, "required", requiredScopes, "scopes", scopesInClaimList)
				}
			}

			ctx = goajwt.WithJWT(ctx, token)
			if validationFunc != nil {
				nextHandler = validationFunc(nextHandler)
			}
			return nextHandler(ctx, rw, req)
		}
	}
}
