package account

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/fabric8-services/fabric8-wit/account/tenant"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/rest"

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
func NewCleanTenant(config tenantConfig) func(context.Context, bool) error {
	return func(ctx context.Context, remove bool) error {
		return CleanTenant(ctx, config, remove)
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
	res, err := c.SetupTenant(goasupport.ForwardContextRequestID(ctx), tenant.SetupTenantPath())
	defer rest.CloseResponse(res)

	return err
}

// UpdateTenant updates excisting tenant in oso
func UpdateTenant(ctx context.Context, config tenantConfig) error {

	c, err := createClient(ctx, config)
	if err != nil {
		return err
	}

	// Ignore response for now
	res, err := c.UpdateTenant(goasupport.ForwardContextRequestID(ctx), tenant.UpdateTenantPath())
	defer rest.CloseResponse(res)

	return err
}

// CleanTenant cleans out a tenant in oso.
func CleanTenant(ctx context.Context, config tenantConfig, remove bool, options ...configuration.HTTPClientOption) error {

	c, err := createClient(ctx, config, options...)
	if err != nil {
		return err
	}

	res, err := c.CleanTenant(goasupport.ForwardContextRequestID(ctx), tenant.CleanTenantPath(), &remove)
	if err != nil {
		return err
	}
	defer rest.CloseResponse(res)

	if res.StatusCode != http.StatusNoContent { // the delete tenant operation returns a `204 No Content` response in case of success
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
func ShowTenant(ctx context.Context, config tenantConfig, options ...configuration.HTTPClientOption) (*tenant.TenantSingle, error) {
	c, err := createClient(ctx, config, options...)
	if err != nil {
		return nil, err
	}
	res, err := c.ShowTenant(goasupport.ForwardContextRequestID(ctx), tenant.ShowTenantPath())
	if err != nil {
		return nil, err
	}
	defer rest.CloseResponse(res)
	switch res.StatusCode {
	case http.StatusOK:
		tenant, err := c.DecodeTenantSingle(res)
		if err != nil {
			return nil, errors.NewInternalError(ctx, err)
		}
		return tenant, nil
	case http.StatusNotFound:
		jsonErr, err := c.DecodeJSONAPIErrors(res)
		if err == nil {
			if len(jsonErr.Errors) > 0 {
				log.Error(ctx, map[string]interface{}{
					"error_msg": jsonErr.Errors[0].Detail,
				}, "failed to retrieve tenant")
				return nil, errors.NewNotFoundError("tenants", *jsonErr.Errors[0].ID)
			}
		} else {
			log.Error(ctx, map[string]interface{}{"error_msg": err}, "failed to parse JSON-API error response")
		}

	}
	return nil, errors.NewInternalError(ctx, fmt.Errorf("Unknown response: '%v' (%d)", res.Status, res.StatusCode))
}

// createClient creates a client to the tenant service with the given configuration and options for the underlying HTTP client
func createClient(ctx context.Context, config tenantConfig, options ...configuration.HTTPClientOption) (*tenant.Client, error) {
	u, err := url.Parse(config.GetTenantServiceURL())
	if err != nil {
		return nil, err
	}
	httpClient := http.DefaultClient
	// apply options
	for _, opt := range options {
		opt(httpClient)
	}
	c := tenant.New(goaclient.HTTPClientDoer(httpClient))
	c.Host = u.Host
	c.Scheme = u.Scheme
	c.SetJWTSigner(goasupport.NewForwardSigner(ctx))
	return c, nil
}
