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

type TestFiltersREST struct {
	// composing with the DBTestSuite to get the Configuration out-of-the-box, even though this particular Controller
	// does not need an access to the DB.
	gormtestsupport.DBTestSuite
	db *gormapplication.GormDB
}

func TestRunFiltersREST(t *testing.T) {
	resource.Require(t, resource.Database)
	pwd, err := os.Getwd()
	require.NoError(t, err)
	suite.Run(t, &TestFiltersREST{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (rest *TestFiltersREST) TestListFiltersOK() {
	resetFn := rest.DisableGormCallbacks()
	defer resetFn()

	// given
	svc := goa.New("filterService")
	ctrl := controller.NewFilterController(svc, rest.Configuration)
	// when
	res, filters := test.ListFilterOK(rest.T(), svc.Context, svc, ctrl)
	// then
	compareWithGolden(rest.T(), filepath.Join("test-files", "filter", "list", "list_available_filters.golden.json"), filters)
	assertResponseHeaders(rest.T(), res)
}
