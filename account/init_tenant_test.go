package account_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/errors"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/configuration"
	testjwt "github.com/fabric8-services/fabric8-wit/test/jwt"
	testrecorder "github.com/fabric8-services/fabric8-wit/test/recorder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TenantConfig struct {
	url string
}

func (t TenantConfig) GetTenantServiceURL() string {
	return t.url
}
func TestShowTenant(t *testing.T) {

	// given
	r, err := testrecorder.New(
		"../test/data/account/show_tenant",
		testrecorder.WithJWTMatcher("../test/jwt/public_key.pem"),
	)
	require.NoError(t, err)
	defer r.Stop()
	config := TenantConfig{
		url: "http://tenant",
	}

	t.Run("ok", func(t *testing.T) {
		// given
		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c")
		require.NoError(t, err)
		// when
		result, err := account.ShowTenant(ctx, config, configuration.WithRoundTripper(r.Transport))
		// then
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "foo@foo.com", *result.Data.Attributes.Email)
	})

	t.Run("not found", func(t *testing.T) {
		// given
		ctx, err := testjwt.NewJWTContext("83fdcae2-634f-4a52-958a-f723cb621700")
		require.NoError(t, err)
		// when
		result, err := account.ShowTenant(ctx, config, configuration.WithRoundTripper(r.Transport))
		// then
		require.Nil(t, result)
		require.Error(t, err)
		t.Logf("tenant error: %v", err)
		assert.IsType(t, errors.NotFoundError{}, err)
	})
}

func TestCleanTenant(t *testing.T) {

	// given
	r, err := testrecorder.New(
		"../test/data/account/delete_tenant",
		testrecorder.WithJWTMatcher("../test/jwt/public_key.pem"),
	)
	require.NoError(t, err)
	defer r.Stop()
	config := TenantConfig{
		url: "http://tenant",
	}

	t.Run("ok", func(t *testing.T) {
		// given
		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c")
		require.NoError(t, err)
		// when
		err = account.CleanTenant(ctx, config, false, configuration.WithRoundTripper(r.Transport))
		// then
		require.NoError(t, err)
	})

	t.Run("failure", func(t *testing.T) {

		t.Run("internal server error", func(t *testing.T) {
			// given
			ctx, err := testjwt.NewJWTContext("83fdcae2-634f-4a52-958a-f723cb621700")
			require.NoError(t, err)
			// when
			err = account.CleanTenant(ctx, config, false, configuration.WithRoundTripper(r.Transport))
			// then
			require.Error(t, err)
			assert.IsType(t, errors.InternalError{}, err)
		})

		t.Run("other error with a message", func(t *testing.T) {
			// given
			ctx, err := testjwt.NewJWTContext("2610c5dc-d700-4b86-b979-2b103e0b1144")
			require.NoError(t, err)
			// when
			err = account.CleanTenant(ctx, config, false, configuration.WithRoundTripper(r.Transport))
			// then
			require.Error(t, err)
			assert.IsType(t, errors.UnauthorizedError{}, err)
		})

		t.Run("other error without a message", func(t *testing.T) {
			// given
			ctx, err := testjwt.NewJWTContext("73a3b0ce-4917-44db-9979-90b1219ca2c6")
			require.NoError(t, err)
			// when
			err = account.CleanTenant(ctx, config, false, configuration.WithRoundTripper(r.Transport))
			// then
			require.Error(t, err)
			assert.IsType(t, errors.InternalError{}, err)
		})
	})

}
