package authz_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space/authz"

	"github.com/dgrijalva/jwt-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	testSpaceID = "a2c706fa-7421-452c-8a34-91016e5a4eab"
)

var (
	scopes = []string{"read:test", "admin:test"}
)

func TestAuthz(t *testing.T) {
	resource.Require(t, resource.Remote)
	suite.Run(t, new(TestAuthzSuite))
}

type TestAuthzSuite struct {
	suite.Suite
	authzService        *authz.KeycloakAuthzService
	config              *configuration.ConfigurationData
	entitlementEndpoint string
	test1Token          string
	test2Token          string
}

func (s *TestAuthzSuite) SetupSuite() {
	var err error
	s.config, err = configuration.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("failed to setup the configuration: %s", err.Error()))
	}
	s.authzService = authz.NewAuthzService(s.config)
	s.entitlementEndpoint, err = s.config.GetKeycloakEndpointEntitlement(nil)
	if err != nil {
		panic(fmt.Errorf("failed to get endpoint from configuration: %s", err.Error()))
	}
	tokenEndpoint, err := s.config.GetKeycloakEndpointToken(nil)
	if err != nil {
		panic(fmt.Errorf("failed to get endpoint from configuration: %s", err.Error()))
	}

	token, err := controller.GenerateUserToken(context.Background(), tokenEndpoint, s.config, s.config.GetKeycloakTestUserName(), s.config.GetKeycloakTestUserSecret())
	if err != nil {
		panic(fmt.Errorf("failed to generate token: %s", err.Error()))
	}
	if token.Token.AccessToken == nil {
		panic("failed to generate token")
	}

	s.test1Token = *token.Token.AccessToken

	token, err = controller.GenerateUserToken(context.Background(), tokenEndpoint, s.config, s.config.GetKeycloakTestUser2Name(), s.config.GetKeycloakTestUser2Secret())
	if err != nil {
		panic(fmt.Errorf("failed to generate token: %s", err.Error()))
	}
	if token.Token.AccessToken == nil {
		panic("failed to generate token")
	}

	s.test2Token = *token.Token.AccessToken
}

func (s *TestAuthzSuite) TestFailsIfNoTokenInContext() {
	ctx := context.Background()
	_, err := s.authzService.Authorize(ctx, s.entitlementEndpoint, testSpaceID)
	require.NotNil(s.T(), err)
	require.IsType(s.T(), errors.UnauthorizedError{}, err)
}

func (s *TestAuthzSuite) TestUserAmongSpaceCollaboratorsOK() {
	ok := s.checkPermissions(s.test1Token, testSpaceID)
	require.True(s.T(), ok)
}

func (s *TestAuthzSuite) TestUserIsNotAmongSpaceCollaboratorsFails() {
	ok := s.checkPermissions(s.test2Token, testSpaceID)
	require.False(s.T(), ok)
}

func (s *TestAuthzSuite) checkPermissions(token string, spaceID string) bool {
	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Raw = token
	ctx := goajwt.WithJWT(context.Background(), tk)
	authzService := authz.NewAuthzService(s.config)
	ok, err := authzService.Authorize(ctx, s.entitlementEndpoint, spaceID)
	require.Nil(s.T(), err)
	return ok
}
