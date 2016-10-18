package account_test

import (
	"fmt"
	"os"
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	if _, c := os.LookupEnv(resource.Database); c != false {
		var err error
		if err = configuration.Setup(""); err != nil {
			panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
		}

		db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())
		if err != nil {
			panic("Failed to connect database: " + err.Error())
		}
		defer db.Close()

		// Migrate the schema
		err = migration.Migrate(db.DB())
		if err != nil {
			panic(err.Error())
		}

	}

	ec := m.Run()
	os.Exit(ec)
}

func TestUserByEmails(t *testing.T) {
	resource.Require(t, resource.Database)
	// this test makes sure that UserByEmails eliminates deleted entries
	ctx := context.Background()
	userRepo := account.NewUserRepository(db)
	identityRepo := account.NewIdentityRepository(db)
	identity := account.Identity{
		FullName: "Test User Integration 123",
		ImageURL: "http://images.com/42",
	}
	email := "primary@example.com"

	err := identityRepo.Create(ctx, &identity)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Unscoped().Delete(&identity)
	}()
	user1 := account.User{Email: email, Identity: identity}
	err = userRepo.Create(ctx, &user1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Unscoped().Delete(&user1)
	}()
	users, err := userRepo.Query(account.UserByEmails([]string{email}), account.UserWithIdentity())
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEqual(t, 0, len(users))
	found := false
	for _, u := range users {
		if u.Email == email {
			found = true
			break
		}
	}
	if found == false {
		t.Errorf("Newly inserted email %v not found in DB", email)
	}
	// try to fetch user by identity
	u, err := userRepo.Query(account.UserFilterByIdentity(identity.ID, db))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, email, u[0].Email)

	// try filtering with non-available uuid
	u, err = userRepo.Query(account.UserFilterByIdentity(uuid.NewV4(), db))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, len(u))

}
