package models

import (
	"fmt"
	"strconv"
)

const (
	ProviderGithub string = "github"
)

// RemoteWorkItem represents a work item as it is stored in the database
type RemoteWorkItem struct {
	Fields map[string]interface{}
}

// WorkItemMap : holds the mapping between Remote key and local key in a WI
type WorkItemMap map[string]interface{}

// RemoteWorkItemSanitizer makes sure that the
// Response from Provider API is sanitized into a
// good looking flat 1-level dictionary
type RemoteWorkItemMapper interface {
	Flatten()
}

// RemoteWorkItemRepository handles the stuff we need to persist w.r.t to RemoteWorkItems
type RemoteWorkItemRepository struct {
	mappings map[string]WorkItemMap
}

// GetWorkItemKeyMap returns a static map based on the provider & WorkItemType.
// This code will be expanded to support different combinations of provider+WorkItemType
func (repo *RemoteWorkItemRepository) GetWorkItemKeyMap(provider string, workItemType *WorkItemType) WorkItemMap {
	return repo.mappings[provider]
}

func NewRemoteWorkItemRepository() *RemoteWorkItemRepository {
	return &RemoteWorkItemRepository{
		// Statically maintain mappings for now.
		// In future, they should be moved to the database.
		// for easier configurability and patching.
		mappings: map[string]WorkItemMap{
			ProviderGithub: WorkItemMap{
				"id":    "system.remote_issue_id",
				"body":  "system.description",
				"title": "system.title",
				"state": "system.status",
			},
		},
	}
}

// GithubRemoteWorkItem is a derivative of RemoteWorkItem
type GithubRemoteWorkItem struct {
	RemoteWorkItem
}

/*
Response from Provider API is sanitized into a good looking flat 1-level
dictionary, Example:

{
	"meta":
	{
		"title":"Some title",
		"description":"some description",
	}
}

would be into

{
	"meta.title":"some title",
	"meta.description":"some description",
}

Flatten sanitizes the api response from Github into a flat dict

*/
func (rwi *GithubRemoteWorkItem) Flatten() {
	fmt.Println("Flattening Github")
}

// MapRemote maps RemoteWorkItem to WorkItem
func (rwi *RemoteWorkItem) MapRemote(wiMap WorkItemMap, wiType *WorkItemType) WorkItem {
	workItem := WorkItem{
		Type:   strconv.FormatUint(wiType.ID, 10),
		Fields: make(map[string]interface{}),
	}

	for fromKey, toKey := range wiMap {
		workItem.Fields[toKey.(string)] = rwi.Fields[fromKey]
	}
	return workItem
}
