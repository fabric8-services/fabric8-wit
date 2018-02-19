package authz_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space/authz"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestAuthz(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	suite.Run(t, new(TestAuthzSuite))
}

type TestAuthzSuite struct {
	suite.Suite
	authzService *authz.KeycloakAuthzService
	config       *configuration.Registry
}

func (s *TestAuthzSuite) SetupSuite() {
	var err error
	s.config, err = configuration.Get()
	if err != nil {
		panic(fmt.Errorf("failed to setup the configuration: %s", err.Error()))
	}
	s.authzService = authz.NewAuthzService(s.config)
}

func (s *TestAuthzSuite) TestFailsIfNoTokenInContext() {
	ctx := context.Background()
	_, err := s.authzService.Authorize(ctx, "", uuid.NewV4().String())
	require.Error(s.T(), err)
	require.IsType(s.T(), errors.UnauthorizedError{}, err)
}
