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
	ProviderGithub     = "github"
)

var WorkItemKeyMaps = map[string]WorkItemMap{
	ProviderGithub: WorkItemMap{
		AttributeExpression("title"): SystemTitle,
		AttributeExpression("body"):  SystemDescription,
		AttributeExpression("state"): SystemStatus,
		AttributeExpression("id"):    SystemRemoteItemId,
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
	return GitHubRemoteWorkItem{issue: j}, nil
}

func (gh GitHubRemoteWorkItem) Get(field AttributeExpression) interface{} {
	return gh.issue[string(field)]
}

// Map maps the remote WorkItem to a local WorkItem
func Map(item AttributeAccesor, mapping WorkItemMap) (app.WorkItem, error) {
	workItem := app.WorkItem{Fields: make(map[string]interface{})}
	for from, to := range mapping {
		workItem.Fields[to] = item.Get(from)
	}
	return workItem, nil
}
