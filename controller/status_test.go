package controller_test

import (
	"testing"

	"time"

	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/suite"
)

type TestStatusREST struct {
	gormtestsupport.DBTestSuite
}

func TestRunStatusREST(t *testing.T) {
	suite.Run(t, &TestStatusREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestStatusREST) SecuredController() (*goa.Service, *StatusController) {
	svc := testsupport.ServiceAsUser("Status-Service", testsupport.TestIdentity)
	return svc, NewStatusController(svc, rest.DB)
}

func (rest *TestStatusREST) UnSecuredController() (*goa.Service, *StatusController) {
	svc := goa.New("Status-Service")
	return svc, NewStatusController(svc, rest.DB)
}

func (rest *TestStatusREST) TestShowStatusOK() {
	t := rest.T()
	resource.Require(t, resource.Database)
	svc, ctrl := rest.UnSecuredController()
	_, res := test.ShowStatusOK(t, svc.Context, svc, ctrl)

	if res.Commit != "0" {
		t.Error("Commit not found")
	}
	if res.StartTime != StartTime {
		t.Error("StartTime is not correct")
	}
	_, err := time.Parse("2006-01-02T15:04:05Z", res.StartTime)
	if err != nil {
		t.Error("Incorrect layout of StartTime: ", err.Error())
	}
}
