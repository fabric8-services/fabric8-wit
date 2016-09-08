package remoteworkitem

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/test"
)

const (
	GithubIssueWithAssignee    = "http://api.github.com/repos/almighty-test/almighty-test-unit/issues/2"
	GithubIssueWithoutAssignee = "https://api.github.com/repos/almighty-test/almighty-test-unit/issues/1"
	JiraIssueWithAssignee      = "http://jira.atlassian.com/rest/api/latest/issue/JRA-9"
	JiraIssueWithoutAssignee   = "http://jira.atlassian.com/rest/api/latest/issue/JRA-10"
)

func provideRemoteGithubData() ([]byte, error) {
	url := GithubIssueWithAssignee
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
	url := JiraIssueWithAssignee
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
	resource.Require(t, resource.UnitTest)

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
	resource.Require(t, resource.UnitTest)

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
		t.Log("Mapping ", localWorkItemKey)
		_, ok := workItem.Fields[localWorkItemKey]
		if ok == false {
			t.Errorf("%s not mapped", localWorkItemKey)
		} else {
			t.Log("Mapped to ", workItem.Fields[localWorkItemKey])
		}
	}
}

func TestJiraIssueMapping(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	content, err := test.LoadTestData("jira_issue_mapping.json", provideRemoteJiraData)
	if err != nil {
		t.Fatal(err)
	}

	workItemMap := WorkItemKeyMaps[ProviderJira]
	remoteWorkItem := RemoteWorkItem{ID: "xyz", Content: []byte(content)}

	remoteWorkItemImpl := RemoteWorkItemImplRegistry[ProviderJira]
	ji, err := remoteWorkItemImpl(remoteWorkItem)

	if err != nil {
		t.Fatal(err)
	}
	workItem, err := Map(ji, workItemMap)
	if err != nil {
		t.Fatal(err)
	}

	for _, localWorkItemKey := range workItemMap {
		t.Log("Mapping ", localWorkItemKey)
		_, ok := workItem.Fields[localWorkItemKey]
		if ok == false {
			t.Errorf("%s not mapped", localWorkItemKey)
		} else {
			t.Log("Mapped to ", workItem.Fields[localWorkItemKey])
		}
	}
}

func TestFlattenGithubResponseMap(t *testing.T) {
	resource.Require(t, resource.UnitTest)
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
	for k := range githubKeyMap {
		_, ok := OneLevelMap[string(k)]
		if ok == false {
			// GithubAssignee is an optional field so that being absent is fine
			t.Error("Could not access the following key from the flattened map ", k)
		}
	}
}

func TestFlattenJiraResponseMap(t *testing.T) {
	resource.Require(t, resource.UnitTest)
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
	for k := range jiraKeyMap {
		_, ok := OneLevelMap[string(k)]
		if ok == false && string(k) != JiraAssignee {
			// JiraAssignee is an optional field so that being absent is fine.
			t.Error("Could not access the following key from the flattened map ", k)
		}
	}
}
