package iterations

import (
	"os"
	"testing"

	"github.com/DATA-DOG/godog"
)

func FeatureContext(s *godog.Suite) {
	iterationCtx := IterationContext{identityHelper: IdentityHelper{}, api: API{}}

	s.BeforeScenario(iterationCtx.Reset)
	s.Step(`^an existing space,$`, iterationCtx.anExistingSpace)
	s.Step(`^a user with permissions to create iterations in a space,$`, iterationCtx.aUserWithPermissions)
	s.Step(`^the user creates a new iteration with start date "([^"]*)" and end date "([^"]*)"$`, iterationCtx.theUserCreatesANewIterationWithStartDateAndEndDate)
	s.Step(`^a new iteration should be created\.$`, iterationCtx.aNewIterationShouldBeCreated)
}

func TestMain(m *testing.M) {
	status := godog.RunWithOptions("godogs", func(s *godog.Suite) {
		FeatureContext(s)
	}, godog.Options{
		Format: "pretty",
		Paths:  []string{"../../features/iterations"},
		Tags:   "~@undone",
	})

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}
