package controller_test

import (
	"os"
	"path/filepath"
	"testing"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestRootREST struct {
	// composing with the DBTestSuite to get the Configuration out-of-the-box, even though this particular Controller
	// does not need an access to the DB.
	gormtestsupport.DBTestSuite
	db *gormapplication.GormDB
}

func TestRunRootREST(t *testing.T) {
	resource.Require(t, resource.Database)
	pwd, err := os.Getwd()
	require.NoError(t, err)
	suite.Run(t, &TestRootREST{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (rest *TestRootREST) TestListRootOK() {

	// given
	svc := goa.New("rootService")
	ctrl := controller.NewRootController(svc)

	// when
	res, root := test.ListRootOK(rest.T(), svc.Context, svc, ctrl)

	// then
	compareWithGoldenAgnostic(rest.T(), filepath.Join("test-files", "root", "list", "ok_root_endpoint.golden.json"), root)
	assertResponseHeaders(rest.T(), res)
}
