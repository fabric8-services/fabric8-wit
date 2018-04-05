package sentry

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/login/tokencontext"
	"github.com/fabric8-services/fabric8-wit/resource"
	testtoken "github.com/fabric8-services/fabric8-wit/test/token"

	"github.com/dgrijalva/jwt-go"
	"github.com/getsentry/raven-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

func failOnNoToken(t *testing.T) context.Context {
	// this is just normal context object with no, token
	// so this should fail saying no token available
	m := testtoken.NewManager()
	return tokencontext.ContextWithTokenManager(context.Background(), m)
}

func failOnParsingToken(t *testing.T) context.Context {
	ctx := failOnNoToken(t)
	// Here we add a token which is incomplete
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	ctx = goajwt.WithJWT(ctx, token)
	return ctx
}

func validToken(t *testing.T, identityID string, identityUsername string) context.Context {
	ctx := failOnNoToken(t)
	// Here we add a token that is perfectly valid
	token, err := testtoken.GenerateTokenObject(identityID, identityUsername, testtoken.PrivateKey())
	require.Nilf(t, err, "could not generate token: %v", errors.WithStack(err))

	ctx = goajwt.WithJWT(ctx, token)
	return ctx
}
func Test_extractUserInfo(t *testing.T) {
	resource.Require(t, resource.UnitTest)

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
				Email:    identity.Username + "@email.com",
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
			require.Equalf(t, tt.want, got, "extractUserInfo() = %v, want %v", got, tt.want)
		})
	}
}
