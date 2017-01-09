package spaces

import (
	"github.com/DATA-DOG/godog"
	"testing"
	"os"
)

func FeatureContext(s *godog.Suite) {
	spaceCtx := SpaceContext{identityHelper: IdentityHelper{}, api:Api{}}

	s.BeforeScenario(spaceCtx.Reset)
	s.Step(`^a user with permissions to create spaces,$`, spaceCtx.aUserWithPermissions)
	s.Step(`^the user creates a new space,$`, spaceCtx.theUserCreatesANewSpace)
	s.Step(`^a new space should be created\.$`, spaceCtx.aNewSpaceShouldBeCreated)

}

func TestMain(m *testing.M) {
	status := godog.RunWithOptions("godogs", func(s *godog.Suite) {
		FeatureContext(s)
	}, godog.Options{
		Format: "progress",
		Paths:  []string{"../../features/spaces"},
	})

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}