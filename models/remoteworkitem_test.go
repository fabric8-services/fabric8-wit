// +build unit

package models

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

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

	// TODO: Add error handling
	workItem := remoteWorkItem.MapRemote(workItemMap, workItemType)

	// Asserting the mappingw.
	for mapFrom, mapTo := range workItemMap {
		valueInRemoteWorkItem := remoteWorkItem.Fields[mapFrom]
		valueInLocalWorkItem := workItem.Fields[mapTo]
		if valueInLocalWorkItem != valueInRemoteWorkItem {
			t.Error(fmt.Sprintf("Incorrect mapping of %s in remote WI to %s in local WI. Expected \"%s\" but found \"%s\"", mapFrom, mapTo, valueInRemoteWorkItem, valueInLocalWorkItem))
		}
	}
}

func TestMapGithub(t *testing.T) {
	url := "https://api.github.com/repos/almighty/almighty-core/issues/131"
	response, err := http.Get(url)

	if err != nil {
		t.Error(err)
	}

	defer response.Body.Close()

	var responseJson map[string]interface{}

	responseDecoder := json.NewDecoder(response.Body)
	err = responseDecoder.Decode(&responseJson)

	if err != nil {
		t.Error(err)
	}

	//t.Log(responseJson["title"], responseJson["body"], responseJson["state"])

	remoteWorkItem := RemoteWorkItem{
		Fields: responseJson,
	}

	workItemType := wellKnown["1"] // Will remove this and add an actual Type.

	workItemMap := NewRemoteWorkItemRepository().GetWorkItemKeyMap(ProviderGithub, workItemType)

	// This is where the real action takes place.
	// TODO: Improve Error handling.
	workItem := remoteWorkItem.MapRemote(workItemMap, workItemType)

	// Asserting the mappingw.
	for mapFrom, mapTo := range workItemMap {
		valueInRemoteWorkItem := remoteWorkItem.Fields[mapFrom]
		valueInLocalWorkItem := workItem.Fields[mapTo]
		if valueInLocalWorkItem != valueInRemoteWorkItem {
			t.Error(fmt.Sprintf("Incorrect mapping of %s in remote WI to %s in local WI. Expected \"%s\" but found \"%s\"", mapFrom, mapTo, valueInRemoteWorkItem, valueInLocalWorkItem))
		}
	}
}
