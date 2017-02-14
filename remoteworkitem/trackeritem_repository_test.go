package remoteworkitem

import (
	"testing"

	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// a normal test function that will kick off TestSuiteTrackeItemRepository
func TestSuiteTrackeItemRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TrackerWorkItemsSuite{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

// ========== TrackeItemRepositorySuite struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
type TrackeItemRepositorySuite struct {
	gormsupport.DBTestSuite
	clean        func()
	tracker      Tracker
	trackerQuery TrackerQuery
}

func (s *TrackeItemRepositorySuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	// Setting up the dependent tracker query and tracker data in the Database
	s.tracker = Tracker{URL: "https://api.github.com/", Type: ProviderGithub}
	s.trackerQuery = TrackerQuery{Query: "some random query", Schedule: "0 0 0 * * *", TrackerID: s.tracker.ID}
	s.T().Log("Created Tracker Query and Tracker")
}

func (s *TrackeItemRepositorySuite) TearDownTest() {
	s.clean()
}

var GitIssueWithAssignee = "http://api.github.com/repos/almighty-test/almighty-test-unit/issues/2"

func (s *TrackeItemRepositorySuite) TestConvertNewWorkItem() {
	s.T().Log("Scenario 1 : Scenario 1: Adding a work item which wasn't present.")
	// given
	remoteItemData := TrackerItemContent{
		Content: []byte(`
				{
					"title": "linking",
					"url": "http://github.com/sbose/api/testonly/1",
					"state": "closed",
					"body": "body of issue",
					"user": {
						"login": "sbose78",
						"url": "https://api.github.com/users/sbose78"
					},
					"assignee": {
						"login": "pranav",
						"url": "https://api.github.com/users/pranav"
					}
				}`),
		ID: "http://github.com/sbose/api/testonly/1",
	}
	// when
	workItem, err := convert(s.DB, int(s.trackerQuery.ID), remoteItemData, ProviderGithub)
	// then
	require.Nil(s.T(), err)
	require.NotNil(s.T(), workItem.Fields)
	assert.Equal(s.T(), "linking", workItem.Fields[workitem.SystemTitle])
	assert.Equal(s.T(), "sbose78", workItem.Fields[workitem.SystemCreator])
	assert.Equal(s.T(), "pranav", workItem.Fields[workitem.SystemAssignees].([]interface{})[0])
	assert.Equal(s.T(), "closed", workItem.Fields[workitem.SystemState])
	require.NotNil(s.T(), workItem.Fields[workitem.SystemDescription])
	description := workItem.Fields[workitem.SystemDescription].(rendering.MarkupContent)
	assert.Equal(s.T(), "body of issue", description.Content)
	assert.Equal(s.T(), rendering.SystemMarkupMarkdown, description.Markup)
}

func (s *TrackeItemRepositorySuite) TestConvertExistingWorkItem() {
	s.T().Log("Adding a work item which wasn't present.")
	// given
	remoteItemData := TrackerItemContent{
		// content is already flattened
		Content: []byte(`
			{
				"title": "linking",
				"url": "http://github.com/sbose/api/testonly/1",
				"state": "closed",
				"body": "body of issue",
				"user.login": "sbose78",
				"user.url": "https://api.github.com/users/sbose78",
				"assignee.login": "pranav",
				"assignee.url": "https://api.github.com/users/pranav"
			}`),
		ID: "http://github.com/sbose/api/testonly/1",
	}
	// when
	workItem, err := convert(s.DB, int(s.trackerQuery.ID), remoteItemData, ProviderGithub)
	// then
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "linking", workItem.Fields[workitem.SystemTitle])
	assert.Equal(s.T(), "sbose78", workItem.Fields[workitem.SystemCreator])
	assert.Equal(s.T(), "pranav", workItem.Fields[workitem.SystemAssignees].([]interface{})[0])
	assert.Equal(s.T(), "closed", workItem.Fields[workitem.SystemState])
	// given
	s.T().Log("Updating the existing work item when it's reimported.")
	remoteItemDataUpdated := TrackerItemContent{
		// content is already flattened
		Content: []byte(`
			{
				"title": "linking-updated",
				"url": "http://github.com/api/testonly/1",
				"state": "closed",
				"body": "body of issue",
				"user.login": "sbose78",
				"user.url": "https://api.github.com/users/sbose78",
				"assignee.login": "pranav",
				"assignee.url": "https://api.github.com/users/pranav"

			}`),
		ID: "http://github.com/sbose/api/testonly/1",
	}
	// when
	workItemUpdated, err := convert(s.DB, int(s.trackerQuery.ID), remoteItemDataUpdated, ProviderGithub)
	// then
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "linking-updated", workItemUpdated.Fields[workitem.SystemTitle])
	assert.Equal(s.T(), "sbose78", workItemUpdated.Fields[workitem.SystemCreator])
	assert.Equal(s.T(), "pranav", workItemUpdated.Fields[workitem.SystemAssignees].([]interface{})[0])
	assert.Equal(s.T(), "closed", workItemUpdated.Fields[workitem.SystemState])
}

func (s *TrackeItemRepositorySuite) TestConvertGithubIssue() {
	// given
	s.T().Log("Scenario 3 : Mapping and persisting a Github issue")
	content, err := test.LoadTestData("github_issue_mapping.json", func() ([]byte, error) {
		return provideRemoteData(GitIssueWithAssignee)
	})
	require.Nil(s.T(), err)
	remoteItemDataGithub := TrackerItemContent{
		Content: content[:],
		ID:      GitIssueWithAssignee, // GH issue url
	}
	// when
	workItemGithub, err := convert(s.DB, int(s.trackerQuery.ID), remoteItemDataGithub, ProviderGithub)
	// then
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "map flatten : test case : with assignee", workItemGithub.Fields[workitem.SystemTitle])
	assert.Equal(s.T(), "sbose78", workItemGithub.Fields[workitem.SystemCreator])
	assert.Equal(s.T(), "sbose78", workItemGithub.Fields[workitem.SystemAssignees].([]interface{})[0])
	assert.Equal(s.T(), "open", workItemGithub.Fields[workitem.SystemState])

}
