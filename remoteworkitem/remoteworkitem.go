package remoteworkitem

import (
	"encoding/json"

	"github.com/almighty/almighty-core/app"
)

const (
	SystemRemoteItemId = "system.remote_item_id"
	SystemTitle        = "system.title"
	SystemDescription  = "system.description"
	SystemStatus       = "system.status"
	SystemAssignee     = "system.assignee"
	SystemCreator      = "system.creator"

	GithubTitle       = "title"
	GithubDescription = "body"
	GithubState       = "state"
	GithubId          = "id"
	GithubCreator     = "user.login"
	GithubAssignee    = "assignee.login"

	JiraTitle    = "fields.summary"
	JiraBody     = "fields.description"
	JiraState    = "fields.status.name"
	JiraId       = "self"
	JiraCreator  = "fields.creator.key"
	JiraAssignee = "fields.assignee"

	ProviderGithub = "github"
	ProviderJira   = "jira"
)

var WorkItemKeyMaps = map[string]WorkItemMap{
	ProviderGithub: WorkItemMap{
		AttributeExpression(GithubTitle):       SystemTitle,
		AttributeExpression(GithubDescription): SystemDescription,
		AttributeExpression(GithubState):       SystemStatus,
		AttributeExpression(GithubId):          SystemRemoteItemId,
		AttributeExpression(GithubCreator):     SystemCreator,
		AttributeExpression(GithubAssignee):    SystemAssignee,
	},
	ProviderJira: WorkItemMap{
		AttributeExpression(JiraTitle):    SystemTitle,
		AttributeExpression(JiraBody):     SystemDescription,
		AttributeExpression(JiraState):    SystemStatus,
		AttributeExpression(JiraId):       SystemRemoteItemId,
		AttributeExpression(JiraCreator):  SystemCreator,
		AttributeExpression(JiraAssignee): SystemAssignee,
	},
}

type WorkItemMap map[AttributeExpression]string

// AttributeExpression represents a commonly understood String format for a target path
type AttributeExpression string

// AttributeAccesor defines the interface between a RemoteWorkItem and the Mapper
type AttributeAccesor interface {
	// Get returns the value based on a commonly understood attribute expression
	Get(field AttributeExpression) interface{}
}

// RemoteWorkItem is the Database stored TrackerItem
type RemoteWorkItem struct {
	ID      string
	Content []byte
}

var RemoteWorkItemImplRegistry = map[string]func(RemoteWorkItem) (AttributeAccesor, error){
	ProviderGithub: NewGitHubRemoteWorkItem,
	ProviderJira:   NewJiraRemoteWorkItem,
}

// GitHubRemoteWorkItem knows how to implement a FieldAccessor on a GitHub Issue JSON struct
type GitHubRemoteWorkItem struct {
	issue map[string]interface{}
}

// NewGitHubRemoteWorkItem creates a new Decoded AttributeAccessor for a GitHub Issue
func NewGitHubRemoteWorkItem(item RemoteWorkItem) (AttributeAccesor, error) {
	var j map[string]interface{}
	err := json.Unmarshal(item.Content, &j)
	if err != nil {
		return nil, err
	}
	j = Flatten(j)
	return GitHubRemoteWorkItem{issue: j}, nil
}

func (gh GitHubRemoteWorkItem) Get(field AttributeExpression) interface{} {
	return gh.issue[string(field)]
}

// JiraRemoteWorkItem knows how to implement a FieldAccessor on a Jira Issue JSON struct
type JiraRemoteWorkItem struct {
	issue map[string]interface{}
}

// NewJiraRemoteWorkItem creates a new Decoded AttributeAccessor for a GitHub Issue
func NewJiraRemoteWorkItem(item RemoteWorkItem) (AttributeAccesor, error) {
	var j map[string]interface{}
	err := json.Unmarshal(item.Content, &j)
	if err != nil {
		return nil, err
	}
	// TODO for sbose: Flatten !
	j = Flatten(j)
	return JiraRemoteWorkItem{issue: j}, nil
}

func (jira JiraRemoteWorkItem) Get(field AttributeExpression) interface{} {
	return jira.issue[string(field)]
}

// Map maps the remote WorkItem to a local WorkItem
func Map(item AttributeAccesor, mapping WorkItemMap) (app.WorkItem, error) {
	workItem := app.WorkItem{Fields: make(map[string]interface{})}
	for from, to := range mapping {
		workItem.Fields[to] = item.Get(from)
	}
	return workItem, nil
}
