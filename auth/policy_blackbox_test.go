package auth_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/almighty/almighty-core/auth"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestPolicy(t *testing.T) {
	resource.Require(t, resource.Remote)
	suite.Run(t, new(TestPolicySuite))
}

type TestPolicySuite struct {
	suite.Suite
	policyManager *auth.KeycloakPolicyManager
}

func (s *TestPolicySuite) SetupSuite() {
	s.policyManager = auth.NewKeycloakPolicyManager(configuration)
}

func (s *TestPolicySuite) TearDownSuite() {
	cleanKeycloakResources(s.T())
}

func (s *TestPolicySuite) TestGetPolicyOK() {
	policy, policyID := createPermissionWithPolicy(s)

	r := &goa.RequestData{
		Request: &http.Request{Host: "domain.io"},
	}
	obtainedPolicy, newPat, err := s.policyManager.GetPolicy(context.Background(), r, policyID)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), obtainedPolicy)
	require.NotNil(s.T(), newPat)
	require.NotNil(s.T(), obtainedPolicy.ID)
	require.Equal(s.T(), policyID, *obtainedPolicy.ID)
	require.Equal(s.T(), policy.Config.UserIDs, obtainedPolicy.Config.UserIDs)
	require.Equal(s.T(), policy.Type, obtainedPolicy.Type)
	require.Equal(s.T(), policy.Name, obtainedPolicy.Name)
}

func (s *TestPolicySuite) TestUpdatePolicyOK() {
	policy, policyID := createPermissionWithPolicy(s)
	secondTestUserID := getUserID(s.T(), configuration.GetKeycloakTestUser2Name(), configuration.GetKeycloakTestUser2Secret())
	policy.RemoveUserFromPolicy(secondTestUserID)
	policy.ID = &policyID
	r := &goa.RequestData{
		Request: &http.Request{Host: "domain.io"},
	}
	pat := getProtectedAPITokenOK(s.T())
	err := s.policyManager.UpdatePolicy(context.Background(), r, *policy, pat)
	require.Nil(s.T(), err)
	obtainedPolicy, newPat, err := s.policyManager.GetPolicy(context.Background(), r, policyID)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), obtainedPolicy)
	require.NotNil(s.T(), newPat)
	require.NotNil(s.T(), obtainedPolicy.ID)
	require.Equal(s.T(), policyID, *obtainedPolicy.ID)
	require.Equal(s.T(), policy.Config.UserIDs, obtainedPolicy.Config.UserIDs)
	require.Equal(s.T(), policy.Type, obtainedPolicy.Type)
	require.Equal(s.T(), policy.Name, obtainedPolicy.Name)
}

func createPermissionWithPolicy(s *TestPolicySuite) (*auth.KeycloakPolicy, string) {
	ctx := context.Background()
	pat := getProtectedAPITokenOK(s.T())

	resourceID, _ := createResource(s.T(), ctx, pat)
	clientId, clientsEndpoint := getClientIDAndEndpoint(s.T())
	policyID, policy := createPolicy(s.T(), ctx, pat)
	require.NotNil(s.T(), policy)

	permission := auth.KeycloakPermission{
		Name:             "test-" + uuid.NewV4().String(),
		Type:             auth.PermissionTypeResource,
		Logic:            auth.PolicyLogicPossitive,
		DecisionStrategy: auth.PolicyDecisionStrategyUnanimous,
		// "config":{"resources":"[\"<ResourceID>\"]","applyPolicies":"[\"<PolicyID>\"]"}
		Config: auth.PermissionConfigData{
			Resources:     "[\"" + resourceID + "\"]",
			ApplyPolicies: "[\"" + policyID + "\"]",
		},
	}

	permissionID, err := auth.CreatePermission(ctx, clientsEndpoint, clientId, permission, pat)
	require.Nil(s.T(), err)
	require.NotEqual(s.T(), "", permissionID)

	return &policy, policyID
}
