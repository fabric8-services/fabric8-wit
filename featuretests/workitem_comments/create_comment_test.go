package workitem_comments

import (
	"github.com/DATA-DOG/godog"
	"testing"
	"os"
)

func FeatureContext(s *godog.Suite) {
	commentCtx := CommentContext{identityHelper: IdentityHelper{}, api:Api{}}

	s.BeforeScenario(commentCtx.Reset)
	s.Step(`^an existing space,$`, commentCtx.anExistingSpace)
	s.Step(`^a user with permissions to comment on work items,$`, commentCtx.aUserWithPermissions)
	s.Step(`^an existing work item exists in the space$`, commentCtx.anExistingWorkItemExistsInTheProject)
	s.Step(`^the user adds a plain text comment to the existing work item,$`, commentCtx.theUserAddsAPlainTextCommentToTheExistingWorkItem)
	s.Step(`^a new comment should be appended against the work item$`, commentCtx.aNewCommentShouldBeAppendedAgainstTheWorkItem)
	s.Step(`^the creator of the comment must be the said user\.$`, commentCtx.theCreatorOfTheCommentMustBeTheSaidUser)
	s.Step(`^an existing work item exists in the space in a closed state$`, commentCtx.anExistingWorkItemExistsInTheProjectInAClosedState)
}

func TestMain(m *testing.M) {
	status := godog.RunWithOptions("godogs", func(s *godog.Suite) {
		FeatureContext(s)
	}, godog.Options{
		Format: "progress",
		Paths:  []string{"../../features/workitem_comments"},
	})

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}
