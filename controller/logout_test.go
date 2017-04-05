package controller

import (
	"testing"

	"github.com/almighty/almighty-core/app/test"
	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/suite"
)

type TestLogoutREST struct {
	suite.Suite
	configuration *config.ConfigurationData
}

func TestRunLogoutREST(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	configuration, err := config.GetConfigurationData()
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
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Logout-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, &LogoutController{Controller: svc.NewController("logout"), logoutService: &login.KeycloakLogoutService{}, configuration: rest.configuration}
}

func (rest *TestLogoutREST) TestLogoutRedirects() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	svc, ctrl := rest.UnSecuredController()

	redirect := "http://domain.com"
	test.LogoutLogoutTemporaryRedirect(t, svc.Context, svc, ctrl, &redirect)
}

func (rest *TestLogoutREST) TestLogoutWithoutReffererAndRedirectParamsBadRequest() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	svc, ctrl := rest.UnSecuredController()

	test.LogoutLogoutBadRequest(t, svc.Context, svc, ctrl, nil)
}
