package spaces

import (
	"os"
	"testing"

	"github.com/DATA-DOG/godog"
)

func FeatureContext(s *godog.Suite) {
	spaceCtx := SpaceContext{identityHelper: IdentityHelper{}, api: Api{}}

	s.BeforeSuite(spaceCtx.CleanupDatabase)
	s.BeforeScenario(spaceCtx.Reset)
	s.Step(`^a user with permissions to create spaces,$`, spaceCtx.aUserWithPermissions)
	s.Step(`^the user creates a new space "([^"]*)",$`, spaceCtx.theUserCreatesANewSpace)
	s.Step(`^a new space should be created\.$`, spaceCtx.aNewSpaceShouldBeCreated)

	s.Step(`^a space "([^"]*)" already exists with the same user as owner,$`, spaceCtx.aSpaceAlreadyExistsWithTheSameUserAsOwner)
	s.Step(`^a new space should not be created\.$`, spaceCtx.aNewSpaceShouldNotBeCreated)
	s.Step(`^a space "([^"]*)" already exists with a different user as owner,$`, spaceCtx.aSpaceAlreadyExistsWithADifferentUserAsOwner)

}

func TestMain(m *testing.M) {
	status := godog.RunWithOptions("godogs", func(s *godog.Suite) {
		FeatureContext(s)
	}, godog.Options{
		Format: "pretty",
		Paths:  []string{"../../features/spaces"},
		Tags:   "~@undone",
	})

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}
