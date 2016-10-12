package remoteworkitem

import (
	"encoding/json"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
)

// List of supported attributes
const (
	// The keys in the flattened response JSON of a typical Github issue.

	GithubTitle       = "title"
	GithubDescription = "body"
	GithubState       = "state"
	GithubID          = "url"
	GithubCreator     = "user.login"
	GithubAssignee    = "assignee.login"

	// The keys in the flattened response JSON of a typical Jira issue.

	JiraTitle    = "fields.summary"
	JiraBody     = "fields.description"
	JiraState    = "fields.status.name"
	JiraID       = "self"
	JiraCreator  = "fields.creator.key"
	JiraAssignee = "fields.assignee"

	ProviderGithub = "github"
	ProviderJira   = "jira"
)

// WorkItemKeyMaps relate remote attribute keys to internal representation
var WorkItemKeyMaps = map[string]WorkItemMap{
	ProviderGithub: WorkItemMap{
		AttributeExpression(GithubTitle):       models.SystemTitle,
		AttributeExpression(GithubDescription): models.SystemDescription,
		AttributeExpression(GithubState):       models.SystemState,
		AttributeExpression(GithubID):          models.SystemRemoteItemID,
		AttributeExpression(GithubCreator):     models.SystemCreator,
		AttributeExpression(GithubAssignee):    models.SystemAssignee,
	},
	ProviderJira: WorkItemMap{
		AttributeExpression(JiraTitle):    models.SystemTitle,
		AttributeExpression(JiraBody):     models.SystemDescription,
		AttributeExpression(JiraState):    models.SystemState,
		AttributeExpression(JiraID):       models.SystemRemoteItemID,
		AttributeExpression(JiraCreator):  models.SystemCreator,
		AttributeExpression(JiraAssignee): models.SystemAssignee,
	},
}

// WorkItemMap will define mappings between remote<->internal attribute
type WorkItemMap map[AttributeExpression]string

// IssueStateMap will define mapping between tracker item issue state and workitem state
type IssueStateMap map[string]string

// AttributeExpression represents a commonly understood String format for a target path
type AttributeExpression string

// AttributeAccessor defines the interface between a RemoteWorkItem and the Mapper
type AttributeAccessor interface {
	// Get returns the value based on a commonly understood attribute expression
	Get(field AttributeExpression) interface{}
}

// RemoteWorkItemImplRegistry contains all possible providers
var RemoteWorkItemImplRegistry = map[string]func(TrackerItem) (AttributeAccessor, error){
	ProviderGithub: NewGitHubRemoteWorkItem,
	ProviderJira:   NewJiraRemoteWorkItem,
}

// GitHubRemoteWorkItem knows how to implement a FieldAccessor on a GitHub Issue JSON struct
type GitHubRemoteWorkItem struct {
	issue map[string]interface{}
}

// NewGitHubRemoteWorkItem creates a new Decoded AttributeAccessor for a GitHub Issue
func NewGitHubRemoteWorkItem(item TrackerItem) (AttributeAccessor, error) {
	var j map[string]interface{}
	err := json.Unmarshal([]byte(item.Item), &j)
	if err != nil {
		return nil, err
	}
	j = Flatten(j)
	return GitHubRemoteWorkItem{issue: j}, nil
}

// Get attribute from issue map
func (gh GitHubRemoteWorkItem) Get(field AttributeExpression) interface{} {
	return gh.issue[string(field)]
}

// JiraRemoteWorkItem knows how to implement a FieldAccessor on a Jira Issue JSON struct
type JiraRemoteWorkItem struct {
	issue map[string]interface{}
}

// NewJiraRemoteWorkItem creates a new Decoded AttributeAccessor for a GitHub Issue
func NewJiraRemoteWorkItem(item TrackerItem) (AttributeAccessor, error) {
	var j map[string]interface{}
	err := json.Unmarshal([]byte(item.Item), &j)
	if err != nil {
		return nil, err
	}
	j = Flatten(j)
	return JiraRemoteWorkItem{issue: j}, nil
}

// Get attribute from issue map
func (jira JiraRemoteWorkItem) Get(field AttributeExpression) interface{} {
	return jira.issue[string(field)]
}

var issueStateMaps = map[string]string{
	"open":   "open",
	"closed": "closed",
	"merged": "resolved",
}

//Mapping of workitem state
func mapIssueStates(WorkItem app.WorkItem, issuemap IssueStateMap) (app.WorkItem, error) {
	var currentstate string
	for _, _ = range WorkItem.Fields {
		for from, to := range issuemap {
			if from == WorkItem.Fields["system.state"] {
				currentstate = to
			}
		}
		WorkItem.Fields["system.state"] = currentstate
	}
	return WorkItem, nil
}

// Map maps the remote WorkItem to a local WorkItem
func Map(item AttributeAccessor, mapping WorkItemMap) (app.WorkItem, error) {
	workItem := app.WorkItem{Fields: make(map[string]interface{})}
	for from, to := range mapping {
		workItem.Fields[to] = item.Get(from)
	}
	mappedWorkItem, _ := mapIssueStates(workItem, issueStateMaps)
	return mappedWorkItem, nil
}
