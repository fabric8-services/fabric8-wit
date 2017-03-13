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

type TestSpaceWorkitemtypesREST struct {
	gormtestsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
	ctx   context.Context
}

func TestRunSpaceWorkitemtypesREST(t *testing.T) {
	suite.Run(t, &TestSpaceWorkitemtypesREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestSpaceWorkitemtypesREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)

	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	rest.ctx = goa.NewContext(context.Background(), nil, req, params)
}

func (rest *TestSpaceWorkitemtypesREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSpaceWorkitemtypesREST) SecuredController() (*goa.Service, *SpaceWorkitemtypesController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("WorkItemType-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewSpaceWorkitemtypesController(svc, rest.db)
}

func (rest *TestSpaceWorkitemtypesREST) UnSecuredController() (*goa.Service, *SpaceWorkitemtypesController) {
	svc := goa.New("WorkItemType-Service")
	return svc, NewSpaceWorkitemtypesController(svc, rest.db)
}
