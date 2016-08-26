package models

import (
	"encoding/json"
)

const (
	SystemRemoteIssueId = "system.remote_issue_id"
	SystemTitle         = "system.title"
	SystemDescription   = "system.description"
	SystemStatus        = "system.status"
	ProviderGithub      = "github"
)

var workItemKeyMaps = map[string]WorkItemMap{
	ProviderGithub: WorkItemMap{
		AttributeExpression("title"): SystemTitle,
		AttributeExpression("body"):  SystemDescription,
		AttributeExpression("state"): SystemStatus,
		AttributeExpression("id"):    SystemRemoteIssueId,
	},
}

type WorkItemMap map[AttributeExpression]string

// AttributeExpression represents a commonly understood String format for a target path
type AttributeExpression string

// AttributeAccesor defines the interface between a RemoteWorkItem and the Mapper
type AttributeAccesor interface {
	// Get returns the value based on a commonly understood field expression
	Get(field AttributeExpression) interface{}
}

// RemoteWorkItem is the Database stored TrackerItem
type RemoteWorkItem struct {
	ID      string
	Content []byte
}

// GitHubRemoteWorkItem knows how to implement a FieldAccessor on a GitHub Issue JSON struct
type GitHubRemoteWorkItem struct {
	issue map[string]interface{}
}

// NewGitHubRemoteWorkItem creates a new Decoded FieldAccessor for a GitHub Issue
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
func Map(item AttributeAccesor, mapping WorkItemMap) (WorkItem, error) {
	workItem := WorkItem{Fields: Fields{}}
	for from, to := range mapping {
		workItem.Fields[to] = item.Get(from)
	}
	return workItem, nil
}
