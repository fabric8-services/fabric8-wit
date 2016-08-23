// +build unit

package models

import "testing"

func TestMap(t *testing.T) {

	// This is the Type to be converted to
	workItemType := wellKnown["1"]  // Will remove this and add an actual Type.

	// remoteWorkItem is fetched from external providers
	remoteWorkItem := RemoteWorkItem{
        Fields : {
            "id" : "1",
            "title":"This is a remote issue title",
            "description":"This is a remote issue description"
        }
    }   

    workItemMap := WorkItemMap{
        "id":"remote_issue_id",
        "description":"description",
        "title":"title"
    }

    workItem,err = remoteWorkItem.MapRemote(workItemMap,workItemType)    

}
