// +build unit

package remoteworkitem

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/almighty/almighty-core/test"
)

func provideRemoteGithubData() ([]byte, error) {
	url := "https://api.github.com/repos/almighty/almighty-core/issues/131"
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return responseData, nil
}

func TestWorkItemMapping(t *testing.T) {

	workItemMap := WorkItemMap{
		AttributeExpression("title"): "system.title",
	}
	remoteWorkItem := RemoteWorkItem{ID: "xyz", Content: []byte(`{"title":"abc"}`)}

	remoteWorkItemImpl := RemoteWorkItemImplRegistry[ProviderGithub]
	gh, err := remoteWorkItemImpl(remoteWorkItem)
	if err != nil {
		t.Fatal(err)
	}
	workItem, err := Map(gh, workItemMap)
	if err != nil {
		t.Fatal(err)
	}

	if workItem.Fields["system.title"] == nil {
		t.Error("system.title not mapped")
	}

	t.Log(workItem)
}

func TestGitHubIssueMapping(t *testing.T) {

	content, err := test.LoadTestData("github_issue_mapping.json", provideRemoteGithubData)
	if err != nil {
		t.Fatal(err)
	}

	workItemMap := WorkItemKeyMaps[ProviderGithub]
	remoteWorkItem := RemoteWorkItem{ID: "xyz", Content: []byte(content)}

	remoteWorkItemImpl := RemoteWorkItemImplRegistry[ProviderGithub]
	gh, err := remoteWorkItemImpl(remoteWorkItem)

	if err != nil {
		t.Fatal(err)
	}
	workItem, err := Map(gh, workItemMap)
	if err != nil {
		t.Fatal(err)
	}

	for _, localWorkItemKey := range workItemMap {
		t.Log("Mapped ", localWorkItemKey)
		if workItem.Fields[localWorkItemKey] == nil {
			t.Error(fmt.Sprintf("%s not mapped", localWorkItemKey))
		}
	}
}
