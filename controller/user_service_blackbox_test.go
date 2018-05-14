package controller_test

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/account/tenant"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/controller"
	testjwt "github.com/fabric8-services/fabric8-wit/test/jwt"
	testrecorder "github.com/fabric8-services/fabric8-wit/test/recorder"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/require"
)

type TenantConfig struct {
	url string
}

func (t TenantConfig) GetTenantServiceURL() string {
	return t.url
}
func TestShowUser(t *testing.T) {
	// given
	ctrl := controller.NewUserServiceController(goa.New("test"))
	config := TenantConfig{
		url: "http://tenant",
	}
	r, err := testrecorder.New(
		"../test/data/account/show_tenant",
		testrecorder.WithJWTMatcher("../test/jwt/public_key.pem"),
	)
	require.NoError(t, err)
	defer r.Stop()
	ctrl.ShowTenant = func(ctx context.Context) (*tenant.TenantSingle, error) {
		return account.ShowTenant(ctx, config, configuration.WithRoundTripper(r.Transport))
	}

	t.Run("ok", func(t *testing.T) {
		// given
		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c")
		require.NoError(t, err)
		// when/then
		test.ShowUserServiceOK(t, ctx, goa.New("test"), ctrl)
	})

	t.Run("not found", func(t *testing.T) {
		// given
		ctx, err := testjwt.NewJWTContext("83fdcae2-634f-4a52-958a-f723cb621700")
		require.NoError(t, err)
		// when/then
		test.ShowUserServiceNotFound(t, ctx, goa.New("test"), ctrl)
	})
}
