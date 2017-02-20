package remoteworkitem

import (
	"context"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// a normal test function that will kick off TestSuiteTrackerItemRepository
func TestSuiteTrackerItemRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TrackerItemRepositorySuite{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

// ========== TrackeItemRepositorySuite struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
type TrackerItemRepositorySuite struct {
	gormsupport.DBTestSuite
	clean        func()
	trackerQuery TrackerQuery
}

func (s *TrackerItemRepositorySuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	// Setting up the dependent tracker query and tracker data in the Database
	tracker := Tracker{URL: "https://api.github.com/", Type: ProviderGithub}
	s.trackerQuery = TrackerQuery{Query: "some random query", Schedule: "0 0 0 * * *", TrackerID: tracker.ID}
}

func (s *TrackerItemRepositorySuite) createIdentity(username string) account.Identity {
	identityRepo := account.NewIdentityRepository(s.DB)
	identity := account.Identity{
		Username:     username,
		ProfileURL:   "https://api.github.com/users/" + username,
		ProviderType: ProviderGithub,
	}
	err := identityRepo.Create(context.Background(), &identity)
	require.Nil(s.T(), err)
	return identity
}

func (s *TrackerItemRepositorySuite) lookupIdentityByID(id string) account.Identity {
	identityRepo := account.NewIdentityRepository(s.DB)
	identityID, err := uuid.FromString(id)
	require.Nil(s.T(), err)
	identity, err := identityRepo.First(account.IdentityFilterByID(identityID))
	require.Nil(s.T(), err)
	return *identity
}

func (s *TrackerItemRepositorySuite) TearDownTest() {
	s.clean()
}

var GitIssueWithAssignee = "http://api.github.com/repos/almighty-test/almighty-test-unit/issues/2"

func (s *TrackerItemRepositorySuite) TestConvertNewWorkItem() {
	// given
	identity := s.createIdentity("jdoe")
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
						"login": "jdoe",
						"url": "https://api.github.com/users/jdoe"
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
	require.NotEmpty(s.T(), workItem.Fields[workitem.SystemAssignees])
	assert.Equal(s.T(), identity.ID.String(), workItem.Fields[workitem.SystemAssignees].([]interface{})[0])
	assert.Equal(s.T(), "closed", workItem.Fields[workitem.SystemState])
	require.NotNil(s.T(), workItem.Fields[workitem.SystemDescription])
	description := workItem.Fields[workitem.SystemDescription].(rendering.MarkupContent)
	assert.Equal(s.T(), "body of issue", description.Content)
	assert.Equal(s.T(), rendering.SystemMarkupMarkdown, description.Markup)
}

func (s *TrackerItemRepositorySuite) TestConvertNewWorkItemWithUnknownAssignee() {
	// given "jdoe" identity does not exist
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
						"login": "jdoe",
						"url": "https://api.github.com/users/jdoe"
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
	require.NotEmpty(s.T(), workItem.Fields[workitem.SystemAssignees])
	assert.Equal(s.T(), "closed", workItem.Fields[workitem.SystemState])
	require.NotNil(s.T(), workItem.Fields[workitem.SystemDescription])
	description := workItem.Fields[workitem.SystemDescription].(rendering.MarkupContent)
	assert.Equal(s.T(), "body of issue", description.Content)
	assert.Equal(s.T(), rendering.SystemMarkupMarkdown, description.Markup)
	// look-up identity in repository
	identityID := workItem.Fields[workitem.SystemAssignees].([]interface{})[0].(string)
	assert.NotNil(s.T(), s.lookupIdentityByID(identityID))
}

func (s *TrackerItemRepositorySuite) TestConvertNewWorkItemWithNullAssignee() {
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
					"assignee": null
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
	assert.Empty(s.T(), workItem.Fields[workitem.SystemAssignees])
	assert.Equal(s.T(), "closed", workItem.Fields[workitem.SystemState])
	require.NotNil(s.T(), workItem.Fields[workitem.SystemDescription])
	description := workItem.Fields[workitem.SystemDescription].(rendering.MarkupContent)
	assert.Equal(s.T(), "body of issue", description.Content)
	assert.Equal(s.T(), rendering.SystemMarkupMarkdown, description.Markup)
}

func (s *TrackerItemRepositorySuite) TestConvertExistingWorkItem() {
	// given
	identity := s.createIdentity("jdoe")
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
				"assignee.login": "jdoe",
				"assignee.url": "https://api.github.com/users/jdoe"
			}`),
		ID: "http://github.com/sbose/api/testonly/1",
	}
	// when
	workItem, err := convert(s.DB, int(s.trackerQuery.ID), remoteItemData, ProviderGithub)
	// then
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "linking", workItem.Fields[workitem.SystemTitle])
	assert.Equal(s.T(), "sbose78", workItem.Fields[workitem.SystemCreator])
	require.NotEmpty(s.T(), workItem.Fields[workitem.SystemAssignees])
	assert.Equal(s.T(), identity.ID.String(), workItem.Fields[workitem.SystemAssignees].([]interface{})[0])
	assert.Equal(s.T(), "closed", workItem.Fields[workitem.SystemState])
	// given
	s.T().Log("Updating the existing work item when it's reimported.")
	identity = s.createIdentity("pranav")
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
	require.NotEmpty(s.T(), workItemUpdated.Fields[workitem.SystemAssignees])
	assert.Equal(s.T(), identity.ID.String(), workItemUpdated.Fields[workitem.SystemAssignees].([]interface{})[0])
	assert.Equal(s.T(), "closed", workItemUpdated.Fields[workitem.SystemState])
}

func (s *TrackerItemRepositorySuite) TestConvertGithubIssue() {
	// given
	identity := s.createIdentity("sbose78")
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
	assert.Equal(s.T(), identity.ID.String(), workItemGithub.Fields[workitem.SystemAssignees].([]interface{})[0])
	assert.Equal(s.T(), "open", workItemGithub.Fields[workitem.SystemState])

}
