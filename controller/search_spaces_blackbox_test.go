package controller_test

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/stretchr/testify/suite"

	"github.com/goadesign/goa"
)

type args struct {
	pageOffset *string
	pageLimit  *int
	q          string
}

type expect func(*testing.T, okScenario, *app.SearchSpaceList)
type expects []expect

type okScenario struct {
	name    string
	args    args
	expects expects
}

type TestSearchSpacesREST struct {
	gormtestsupport.DBTestSuite
}

func TestRunSearchSpacesREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestSearchSpacesREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (rest *TestSearchSpacesREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
}

func (rest *TestSearchSpacesREST) SecuredController() (*goa.Service, *SearchController) {
	svc := testsupport.ServiceAsUser("Search-Service", testsupport.TestIdentity)
	return svc, NewSearchController(svc, rest.GormDB, rest.Configuration)
}

func (rest *TestSearchSpacesREST) UnSecuredController() (*goa.Service, *SearchController) {
	svc := goa.New("Search-Service")
	return svc, NewSearchController(svc, rest.GormDB, rest.Configuration)
}

func (rest *TestSearchSpacesREST) TestSpacesSearchOK() {
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB,
		tf.Spaces(2, func(fxt *tf.TestFixture, idx int) error {
			fxt.Spaces[idx].Description = strings.ToTitle("description for " + fxt.Spaces[idx].Name)
			return nil
		}),
		tf.Iterations(2, func(fxt *tf.TestFixture, idx int) error {
			fxt.Iterations[idx].SpaceID = fxt.Spaces[idx].ID
			return nil
		}),
	)
	tests := []okScenario{
		{"With uppercase fullname query", args{offset("0"), limit(10), strings.ToUpper(fxt.Spaces[0].Name)}, expects{totalCount(1)}},
		{"With lowercase fullname query", args{offset("0"), limit(10), strings.ToLower(fxt.Spaces[0].Name)}, expects{totalCount(1)}},
		{"With uppercase description query", args{offset("0"), limit(10), "DESCRIPTION FOR " + fxt.Spaces[0].Name}, expects{totalCount(1)}},
		{"With lowercase description query", args{offset("0"), limit(10), "description for " + fxt.Spaces[0].Name}, expects{totalCount(1)}},
		{"with special chars", args{offset("0"), limit(10), "&:\n!#%?*"}, expects{totalCount(0)}},
		{"with * to list all", args{offset("0"), limit(10), "*"}, expects{totalCountAtLeast(len(fxt.Spaces))}},
		{"with multi page", args{offset("0"), limit(1), "space"}, expects{hasLinks("Next")}},
		{"with last page", args{offset(strconv.Itoa(len(fxt.Spaces) - 1)), limit(10), "space"}, expects{hasNoLinks("Next"), hasLinks("Prev")}},
		{"with different values", args{offset("0"), limit(10), fxt.Spaces[0].Name}, expects{differentValues()}},
	}
	svc, ctrl := rest.UnSecuredController()
	// when/then
	for _, tt := range tests {
		_, result := test.SpacesSearchOK(rest.T(), svc.Context, svc, ctrl, tt.args.pageLimit, tt.args.pageOffset, tt.args.q)
		for _, expect := range tt.expects {
			expect(rest.T(), tt, result)
		}
	}
}

func totalCount(count int) expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchSpaceList) {
		if got := result.Meta.TotalCount; got != count {
			t.Errorf("%s got = %v, want %v", scenario.name, got, count)
		}
	}
}

func totalCountAtLeast(count int) expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchSpaceList) {
		got := result.Meta.TotalCount
		if !(got >= count) {
			t.Errorf("%s got %v, wanted at least %v", scenario.name, got, count)
		}
	}
}

func hasLinks(linkNames ...string) expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchSpaceList) {
		for _, linkName := range linkNames {
			link := linkName
			if reflect.Indirect(reflect.ValueOf(result.Links)).FieldByName(link).IsNil() {
				t.Errorf("%s got empty link, wanted %s", scenario.name, link)
			}
		}
	}
}

func hasNoLinks(linkNames ...string) expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchSpaceList) {
		for _, linkName := range linkNames {
			if !reflect.Indirect(reflect.ValueOf(result.Links)).FieldByName(linkName).IsNil() {
				t.Errorf("%s got link, wanted empty %s", scenario.name, linkName)
			}
		}
	}
}

func differentValues() expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchSpaceList) {
		var prev *app.Space

		for i := range result.Data {
			s := result.Data[i]
			if prev == nil {
				prev = s
			} else {
				if *prev.Attributes.Name == *s.Attributes.Name {
					t.Errorf("%s got equal name, wanted different %s", scenario.name, *s.Attributes.Name)
				}
				if *prev.Attributes.Description == *s.Attributes.Description {
					t.Errorf("%s got equal description, wanted different %s", scenario.name, *s.Attributes.Description)
				}
				if *prev.ID == *s.ID {
					t.Errorf("%s got equal ID, wanted different %s", scenario.name, *s.ID)
				}
				if prev.Type != s.Type {
					t.Errorf("%s got non equal Type, wanted same %s", scenario.name, s.Type)
				}
			}
		}
	}
}

func limit(n int) *int {
	return &n
}
func offset(n string) *string {
	return &n
}
