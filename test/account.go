package test

import (
	"github.com/almighty/almighty-core/account"
	uuid "github.com/satori/go.uuid"
)

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

// TestUser only creates in memory obj for testing purposes
var TestUser = account.User{
	ID:       uuid.NewV4(),
	Email:    "testdeveloper@testalm.io",
	FullName: "Test Developer",
}

// TestIdentity2 only creates in memory obj for testing purposes
var TestIdentity2 = account.Identity{
	ID:       uuid.NewV4(),
	FullName: "Test Developer Identity 2",
}

// TestUser2 only creates in memory obj for testing purposes.
// This TestUser2 can be used to verify that some entity created by TestUser
// can be later updated or deleted (or not) by another user.
var TestUser2 = account.User{
	ID:       uuid.NewV4(),
	Email:    "testdeveloper2@testalm.io",
	Identity: TestIdentity2,
}
