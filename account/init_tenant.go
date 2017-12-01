package account

import (
	"fmt"
	"net/http"

	"context"

	"net/url"

	"github.com/fabric8-services/fabric8-wit/account/tenant"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/goasupport"
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

// NewCleanTenant creates a new tenant service in oso
func NewCleanTenant(config tenantConfig) func(context.Context) error {
	return func(ctx context.Context) error {
		return CleanTenant(ctx, config)
	}
}

// CodebaseInitTenantProvider the function that provides a `tenant.TenantSingle`
type CodebaseInitTenantProvider func(context.Context) (*tenant.TenantSingle, error)

// NewShowTenant view an existing tenant in oso
func NewShowTenant(config tenantConfig) CodebaseInitTenantProvider {
	return func(ctx context.Context) (*tenant.TenantSingle, error) {
		return ShowTenant(ctx, config)
	}
}

// InitTenant creates a new tenant service in oso
func InitTenant(ctx context.Context, config tenantConfig) error {

	c, err := createClient(ctx, config)
	if err != nil {
		return err
	}

	// Ignore response for now
	_, err = c.SetupTenant(goasupport.ForwardContextRequestID(ctx), tenant.SetupTenantPath())

	return err
}

// UpdateTenant updates excisting tenant in oso
func UpdateTenant(ctx context.Context, config tenantConfig) error {

	c, err := createClient(ctx, config)
	if err != nil {
		return err
	}

	// Ignore response for now
	_, err = c.UpdateTenant(goasupport.ForwardContextRequestID(ctx), tenant.UpdateTenantPath())

	return err
}

// CleanTenant cleans out a tenant in oso.
func CleanTenant(ctx context.Context, config tenantConfig) error {

	c, err := createClient(ctx, config)
	if err != nil {
		return err
	}

	res, err := c.CleanTenant(goasupport.ForwardContextRequestID(ctx), tenant.CleanTenantPath())
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		jsonErr, err := c.DecodeJSONAPIErrors(res)
		if err == nil {
			if len(jsonErr.Errors) > 0 {
				return errors.NewInternalError(ctx, fmt.Errorf(jsonErr.Errors[0].Detail))
			}
		}
	}
	return nil
}

// ShowTenant fetches the current tenant state.
func ShowTenant(ctx context.Context, config tenantConfig) (*tenant.TenantSingle, error) {

	c, err := createClient(ctx, config)
	if err != nil {
		return nil, err
	}

	res, err := c.ShowTenant(goasupport.ForwardContextRequestID(ctx), tenant.ShowTenantPath())
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusOK {
		tenant, err := c.DecodeTenantSingle(res)
		if err != nil {
			return nil, errors.NewInternalError(ctx, err)
		}
		return tenant, nil
	} else if res.StatusCode > 400 {
		jsonErr, err := c.DecodeJSONAPIErrors(res)
		if err == nil {
			if len(jsonErr.Errors) > 0 {
				return nil, errors.NewInternalError(ctx, fmt.Errorf(jsonErr.Errors[0].Detail))
			}
		}
	}
	return nil, errors.NewInternalError(ctx, fmt.Errorf("Unknown response "+res.Status))
}

func createClient(ctx context.Context, config tenantConfig) (*tenant.Client, error) {
	u, err := url.Parse(config.GetTenantServiceURL())
	if err != nil {
		return nil, err
	}

	c := tenant.New(goaclient.HTTPClientDoer(http.DefaultClient))
	c.Host = u.Host
	c.Scheme = u.Scheme
	c.SetJWTSigner(goasupport.NewForwardSigner(ctx))
	return c, nil
}
