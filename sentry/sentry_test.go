package sentry

import (
	"context"
	"crypto/rsa"
	"reflect"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/login/tokencontext"
	"github.com/fabric8-services/fabric8-wit/token"

	"github.com/dgrijalva/jwt-go"
	"github.com/getsentry/raven-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

func Test_extractUserInfo(t *testing.T) {

	identity := account.Identity{
		ID:       uuid.NewV4(),
		Username: "testuser",
	}

	tests := []struct {
		name    string
		ctx     context.Context
		want    *raven.User
		wantErr bool
	}{
		{
			name:    "Given some random context",
			ctx:     context.Background(),
			wantErr: true,
		},
		{
			name:    "fail on no token",
			ctx:     failOnNoToken(t),
			wantErr: true,
		},
		{
			name:    "fail on parsing token",
			ctx:     failOnParsingToken(t),
			wantErr: true,
		},
		{
			name:    "pass on parsing token",
			ctx:     validToken(t, identity.ID.String(), identity.Username),
			wantErr: false,
			want: &raven.User{
				Username: identity.Username,
				ID:       identity.ID.String(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractUserInfo(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractUserInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractUserInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func failOnNoToken(t *testing.T) context.Context {
	config, err := configuration.New("")
	if err != nil {
		t.Errorf("failed to create new config: %v", err)
	}

	m, err := token.NewFakeManager(config)
	if err != nil {
		t.Errorf("failed to create new manager: %v", err)
	}

	return tokencontext.ContextWithTokenManager(context.Background(), m)
}

func failOnParsingToken(t *testing.T) context.Context {
	ctx := failOnNoToken(t)
	// Here we add a token which is not complete
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	ctx = goajwt.WithJWT(ctx, token)
	return ctx
}

func validToken(t *testing.T, identityID string, identityUsername string) context.Context {
	ctx := failOnNoToken(t)
	token := GenerateToken(t, identityID, identityUsername)
	ctx = goajwt.WithJWT(ctx, token)
	return ctx
}

// GenerateToken generates a JWT token and signs it using the default private key
func GenerateToken(t *testing.T, identityID string, identityUsername string) *jwt.Token {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims.(jwt.MapClaims)["uuid"] = identityID
	token.Claims.(jwt.MapClaims)["preferred_username"] = identityUsername
	token.Claims.(jwt.MapClaims)["sub"] = identityID

	key, kid, err := privateKey()
	if err != nil {
		t.Fatalf("could not retrieve private keys: %v", errors.WithStack(err))
	}

	token.Header["kid"] = kid
	token.Raw, err = token.SignedString(key)
	if err != nil {
		t.Fatalf("could not extract signed string: %v", errors.WithStack(err))
	}

	return token
}

func privateKey() (*rsa.PrivateKey, string, error) {
	key, kid := []byte(configuration.DefaultUserAccountPrivateKey), configuration.DefaultUserAccountPrivateKeyID
	pk, err := jwt.ParseRSAPrivateKeyFromPEM(key)
	return pk, kid, err
}
