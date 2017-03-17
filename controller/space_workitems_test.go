package controller_test

import (
	"net/http"
	"net/url"
	"testing"

	"golang.org/x/net/context"

	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/suite"
)

type TestSpaceWorkitemREST struct {
	gormtestsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
	ctx   context.Context
}

func TestRunSpaceWorkitemREST(t *testing.T) {
	suite.Run(t, &TestSpaceWorkitemREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestSpaceWorkitemREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)

	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	rest.ctx = goa.NewContext(context.Background(), nil, req, params)
}

func (rest *TestSpaceWorkitemREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSpaceWorkitemREST) SecuredController() (*goa.Service, *SpaceWorkitemsController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("WorkItem-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewSpaceWorkitemsController(svc, rest.db)
}

func (rest *TestSpaceWorkitemREST) UnSecuredController() (*goa.Service, *SpaceWorkitemsController) {
	svc := goa.New("WorkItem-Service")
	return svc, NewSpaceWorkitemsController(svc, rest.db)
}
