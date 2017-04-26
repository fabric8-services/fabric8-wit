package controller_test

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

func TestRunSearchUser(t *testing.T) {
	resource.Require(t, resource.Database)
	pwd, err := os.Getwd()
	require.Nil(t, err)
	suite.Run(t, &TestSearchUserSearch{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

type TestSearchUserSearch struct {
	gormtestsupport.DBTestSuite
	db         *gormapplication.GormDB
	svc        *goa.Service
	controller *SearchController
	clean      func()
}

func (s *TestSearchUserSearch) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.svc = goa.New("test")
	s.db = gormapplication.NewGormDB(s.DB)
	s.controller = NewSearchController(s.svc, s.db, s.Configuration)
}

func (s *TestSearchUserSearch) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *TestSearchUserSearch) TearDownTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

type userSearchTestArgs struct {
	pageOffset *string
	pageLimit  *int
	q          string
}

type userSearchTestExpect func(*testing.T, okScenarioUserSearchTest, *app.UserList)
type userSearchTestExpects []userSearchTestExpect

type okScenarioUserSearchTest struct {
	name                  string
	userSearchTestArgs    userSearchTestArgs
	userSearchTestExpects userSearchTestExpects
}

func (s *TestSearchUserSearch) TestUsersSearchOK() {

	idents := s.createTestData()
	defer s.cleanTestData(idents)

	tests := []okScenarioUserSearchTest{
		{"With uppercase fullname query", userSearchTestArgs{s.offset("0"), limit(10), "TEST_AB"}, userSearchTestExpects{s.totalCount(1)}},
		{"With uppercase fullname query", userSearchTestArgs{s.offset("0"), limit(10), "TEST_AB"}, userSearchTestExpects{s.totalCount(1)}},
		{"With uppercase email query", userSearchTestArgs{s.offset("0"), limit(10), "EMAIL_TEST_AB"}, userSearchTestExpects{s.totalCount(1)}},
		{"With lowercase email query", userSearchTestArgs{s.offset("0"), limit(10), "email_test_ab"}, userSearchTestExpects{s.totalCount(1)}},
		{"with special chars", userSearchTestArgs{s.offset("0"), limit(10), "&:\n!#%?*"}, userSearchTestExpects{s.totalCount(0)}},
		{"with * to list all", userSearchTestArgs{s.offset("0"), limit(10), "*"}, userSearchTestExpects{s.totalCountAtLeast(len(idents))}},
		{"with multi page", userSearchTestArgs{s.offset("0"), limit(10), "TEST"}, userSearchTestExpects{s.hasLinks("Next")}},
		{"with last page", userSearchTestArgs{s.offset(strconv.Itoa(len(idents) - 1)), limit(10), "TEST"}, userSearchTestExpects{s.hasNoLinks("Next"), s.hasLinks("Prev")}},
		{"with different values", userSearchTestArgs{s.offset("0"), s.limit(10), "TEST"}, userSearchTestExpects{s.differentValues()}},
	}

	for _, tt := range tests {
		_, result := test.UsersSearchOK(s.T(), context.Background(), s.svc, s.controller, tt.userSearchTestArgs.pageLimit, tt.userSearchTestArgs.pageOffset, tt.userSearchTestArgs.q)
		for _, userSearchTestExpect := range tt.userSearchTestExpects {
			userSearchTestExpect(s.T(), tt, result)
		}
	}
}

func (s *TestSearchUserSearch) TestUsersSearchBadRequest() {
	t := s.T()
	tests := []struct {
		name               string
		userSearchTestArgs userSearchTestArgs
	}{
		{"with empty query", userSearchTestArgs{s.offset("0"), limit(10), ""}},
	}

	for _, tt := range tests {
		test.UsersSearchBadRequest(t, context.Background(), s.svc, s.controller, tt.userSearchTestArgs.pageLimit, tt.userSearchTestArgs.pageOffset, tt.userSearchTestArgs.q)
	}
}

func (s *TestSearchUserSearch) createTestData() []account.Identity {
	names := []string{"TEST_A", "TEST_AB", "TEST_B", "TEST_C"}
	for i := 0; i < 20; i++ {
		names = append(names, "TEST_"+strconv.Itoa(i))
	}

	idents := []account.Identity{}

	err := application.Transactional(s.db, func(app application.Application) error {
		for _, name := range names {

			user := account.User{
				FullName: name,
				ImageURL: "http://example.org/" + name + ".png",
				Email:    strings.ToLower("email_" + name + "@" + name + ".org"),
			}
			err := app.Users().Create(context.Background(), &user)
			require.Nil(s.T(), err)

			ident := account.Identity{
				User:         user,
				Username:     uuid.NewV4().String() + "test" + name,
				ProviderType: "kc",
			}
			err = app.Identities().Create(context.Background(), &ident)
			require.Nil(s.T(), err)

			idents = append(idents, ident)
		}
		return nil
	})
	require.Nil(s.T(), err)
	return idents
}

func (s *TestSearchUserSearch) cleanTestData(idents []account.Identity) {
	err := application.Transactional(s.db, func(app application.Application) error {
		db := app.(*gormapplication.GormTransaction).DB()
		db = db.Unscoped()
		for _, ident := range idents {
			db.Delete(ident)
			db.Delete(&account.User{}, "id = ?", ident.User.ID)
		}
		return nil
	})
	require.Nil(s.T(), err)
}

func (s *TestSearchUserSearch) totalCount(count int) userSearchTestExpect {
	return func(t *testing.T, scenario okScenarioUserSearchTest, result *app.UserList) {
		if got := result.Meta.TotalCount; got != count {
			t.Errorf("%s got = %v, want %v", scenario.name, got, count)
		}
	}
}

func (s *TestSearchUserSearch) totalCountAtLeast(count int) userSearchTestExpect {
	return func(t *testing.T, scenario okScenarioUserSearchTest, result *app.UserList) {
		got := result.Meta.TotalCount
		if !(got >= count) {
			t.Errorf("%s got %v, wanted at least %v", scenario.name, got, count)
		}
	}
}

func (s *TestSearchUserSearch) hasLinks(linkNames ...string) userSearchTestExpect {
	return func(t *testing.T, scenario okScenarioUserSearchTest, result *app.UserList) {
		for _, linkName := range linkNames {
			link := linkName
			if reflect.Indirect(reflect.ValueOf(result.Links)).FieldByName(link).IsNil() {
				t.Errorf("%s got empty link, wanted %s", scenario.name, link)
			}
		}
	}
}

func (s *TestSearchUserSearch) hasNoLinks(linkNames ...string) userSearchTestExpect {
	return func(t *testing.T, scenario okScenarioUserSearchTest, result *app.UserList) {
		for _, linkName := range linkNames {
			if !reflect.Indirect(reflect.ValueOf(result.Links)).FieldByName(linkName).IsNil() {
				t.Errorf("%s got link, wanted empty %s", scenario.name, linkName)
			}
		}
	}
}

func (s *TestSearchUserSearch) differentValues() userSearchTestExpect {
	return func(t *testing.T, scenario okScenarioUserSearchTest, result *app.UserList) {
		var prev *app.UserData

		for i := range result.Data {
			u := result.Data[i]
			if prev == nil {
				prev = u
			} else {
				if *prev.Attributes.FullName == *u.Attributes.FullName {
					t.Errorf("%s got equal Fullname, wanted different %s", scenario.name, *u.Attributes.FullName)
				}
				if *prev.Attributes.ImageURL == *u.Attributes.ImageURL {
					t.Errorf("%s got equal ImageURL, wanted different %s", scenario.name, *u.Attributes.ImageURL)
				}
				if *prev.ID == *u.ID {
					t.Errorf("%s got equal ID, wanted different %s", scenario.name, *u.ID)
				}
				if prev.Type != u.Type {
					t.Errorf("%s got non equal Type, wanted same %s", scenario.name, u.Type)
				}
			}
		}
	}
}

func (s *TestSearchUserSearch) limit(n int) *int {
	return &n
}
func (s *TestSearchUserSearch) offset(n string) *string {
	return &n
}
