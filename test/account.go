package test

import (
	"github.com/almighty/almighty-core/account"
	uuid "github.com/satori/go.uuid"
)

// TestIdentity only creates in memory obj for testing purposes
var TestIdentity = account.Identity{
	ID:       uuid.NewV4(),
	FullName: "Test Developer Identity",
}

// TestUser only creates in memory obj for testing purposes
var TestUser = account.User{
	ID:       uuid.NewV4(),
	Email:    "testdeveloper@testalm.io",
	Identity: TestIdentity,
}
