package controller_test

import (
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/suite"
)

type TestCollaboratorsREST struct {
	suite.Suite
	config CollaboratorsConfiguration
}

func TestRunCollaboratorsREST(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	suite.Run(t, &TestCollaboratorsREST{config: &dummyCollaboratorsConfiguration{}})
}

func (rest *TestCollaboratorsREST) SetupTest() {
}

func (rest *TestCollaboratorsREST) TearDownTest() {
}

func (rest *TestCollaboratorsREST) UnSecuredController() (*goa.Service, *CollaboratorsController) {
	svc := testsupport.ServiceAsUser("Logout-Service", testsupport.TestIdentity)
	return svc, NewCollaboratorsController(svc, nil, rest.config)
}

func (rest *TestCollaboratorsREST) TestListRedirect() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	svc, ctrl := rest.UnSecuredController()
	test.ListCollaboratorsTemporaryRedirect(t, svc.Context, svc, ctrl, uuid.NewV4())
}

type dummyCollaboratorsConfiguration struct {
}

func (c *dummyCollaboratorsConfiguration) GetKeycloakEndpointEntitlement(*http.Request) (string, error) {
	return "", nil
}

func (c *dummyCollaboratorsConfiguration) GetCacheControlCollaborators() string {
	return ""
}

func (c *dummyCollaboratorsConfiguration) IsAuthorizationEnabled() bool {
	return true
}

func (c *dummyCollaboratorsConfiguration) GetAuthServiceURL() string {
	return "localhost"
}
