package controller_test

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	testsupport "github.com/fabric8-services/fabric8-wit/test"

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type args struct {
	pageOffset *string
	pageLimit  *int
	q          string
	sort       *string
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
	db *gormapplication.GormDB
}

func TestRunSearchSpacesREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestSearchSpacesREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestSearchSpacesREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
	rest.db = gormapplication.NewGormDB(rest.DB)
}

func (rest *TestSearchSpacesREST) SecuredController() (*goa.Service, *SearchController) {
	svc := testsupport.ServiceAsUser("Search-Service", testsupport.TestIdentity)
	return svc, NewSearchController(svc, rest.db, rest.Configuration)
}

func (rest *TestSearchSpacesREST) UnSecuredController() (*goa.Service, *SearchController) {
	svc := goa.New("Search-Service")
	return svc, NewSearchController(svc, rest.db, rest.Configuration)
}

func (rest *TestSearchSpacesREST) TestSpacesSearchOK() {
	// given
	prefix := time.Now().Format("2006_Jan_2_15_04_05_") // using a unique prefix to make sure the test data will not collide with existing, older spaces.
	idents, err := createTestData(rest.db, prefix)
	require.NoError(rest.T(), err)
	tests := []okScenario{
		{"With uppercase fullname query", args{ptr.String("0"), ptr.Int(10), prefix + "TEST_AB", nil}, expects{totalCount(1)}},
		{"With lowercase fullname query", args{ptr.String("0"), ptr.Int(10), prefix + "TEST_AB", nil}, expects{totalCount(1)}},
		{"With uppercase description query", args{ptr.String("0"), ptr.Int(10), "DESCRIPTION FOR " + prefix + "TEST_AB", nil}, expects{totalCount(1)}},
		{"With lowercase description query", args{ptr.String("0"), ptr.Int(10), "description for " + prefix + "test_ab", nil}, expects{totalCount(1)}},
		{"with special chars", args{ptr.String("0"), ptr.Int(10), "&:\n!#%?*", nil}, expects{totalCount(0)}},
		{"with * to list all", args{ptr.String("0"), ptr.Int(10), "*", nil}, expects{totalCountAtLeast(len(idents))}},
		{"with multi page", args{ptr.String("0"), ptr.Int(10), prefix + "TEST", nil}, expects{hasLinks("Next")}},
		{"with last page", args{ptr.String(strconv.Itoa(len(idents) - 1)), ptr.Int(10), prefix + "TEST", nil}, expects{hasNoLinks("Next"), hasLinks("Prev")}},
		{"with different values", args{ptr.String("0"), ptr.Int(10), prefix + "TEST", nil}, expects{differentValues()}},

		{
			name: "sorted by name ascending",
			args: args{
				pageOffset: ptr.String("0"),
				pageLimit:  ptr.Int(10),
				q:          prefix + "TEST",
				sort:       ptr.String("name"),
			},
			expects: expects{sortedSpaces(prefix+"TEST_0", prefix+"TEST_1")},
		},
		{
			name: "sorted by name descending",
			args: args{
				pageOffset: ptr.String("0"),
				pageLimit:  ptr.Int(10),
				q:          prefix + "TEST",
				sort:       ptr.String("-name"),
			},
			expects: expects{sortedSpaces(prefix+"TEST_C", prefix+"TEST_B")},
		},
		{
			name: "sorted by owner ascending",
			args: args{
				pageOffset: ptr.String("0"),
				pageLimit:  ptr.Int(10),
				q:          prefix + "TEST",
				sort:       ptr.String("owner"),
			},
			expects: expects{sortedSpaces(prefix+"TEST_AB", prefix+"TEST_B")},
		},
		{
			name: "sorted by owner descending",
			args: args{
				pageOffset: ptr.String("0"),
				pageLimit:  ptr.Int(10),
				q:          prefix + "TEST",
				sort:       ptr.String("-owner"),
			},
			expects: expects{sortedSpaces(prefix+"TEST_AB", prefix+"TEST_B")},
		},
		{
			name: "sorted by creation timestamp ascending",
			args: args{
				pageOffset: ptr.String("0"),
				pageLimit:  ptr.Int(10),
				q:          prefix + "TEST",
				sort:       ptr.String("created"),
			},
			expects: expects{sortedSpaces(prefix+"TEST_A", prefix+"TEST_AB")},
		},
		{
			name: "sorted by creation timestamp descending",
			args: args{
				pageOffset: ptr.String("0"),
				pageLimit:  ptr.Int(10),
				q:          prefix + "TEST",
				sort:       ptr.String("-created"),
			},
			expects: expects{sortedSpaces(prefix+"TEST_19", prefix+"TEST_18")},
		},
	}
	svc, ctrl := rest.UnSecuredController()
	// when/then
	for _, tt := range tests {
		_, result := test.SpacesSearchOK(rest.T(), svc.Context, svc, ctrl, tt.args.pageLimit, tt.args.pageOffset, tt.args.q, tt.args.sort)
		for _, expect := range tt.expects {
			expect(rest.T(), tt, result)
		}
	}
}

func createTestData(db application.DB, prefix string) ([]space.Space, error) {
	names := []string{prefix + "TEST_A", prefix + "TEST_AB", prefix + "TEST_B", prefix + "TEST_C"}
	for i := 0; i < 20; i++ {
		names = append(names, prefix+"TEST_"+strconv.Itoa(i))
	}

	spaces := []space.Space{}

	err := application.Transactional(db, func(app application.Application) error {
		for _, name := range names {
			space := space.Space{
				Name:            name,
				Description:     strings.ToTitle("description for " + name),
				SpaceTemplateID: spacetemplate.SystemLegacyTemplateID,
			}
			newSpace, err := app.Spaces().Create(context.Background(), &space)
			if err != nil {
				return err
			}
			spaces = append(spaces, *newSpace)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to insert testdata %v", err)
	}
	return spaces, nil
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

func sortedSpaces(expectedSpaces ...string) expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchSpaceList) {
		for i, space := range expectedSpaces {
			assert.Equal(t, space, *result.Data[i].Attributes.Name)
		}
	}
}
