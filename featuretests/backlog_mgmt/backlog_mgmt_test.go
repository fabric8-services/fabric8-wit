package backlog_mgmt

import (
	"github.com/DATA-DOG/godog"
	"testing"
	"os"
)

func FeatureContext(s *godog.Suite) {
	backlogCtx := BacklogContext{identityHelper: IdentityHelper{}, api:Api{}}

	s.BeforeScenario(backlogCtx.Reset)
	s.Step(`^an existing space,$`, backlogCtx.anExistingSpace)
	s.Step(`^a user with permissions to add items to backlog,$`, backlogCtx.aUserWithPermissions)
	s.Step(`^the user adds an item to the backlog with title and description,$`, backlogCtx.theUserAddsAnItemToTheBacklogWithTitleAndDescription)
	s.Step(`^a new work item with a space-unique ID should be created in the backlog$`, backlogCtx.aNewWorkItemShouldBeCreatedInTheBacklog)
	s.Step(`^the creator of the work item must be the said user\.$`, backlogCtx.theCreatorOfTheWorkItemMustBeTheSaidUser)
}

func TestMain(m *testing.M) {
	status := godog.RunWithOptions("godogs", func(s *godog.Suite) {
		FeatureContext(s)
	}, godog.Options{
		Format: "progress",
		Paths:  []string{"../../features/backlog_mgmt"},
	})

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}