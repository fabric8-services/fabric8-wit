package test

import (
	"context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/models"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// TestUser only creates in memory obj for testing purposes
var TestUser = account.User{
	ID:       uuid.NewV4(),
	Email:    "testdeveloper@testalm.io",
	FullName: "Test Developer",
}

// TestUser2 only creates in memory obj for testing purposes.
// This TestUser2 can be used to verify that some entity created by TestUser
// can be later updated or deleted (or not) by another user.
var TestUser2 = account.User{
	ID:       uuid.NewV4(),
	Email:    "testdeveloper2@testalm.io",
	FullName: "Test Developer 2",
}

// TestIdentity only creates in memory obj for testing purposes
var TestIdentity = account.Identity{
	ID:       uuid.NewV4(),
	Username: "TestDeveloper",
	User:     TestUser,
}

// TestObserverIdentity only creates in memory obj for testing purposes
var TestObserverIdentity = account.Identity{
	ID:       uuid.NewV4(),
	Username: "TestObserver",
	User:     TestUser,
}

// TestIdentity2 only creates in memory obj for testing purposes
var TestIdentity2 = account.Identity{
	ID:       uuid.NewV4(),
	Username: "TestDeveloper2",
	User:     TestUser2,
}

// CreateTestIdentity creates an identity with the given `username` in the database. For testing purpose only.
func CreateTestIdentity(db *gorm.DB, username, providerType string) (account.Identity, error) {
	identityRepository := account.NewIdentityRepository(db)
	testIdentity := account.Identity{
		Username:     username,
		ProviderType: providerType,
	}
	err := models.Transactional(db, func(tx *gorm.DB) error {
		return identityRepository.Create(context.Background(), &testIdentity)
	})
	log.Logger().Infoln("Created identity with id=", testIdentity.ID.String())
	return testIdentity, err
}
