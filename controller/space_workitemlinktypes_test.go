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

type TestSpaceWorkitemlinktypesREST struct {
	gormtestsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
	ctx   context.Context
}

func TestRunSpaceWorkitemlinktypesREST(t *testing.T) {
	suite.Run(t, &TestSpaceWorkitemlinktypesREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestSpaceWorkitemlinktypesREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)

	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	rest.ctx = goa.NewContext(context.Background(), nil, req, params)
}

func (rest *TestSpaceWorkitemlinktypesREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSpaceWorkitemlinktypesREST) SecuredController() (*goa.Service, *SpaceWorkitemlinktypesController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("WorkItemLinkType-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewSpaceWorkitemlinktypesController(svc, rest.db)
}

func (rest *TestSpaceWorkitemlinktypesREST) UnSecuredController() (*goa.Service, *SpaceWorkitemlinktypesController) {
	svc := goa.New("WorkItemLinkType-Service")
	return svc, NewSpaceWorkitemlinktypesController(svc, rest.db)
}
