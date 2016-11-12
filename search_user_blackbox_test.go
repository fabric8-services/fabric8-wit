package main_test

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"strings"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"golang.org/x/net/context"
)

type args struct {
	pageOffset *string
	pageLimit  *int
	q          string
}

type expect func(*testing.T, okScenario, *app.SearchResponseUsers)
type expects []expect

type okScenario struct {
	name    string
	args    args
	expects expects
}

func TestUsersSearchOK(t *testing.T) {
	resource.Require(t, resource.Database)

	idents := createTestData()
	defer cleanTestData(idents)

	tests := []okScenario{
		{"With uppercase fullname query", args{offset("0"), limit(10), "TEST_AB"}, expects{totalCount(1)}},
		{"With uppercase fullname query", args{offset("0"), limit(10), "TEST_AB"}, expects{totalCount(1)}},
		{"With uppercase email query", args{offset("0"), limit(10), "EMAIL_TEST_AB"}, expects{totalCount(1)}},
		{"With lowercase email query", args{offset("0"), limit(10), "email_test_ab"}, expects{totalCount(1)}},
		{"with special chars", args{offset("0"), limit(10), "&:\n!#%?*"}, expects{totalCount(0)}},
		{"with * to list all", args{offset("0"), limit(10), "*"}, expects{totalCountAtLeast(len(idents))}},
		{"with multi page", args{offset("0"), limit(10), "TEST"}, expects{hasLinks("Next")}},
		{"with last page", args{offset(strconv.Itoa(len(idents) - 1)), limit(10), "TEST"}, expects{hasNoLinks("Next"), hasLinks("Prev")}},
		{"with different values", args{offset("0"), limit(10), "TEST"}, expects{differentValues()}},
	}

	service := goa.New("TestUserSearch-Service")
	controller := NewSearchController(service, gormapplication.NewGormDB(DB))

	for _, tt := range tests {
		_, result := test.UsersSearchOK(t, context.Background(), service, controller, tt.args.pageLimit, tt.args.pageOffset, tt.args.q)
		for _, expect := range tt.expects {
			expect(t, tt, result)
		}
	}
}

func TestUsersSearchBadRequest(t *testing.T) {
	resource.Require(t, resource.Database)

	tests := []struct {
		name string
		args args
	}{
		{"with empty query", args{offset("0"), limit(10), ""}},
	}

	service := goa.New("TestUserSearch-Service")
	controller := NewSearchController(service, gormapplication.NewGormDB(DB))

	for _, tt := range tests {
		test.UsersSearchBadRequest(t, context.Background(), service, controller, tt.args.pageLimit, tt.args.pageOffset, tt.args.q)
	}
}

func createTestData() []account.Identity {
	names := []string{"TEST_A", "TEST_AB", "TEST_B", "TEST_C"}
	for i := 0; i < 20; i++ {
		names = append(names, "TEST_"+strconv.Itoa(i))
	}

	idents := []account.Identity{}

	err := application.Transactional(gormapplication.NewGormDB(DB), func(app application.Application) error {
		for _, name := range names {
			ident := account.Identity{
				FullName: name,
				ImageURL: "http://example.org/" + name + ".png",
				Emails: []account.User{
					account.User{
						Email: strings.ToLower("email_" + name + "@" + name + ".org"),
					},
					account.User{
						Email: strings.ToLower("email2_" + name + "@" + name + ".org"),
					},
				},
			}
			err := app.Identities().Create(context.Background(), &ident)
			if err != nil {
				return err
			}
			idents = append(idents, ident)
		}
		return nil
	})
	if err != nil {
		fmt.Println("Failed to insert testdata", err)
	}
	return idents
}

func cleanTestData(idents []account.Identity) {
	err := application.Transactional(gormapplication.NewGormDB(DB), func(app application.Application) error {
		db := app.(*gormapplication.GormTransaction).DB()
		db = db.Unscoped()
		for _, ident := range idents {
			db.Delete(ident)
			db.Delete(&account.User{}, "identity_id = ?", ident.ID)
		}
		return nil
	})
	if err != nil {
		fmt.Println("Failed to insert testdata", err)
	}
}

func totalCount(count int) expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchResponseUsers) {
		if got := result.Meta["total-count"].(int); got != count {
			t.Errorf("%s got = %v, want %v", scenario.name, got, count)
		}
	}
}

func totalCountAtLeast(count int) expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchResponseUsers) {
		got := result.Meta["total-count"].(int)
		if got == count {
			return
		}
		if got < count {
			t.Errorf("%s got %v, wanted at least %v", scenario.name, got, count)
		}
	}
}

func hasLinks(linkNames ...string) expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchResponseUsers) {
		for _, linkName := range linkNames {
			link := linkName
			if reflect.Indirect(reflect.ValueOf(result.Links)).FieldByName(link).IsNil() {
				t.Errorf("%s got empty link, wanted %s", scenario.name, link)
			}
		}
	}
}

func hasNoLinks(linkNames ...string) expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchResponseUsers) {
		for _, linkName := range linkNames {
			if !reflect.Indirect(reflect.ValueOf(result.Links)).FieldByName(linkName).IsNil() {
				t.Errorf("%s got link, wanted empty %s", scenario.name, linkName)
			}
		}
	}
}

func differentValues() expect {
	return func(t *testing.T, scenario okScenario, result *app.SearchResponseUsers) {
		var prev *app.Users

		for i := range result.Data {
			u := result.Data[i]
			if prev == nil {
				prev = u
			} else {
				if *prev.Attributes.Fullname == *u.Attributes.Fullname {
					t.Errorf("%s got equal Fullname, wanted different %s", scenario.name, *u.Attributes.Fullname)
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

func limit(n int) *int {
	return &n
}
func offset(n string) *string {
	return &n
}
