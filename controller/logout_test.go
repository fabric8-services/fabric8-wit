package controller

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"
	config "github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/suite"
)

type TestLogoutREST struct {
	suite.Suite
	configuration *config.Registry
}

func TestRunLogoutREST(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	configuration, err := config.Get()
	if err != nil {
		t.Fatalf("Failed to setup the configuration: %s", err.Error())
	}
	suite.Run(t, &TestLogoutREST{configuration: configuration})
}

func (rest *TestLogoutREST) SetupTest() {
}

func (rest *TestLogoutREST) TearDownTest() {
}

func (rest *TestLogoutREST) UnSecuredController() (*goa.Service, *LogoutController) {
	svc := testsupport.ServiceAsUser("Logout-Service", testsupport.TestIdentity)
	return svc, &LogoutController{Controller: svc.NewController("logout"), configuration: rest.configuration}
}

func (rest *TestLogoutREST) TestLogoutRedirects() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	svc, ctrl := rest.UnSecuredController()
	test.LogoutLogoutTemporaryRedirect(t, svc.Context, svc, ctrl)
}
