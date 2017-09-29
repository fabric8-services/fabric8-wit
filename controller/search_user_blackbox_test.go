package controller_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"

	"github.com/stretchr/testify/suite"
)

type TestSearchUserREST struct {
	suite.Suite
	config *configuration.ConfigurationData
}

func (rest *TestSearchUserREST) TestRunSearchUserREST(t *testing.T) {
	resource.Require(rest.T(), resource.UnitTest)
	t.Parallel()
	suite.Run(rest.T(), &TestSearchUserREST{})
}

func (rest *TestSearchUserREST) SetupSuite() {
	config, err := configuration.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("failed to setup the configuration: %s", err.Error()))
	}
	rest.config = config
}

func (rest *TestSearchUserREST) TestUsersSearchRedirected() {
	svc := testsupport.ServiceAsServiceAccountUser("Users-ServiceAccount-Service", testsupport.TestIdentity)
	ctr := controller.NewSearchController(svc, nil, rest.config)
	test.UsersSearchTemporaryRedirect(rest.T(), context.Background(), svc, ctr)
}
