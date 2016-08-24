// +build unit

package models

import (
	"encoding/json"
	"io/ioutil"
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
	t.Log("Test remoteWorkItem", remoteWorkItem)
	t.Log("Test workItemMap ", workItemMap)

	// TODO: Add error handling
	workItem := remoteWorkItem.MapRemote(workItemMap, workItemType)

	// TODO: compare the workItem fields with those of the remoteWorkItem

	t.Log("Test workItem ", workItem)
}

func TestMapGithub(t *testing.T) {
	url := "https://api.github.com/repos/almighty/almighty-core/issues/131"
	response, err := http.Get(url)

	if err != nil {
		t.Error("Error fetching github api data ")
	}

	defer response.Body.Close()
	responseData, err := ioutil.ReadAll(response.Body)

	// Convert the json string to a json object
	var responseJson map[string]interface{}
	json.Unmarshal(responseData, &responseJson)

	//t.Log(responseJson["title"], responseJson["body"], responseJson["state"])

	githubRemoteWorkItem := GithubRemoteWorkItem{
		RemoteWorkItem{
			Fields: make(map[string]interface{}),
		},
	}
	githubRemoteWorkItem.Fields = responseJson

	/*
		            Create the map which would be used to map keys from Github to local WI

					 * The key refers to the key in the Gitub issue json object
				     * The Value refers to the key that must appear in the WorkItem.Fields
	*/

	workItemMap := githubRemoteWorkItem.GetWorkItemKeyMap()

	workItemType := wellKnown["1"] // Will remove this and add an actual Type.

	// If the data is not mappable as is : then this method will do it for you.
	githubRemoteWorkItem.Flatten()

	// This is where the real action takes place.
	// TODO: Improve Error handling.
	workItem := githubRemoteWorkItem.MapRemote(workItemMap, workItemType)

	// For now, just printed the generated WorkItem.
	// This has to be replaced with Table-driven tests and assert statement.
	t.Log(workItem)

	/*
				       the output of the above logger :

					   {0  1 0
		                   map[
		                        status:open
		                        remote_issue_id:1.71624394e+08
		                        description:related https://trello.com/c/YNeXoM2R/103-create-remoteissue-to-workitemtype-mapping-model
		                        title:As a user I should be able to Map the data in a Remote Issue into a WorkItem Type
		                    ]
		                }

	*/
}
