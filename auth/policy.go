package auth

import (
	"context"

	"github.com/almighty/almighty-core/errors"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

// AuthzPolicyManager represents a space collaborators policy manager
type AuthzPolicyManager interface {
	AuthzResourceManager
	GetPolicy(ctx context.Context, request *goa.RequestData, policyID string) (*KeycloakPolicy, *string, error)
	UpdatePolicy(ctx context.Context, request *goa.RequestData, policy KeycloakPolicy, pat string) error
	VerifyUser(ctx context.Context, request *goa.RequestData, resourceName string) (bool, error)
	AddUserToPolicy(p *KeycloakPolicy, userID string) bool
	RemoveUserFromPolicy(p *KeycloakPolicy, userID string) bool
}

// KeycloakPolicyManager implements AuthzPolicyManager interface
type KeycloakPolicyManager struct {
	configuration   KeycloakConfiguration
	resourceManager AuthzResourceManager
}

// NewKeycloakPolicyManager constructs KeycloakPolicyManager
func NewKeycloakPolicyManager(config KeycloakConfiguration, resourceManager AuthzResourceManager) *KeycloakPolicyManager {
	return &KeycloakPolicyManager{config, resourceManager}
}

// VerifyUser returns true if the user among the resource collaborators
func (m *KeycloakPolicyManager) VerifyUser(ctx context.Context, request *goa.RequestData, resourceName string) (bool, error) {
	entitlementEndpoint, err := m.configuration.GetKeycloakEndpointEntitlement(request)
	if err != nil {
		return false, err
	}
	token := goajwt.ContextJWT(ctx)
	if token == nil {
		return false, errors.NewUnauthorizedError("Missing token")
	}

	return VerifyResourceUser(ctx, token.Raw, resourceName, entitlementEndpoint)
}

// AddUserToPolicy adds the user ID to the policy
func (m *KeycloakPolicyManager) AddUserToPolicy(p *KeycloakPolicy, userID string) bool {
	return p.AddUserToPolicy(userID)
}

// RemoveUserFromPolicy removes the user ID from the policy
func (m *KeycloakPolicyManager) RemoveUserFromPolicy(p *KeycloakPolicy, userID string) bool {
	return p.RemoveUserFromPolicy(userID)
}

// GetPolicy obtains the space collaborators policy
func (m *KeycloakPolicyManager) GetPolicy(ctx context.Context, request *goa.RequestData, policyID string) (*KeycloakPolicy, *string, error) {
	clientsEndpoint, err := m.configuration.GetKeycloakEndpointClients(request)
	if err != nil {
		return nil, nil, err
	}
	pat, err := getPat(request, m.configuration)
	if err != nil {
		return nil, nil, err
	}
	publicClientID := m.configuration.GetKeycloakClientID()
	clientID, err := GetClientID(context.Background(), clientsEndpoint, publicClientID, pat)
	if err != nil {
		return nil, nil, err
	}

	policy, err := GetPolicy(ctx, clientsEndpoint, clientID, policyID, pat)
	if err != nil {
		return nil, nil, err
	}

	return policy, &pat, nil
}

// UpdatePolicy updates the space collaborators policy
func (m *KeycloakPolicyManager) UpdatePolicy(ctx context.Context, request *goa.RequestData, policy KeycloakPolicy, pat string) error {
	clientsEndpoint, err := m.configuration.GetKeycloakEndpointClients(request)
	if err != nil {
		return err
	}
	publicClientID := m.configuration.GetKeycloakClientID()
	clientID, err := GetClientID(context.Background(), clientsEndpoint, publicClientID, pat)
	if err != nil {
		return err
	}

	return UpdatePolicy(ctx, clientsEndpoint, clientID, policy, pat)
}

// CreateResource creates a keyclaok resource and associated permission and policy
func (m *KeycloakPolicyManager) CreateResource(ctx context.Context, request *goa.RequestData, name string, rType string, uri *string, scopes *[]string, userID string, policyName string) (*Resource, error) {
	return m.resourceManager.CreateResource(ctx, request, name, rType, uri, scopes, userID, policyName)
}

// DeleteResource deletes the keycloak resource and associated permission and policy
func (m *KeycloakPolicyManager) DeleteResource(ctx context.Context, request *goa.RequestData, resource Resource) error {
	return m.resourceManager.DeleteResource(ctx, request, resource)
}
