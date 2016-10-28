package remoteworkitem

import (
	"encoding/json"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
)

// List of supported attributes
const (
	SystemRemoteItemID = "system.remote_item_id"
	SystemTitle        = "system.title"
	SystemDescription  = "system.description"
	SystemState        = "system.state"
	SystemAssignee     = "system.assignee"
	SystemCreator      = "system.creator"

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
		AttributeMapper{AttributeExpression(GithubTitle), StringConverter{}}:       models.SystemTitle,
		AttributeMapper{AttributeExpression(GithubDescription), StringConverter{}}: models.SystemDescription,
		AttributeMapper{AttributeExpression(GithubState), GithubStateConverter{}}:  models.SystemState,
		AttributeMapper{AttributeExpression(GithubID), StringConverter{}}:          models.SystemRemoteItemID,
		AttributeMapper{AttributeExpression(GithubCreator), StringConverter{}}:     models.SystemCreator,
		AttributeMapper{AttributeExpression(GithubAssignee), StringConverter{}}:    models.SystemAssignee,
	},
	ProviderJira: WorkItemMap{
		AttributeMapper{AttributeExpression(JiraTitle), StringConverter{}}:      models.SystemTitle,
		AttributeMapper{AttributeExpression(JiraBody), StringConverter{}}:       models.SystemDescription,
		AttributeMapper{AttributeExpression(JiraState), GithubStateConverter{}}: models.SystemState,
		AttributeMapper{AttributeExpression(JiraID), StringConverter{}}:         models.SystemRemoteItemID,
		AttributeMapper{AttributeExpression(JiraCreator), StringConverter{}}:    models.SystemCreator,
		AttributeMapper{AttributeExpression(JiraAssignee), StringConverter{}}:   models.SystemAssignee,
	},
}

type AttributeConverter interface {
	Convert(interface{}) (interface{}, error)
}

type StateConverter interface{}

type StringConverter struct{}

type GithubStateConverter struct{}

// Convert method map the external tracker item to ALM WorkItem
func (sc StringConverter) Convert(value interface{}) (interface{}, error) {
	return value, nil
}

func (ghc GithubStateConverter) Convert(value interface{}) (interface{}, error) {
	if value.(string) == "closed" {
		value = "closed"
	} else if value.(string) == "open" {
		value = "open"
	}
	return value, nil
}

type AttributeMapper struct {
	expression         AttributeExpression
	attributeConverter AttributeConverter
}

// WorkItemMap will define mappings between remote<->internal attribute
type WorkItemMap map[AttributeMapper]string

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
// and it should also know how to convert a value in remote work item for use in local WI
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

// Map maps the remote WorkItem to a local WorkItem
func Map(item AttributeAccessor, mapping WorkItemMap) (app.WorkItem, error) {
	workItem := app.WorkItem{Fields: make(map[string]interface{})}
	for from, to := range mapping {
		originalValue := item.Get(from.expression)
		convertedValue, err := from.attributeConverter.Convert(originalValue)
		if err == nil {
			workItem.Fields[to] = convertedValue
		}
	}
	return workItem, nil
}
