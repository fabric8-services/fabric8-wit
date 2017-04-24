package controller_test

import (
	"testing"

	"time"

	"github.com/almighty/almighty-core/app/test"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/suite"
)

type TestStatusREST struct {
	gormtestsupport.DBTestSuite

	clean func()
}

func TestRunStatusREST(t *testing.T) {
	suite.Run(t, &TestStatusREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestStatusREST) SetupTest() {
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestStatusREST) TearDownTest() {
	rest.clean()
}

func (rest *TestStatusREST) SecuredController() (*goa.Service, *StatusController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Status-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
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
