package account

import (
	"net/http"

	"context"

	"net/url"

	"github.com/almighty/almighty-core/account/tenant"
	"github.com/almighty/almighty-core/goasupport"
	goaclient "github.com/goadesign/goa/client"
)

type tenantConfig interface {
	GetTenantServiceURL() string
}

// NewInitTenant creates a new tenant service in oso
func NewInitTenant(config tenantConfig) func(context.Context) error {
	return func(ctx context.Context) error {
		return InitTenant(ctx, config)
	}
}

// NewUpdateTenant creates a new tenant service in oso
func NewUpdateTenant(config tenantConfig) func(context.Context) error {
	return func(ctx context.Context) error {
		return UpdateTenant(ctx, config)
	}
}

// NewCurrentTenantVersion creates a new tenant service in oso
func NewCurrentTenantVersion(config tenantConfig) func(context.Context) error {
	return func(ctx context.Context) error {
		return CurrentTenantVersion(ctx, config)
	}
}

// NewLatestTenantVersion creates a new tenant service in oso
func NewLatestTenantVersion(config tenantConfig) func(context.Context) error {
	return func(ctx context.Context) error {
		return LatestTenantVersion(ctx, config)
	}
}

// InitTenant creates a new tenant service in oso
func InitTenant(ctx context.Context, config tenantConfig) error {

	// Check if user profile registrationComplete=True,
	// if false, dont initialize tenant

	u, err := url.Parse(config.GetTenantServiceURL())
	if err != nil {
		return err
	}

	c := tenant.New(goaclient.HTTPClientDoer(http.DefaultClient))
	c.Host = u.Host
	c.Scheme = u.Scheme
	c.SetJWTSigner(goasupport.NewForwardSigner(ctx))

	// Ignore response for now
	_, err = c.SetupTenant(ctx, tenant.SetupTenantPath())
	if err != nil {
		return err
	}
	return nil
}

// UpdateTenant creates a new tenant service in oso
func UpdateTenant(ctx context.Context, config tenantConfig) error {

	u, err := url.Parse(config.GetTenantServiceURL())
	if err != nil {
		return err
	}

	c := tenant.New(goaclient.HTTPClientDoer(http.DefaultClient))
	c.Host = u.Host
	c.Scheme = u.Scheme
	c.SetJWTSigner(goasupport.NewForwardSigner(ctx))

	// Ignore response for now
	_, err = c.UpdateTenant(ctx, tenant.SetupTenantPath())
	if err != nil {
		return err
	}
	return nil
}

// LatestTenantVersion returns the latest tenant version(s) available for the user to deploy
func LatestTenantVersion(ctx context.Context, config tenantConfig) error {
	/*
		There could be 2 scenarious where this would come handy:

		- tenant update has failed and the this API call when compared with the
		API call of 'Get Current' would show that

		- tenant update wasn't done and there's a new updated version available
	*/
	return nil
}

// CurrentTenantVersion gets the current version of tenant services deployed for the user.
func CurrentTenantVersion(ctx context.Context, config tenantConfig) error {
	// pipelines UI is currently calling OSO directly to get this info
	return nil
}
