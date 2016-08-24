package models

import (
	"fmt"
	"strconv"
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
	GetWorkItemKeyMap()
	MapRemote()
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

// GetWorkItemKeyMap returns the dictionary that would map a remote WorkItem to a local WorkItem
func (rwi *GithubRemoteWorkItem) GetWorkItemKeyMap() WorkItemMap {
	workItemMap := WorkItemMap{
		"id":    "remote_issue_id",
		"body":  "description",
		"title": "title",
		"state": "status",
	}
	return workItemMap
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
