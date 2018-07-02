package controller_test

import (
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/suite"
)

type TestFiltersREST struct {
	// composing with the DBTestSuite to get the Configuration out-of-the-box, even though this particular Controller
	// does not need an access to the DB.
	gormtestsupport.DBTestSuite
}

func TestRunFiltersREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestFiltersREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (rest *TestFiltersREST) TestListFiltersOK() {

	// given
	svc := goa.New("filterService")
	ctrl := controller.NewFilterController(svc, rest.Configuration)
	// when
	res, filters := test.ListFilterOK(rest.T(), svc.Context, svc, ctrl)
	// then
	compareWithGoldenAgnostic(rest.T(), filepath.Join("test-files", "filter", "list", "list_available_filters.golden.json"), filters)
	assertResponseHeaders(rest.T(), res)
}
