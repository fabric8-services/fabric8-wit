package remoteworkitem

import (
	"encoding/json"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/workitem"
	"github.com/pkg/errors"
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
		AttributeMapper{AttributeExpression(GithubTitle), StringConverter{}}:                                            workitem.SystemTitle,
		AttributeMapper{AttributeExpression(GithubDescription), MarkupConverter{markup: workitem.SystemMarkupMarkdown}}: workitem.SystemDescription,
		AttributeMapper{AttributeExpression(GithubState), GithubStateConverter{}}:                                       workitem.SystemState,
		AttributeMapper{AttributeExpression(GithubID), StringConverter{}}:                                               workitem.SystemRemoteItemID,
		AttributeMapper{AttributeExpression(GithubCreator), StringConverter{}}:                                          workitem.SystemCreator,
		AttributeMapper{AttributeExpression(GithubAssignee), ListStringConverter{}}:                                     workitem.SystemAssignees,
	},
	ProviderJira: WorkItemMap{
		AttributeMapper{AttributeExpression(JiraTitle), StringConverter{}}:                                     workitem.SystemTitle,
		AttributeMapper{AttributeExpression(JiraBody), StringConverter{}}:                                      workitem.SystemDescription,
		AttributeMapper{AttributeExpression(JiraBody), MarkupConverter{markup: workitem.SystemMarkupJiraWiki}}: workitem.SystemDescription,
		AttributeMapper{AttributeExpression(JiraState), JiraStateConverter{}}:                                  workitem.SystemState,
		AttributeMapper{AttributeExpression(JiraID), StringConverter{}}:                                        workitem.SystemRemoteItemID,
		AttributeMapper{AttributeExpression(JiraCreator), StringConverter{}}:                                   workitem.SystemCreator,
		AttributeMapper{AttributeExpression(JiraAssignee), ListStringConverter{}}:                              workitem.SystemAssignees,
	},
}

type AttributeConverter interface {
	Convert(interface{}, AttributeAccessor) (interface{}, error)
}

type StateConverter interface{}

type StringConverter struct{}

// MarkupConverter converts to a 'MarkupContent' element with the given 'Markup' value
type MarkupConverter struct {
	markup string
}

type ListStringConverter struct{}

type GithubStateConverter struct{}

type JiraStateConverter struct{}

// Convert method map the external tracker item to ALM WorkItem
func (sc StringConverter) Convert(value interface{}, item AttributeAccessor) (interface{}, error) {
	return value, nil
}

// Convert returns the given `value` if the `item` is not nil`, otherwise returns `nil`
func (converter MarkupConverter) Convert(value interface{}, item AttributeAccessor) (interface{}, error) {
	// return a 'nil' result if the supplied 'value' was nil
	if value == nil {
		return nil, nil
	}
	switch value.(type) {
	case string:
		return workitem.MarkupContent{Content: value.(string), Markup: converter.markup}, nil
	default:
		return nil, errors.Errorf("Unexpected type of value to convert: %T", value)
	}
}

// Convert method map the external tracker item to ALM WorkItem
func (sc ListStringConverter) Convert(value interface{}, item AttributeAccessor) (interface{}, error) {
	return []interface{}{value}, nil
}

func (ghc GithubStateConverter) Convert(value interface{}, item AttributeAccessor) (interface{}, error) {
	if value.(string) == "closed" {
		value = "closed"
	} else if value.(string) == "open" {
		value = "open"
	}
	return value, nil
}

func (jhc JiraStateConverter) Convert(value interface{}, item AttributeAccessor) (interface{}, error) {
	if value.(string) == "closed" {
		value = "closed"
	} else if value.(string) == "open" {
		value = "open"
	} else if value.(string) == "in progress" {
		value = "in progress"
	} else if value.(string) == "resolved" {
		value = "resolved"
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
		convertedValue, err := from.attributeConverter.Convert(originalValue, item)
		if err == nil {
			workItem.Fields[to] = convertedValue
		}
	}
	return workItem, nil
}
