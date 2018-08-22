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
	"strings"

	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

// TokenContext is a new goa middleware that aims to extract the token from the
// Authorization header when possible. If the Authorization header is missing in the request,
// no error is returned. However, if the Authorization header contains a
// token, it will be stored it in the context.
func TokenContext(validationKeys interface{}, validationFunc goa.Middleware, scheme *goa.JWTSecurity) goa.Middleware {
	var rsaKeys []*rsa.PublicKey
	var hmacKeys [][]byte

	rsaKeys, ecdsaKeys, hmacKeys := partitionKeys(validationKeys)

	return func(nextHandler goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			// TODO: implement the QUERY string handler too
			if scheme.In != goa.LocHeader {
				log.Error(ctx, nil, fmt.Sprintf("whoops, security scheme with location (in) %q not supported", scheme.In))
				return fmt.Errorf("whoops, security scheme with location (in) %q not supported", scheme.In)
			}
			val := req.Header.Get(scheme.Name)
			if val != "" && strings.HasPrefix(strings.ToLower(val), "bearer ") {
				log.Debug(ctx, nil, "found header 'Authorization: Bearer JWT-token...'")
				incomingToken := strings.Split(val, " ")[1]
				log.Debug(ctx, nil, "extracted the incoming token %v ", incomingToken)

				var (
					token  *jwt.Token
					err    error
					parsed = false
				)

				if len(rsaKeys) > 0 {
					token, err = validateRSAKeys(rsaKeys, "RS", incomingToken)
					if err == nil {
						parsed = true
					}
				}

				if !parsed && len(ecdsaKeys) > 0 {
					token, err = validateECDSAKeys(ecdsaKeys, "ES", incomingToken)
					if err == nil {
						parsed = true
					}
				}

				if !parsed && len(hmacKeys) > 0 {
					token, err = validateHMACKeys(hmacKeys, "HS", incomingToken)
					if err == nil {
						parsed = true
					}
				}

				if !parsed {
					log.Warn(ctx, nil, "unable to parse JWT token: %v", err)
				}

				ctx = goajwt.WithJWT(ctx, token)
			}

			return nextHandler(ctx, rw, req)
		}
	}
}
