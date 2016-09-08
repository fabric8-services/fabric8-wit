package remoteworkitem

import (
	"encoding/json"
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

func provideRemoteJiraData() ([]byte, error) {
	url := "http://jira.atlassian.com/rest/api/latest/issue/JRA-9"
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

func TestFlattenGithubResponseMap(t *testing.T) {
	testString, err := test.LoadTestData("github_issue_mapping.json", provideRemoteGithubData)
	if err != nil {
		t.Fatal(err)
	}
	var nestedMap map[string]interface{}
	err = json.Unmarshal(testString, &nestedMap)

	if err != nil {
		t.Error("Incorrect dataset ", testString)
	}

	OneLevelMap := Flatten(nestedMap)

	githubKeyMap := WorkItemKeyMaps[ProviderGithub]

	// Verifying if the new map is usable.
	for k, _ := range githubKeyMap {
		remoteItemKey, ok := OneLevelMap[string(k)]
		t.Log("Key value pair of remote Item: ", string(k), remoteItemKey)
		if ok == false {
			t.Error("Could not access the following key from the flattened map ", k)
		}
	}
}

func TestFlattenJiraResponseMap(t *testing.T) {
	testString, err := test.LoadTestData("jira_issue_mapping.json", provideRemoteJiraData)
	if err != nil {
		t.Fatal(err)
	}
	var nestedMap map[string]interface{}
	err = json.Unmarshal(testString, &nestedMap)

	if err != nil {
		t.Error("Incorrect dataset ", testString)
	}

	OneLevelMap := Flatten(nestedMap)
	jiraKeyMap := WorkItemKeyMaps[ProviderJira]

	// Verifying if the newly converted map is usable.
	for k, _ := range jiraKeyMap {
		remoteItemKey, ok := OneLevelMap[string(k)]
		t.Log("Key value pair of remote Item: ", string(k), remoteItemKey)
		if ok == false {
			t.Error("Could not access the following key from the flattened map ", k)
		}
	}
}
