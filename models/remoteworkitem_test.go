// +build unit

package models

import "testing"

func TestMap(t *testing.T) {

	// This is the Type to be converted to
	workItemType := wellKnown["1"] // Will remove this and add an actual Type.

	remoteWorkItem := RemoteWorkItem{
		Fields: map[string]interface{}{
			"id":          "3",
			"description": "I am a remote issue's description in Github",
			"title":       "I am the title of the issue",
		},
	}

	workItemMap := WorkItemMap{
		"id":          "remote_issue_id",
		"description": "description",
		"title":       "title",
	}
	t.Log("Test remoteWorkItem", remoteWorkItem)
	t.Log("Test workItemMap ", workItemMap)
	workItem := remoteWorkItem.MapRemote(workItemMap, workItemType)

	// TODO: Add code to compare the workItem fields with those of the remoteWorkItem

	t.Log("Test workItem", workItem)
}
