package controller_test

import (
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/configuration"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/stretchr/testify/suite"
)

type TestUserREST struct {
	suite.Suite
	config *configuration.ConfigurationData
}

func (rest *TestUserREST) TestRunUserREST(t *testing.T) {
	resource.Require(rest.T(), resource.UnitTest)
	t.Parallel()
	suite.Run(rest.T(), &TestUserREST{})
}

func (rest *TestUserREST) SetupSuite() {
	config, err := configuration.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	rest.config = config
}

func (rest *TestUserREST) TestCurrentAuthorizedMissingUUID() {
	svc := testsupport.ServiceAsServiceAccountUser("Users-ServiceAccount-Service", testsupport.TestIdentity)
	ctr := NewUserController(svc, rest.config)
	test.ShowUserTemporaryRedirect(rest.T(), svc.Context, svc, ctr)
}
