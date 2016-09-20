package remoteworkitem

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/test"
	"github.com/stretchr/testify/assert"
)

const (
	GithubIssueWithAssignee    = "http://api.github.com/repos/almighty-test/almighty-test-unit/issues/2"
	GithubIssueWithoutAssignee = "http://api.github.com/repos/almighty-test/almighty-test-unit/issues/1"
	JiraIssueWithAssignee      = "http://jira.atlassian.com/rest/api/latest/issue/JRA-9"
	JiraIssueWithoutAssignee   = "http://jira.atlassian.com/rest/api/latest/issue/JRA-10"
)

func provideRemoteData(dataUrl string) ([]byte, error) {
	url := dataUrl
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

func provideRemoteGithubDataWithAssignee() ([]byte, error) {
	return provideRemoteData(GithubIssueWithAssignee)
}

func provideRemoteJiraDataWithAssignee() ([]byte, error) {
	return provideRemoteData(JiraIssueWithAssignee)
}

func provideRemoteGithubDataWithoutAssignee() ([]byte, error) {
	return provideRemoteData(GithubIssueWithoutAssignee)
}

func provideRemoteJiraDataWithoutAssignee() ([]byte, error) {
	return provideRemoteData(JiraIssueWithoutAssignee)
}

func TestWorkItemMapping(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	workItemMap := WorkItemMap{
		AttributeExpression("title"): "system.title",
	}
	jsonContent := `{"title":"abc"}`
	remoteTrackerItem := TrackerItem{Item: jsonContent, RemoteItemID: "xyz", BatchID: "bxyz", TrackerQueryID: uint64(0)}

	remoteWorkItemImpl := RemoteWorkItemImplRegistry[ProviderGithub]
	gh, err := remoteWorkItemImpl(remoteTrackerItem)
	if err != nil {
		t.Fatal(err)
	}
	workItem, err := Map(gh, workItemMap)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, workItem.Fields["system.title"], "system.title not mapped")
}

func TestGitHubIssueMapping(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	content, err := test.LoadTestData("github_issue_mapping.json", provideRemoteGithubDataWithAssignee)
	if err != nil {
		t.Fatal(err)
	}

	workItemMap := WorkItemKeyMaps[ProviderGithub]
	remoteTrackerkItem := TrackerItem{Item: string(content[:]), RemoteItemID: "xyz", BatchID: "bxyz", TrackerQueryID: uint64(0)}

	remoteWorkItemImpl := RemoteWorkItemImplRegistry[ProviderGithub]
	gh, err := remoteWorkItemImpl(remoteTrackerkItem)

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
		assert.Equal(t, ok, true, fmt.Sprintf("%s not mapped", localWorkItemKey))
	}
}
func TestJiraIssueMapping(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	content, err := test.LoadTestData("jira_issue_mapping.json", provideRemoteJiraDataWithAssignee)
	if err != nil {
		t.Fatal(err)
	}

	workItemMap := WorkItemKeyMaps[ProviderJira]
	remoteTrackerItem := TrackerItem{Item: string(content[:]), RemoteItemID: "xyz", BatchID: "bxyz", TrackerQueryID: uint64(0)}
	remoteWorkItemImpl := RemoteWorkItemImplRegistry[ProviderJira]
	ji, err := remoteWorkItemImpl(remoteTrackerItem)

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
		assert.Equal(t, ok, true, fmt.Sprintf("%s not mapped", localWorkItemKey))
	}
}

func TestFlattenGithubResponseMap(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	testString, err := test.LoadTestData("github_issue_mapping.json", provideRemoteGithubDataWithAssignee)
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
		assert.Equal(t, ok, true, fmt.Sprintf("Could not access %s from the flattened map ", k))
	}
}

func TestFlattenGithubResponseMapWithoutAssignee(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	testString, err := test.LoadTestData("github_issue_mapping.json", provideRemoteGithubDataWithoutAssignee)
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
		if k == GithubAssignee {
			continue
		}
		assert.Equal(t, ok, true, fmt.Sprintf("Could not access %s from the flattened map ", k))
	}
}

func TestFlattenJiraResponseMap(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	testString, err := test.LoadTestData("jira_issue_mapping.json", provideRemoteJiraDataWithAssignee)
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
		assert.Equal(t, ok, true, fmt.Sprint("Could not access %s from the flattened map ", k))
	}
}

func TestFlattenJiraResponseMapWithoutAssignee(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	testString, err := test.LoadTestData("jira_issue_mapping.json", provideRemoteJiraDataWithoutAssignee)
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
		if k == JiraAssignee {
			continue
		}
		assert.Equal(t, ok, true, fmt.Sprint("Could not access %s from the flattened map ", k))
	}
}

func TestNewGitHubRemoteWorkItem(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	jsonContent := `{"admins":[{"name":"aslak"}],"name":"shoubhik", "assignee":{"fixes": 2, "complete" : true,"foo":[ 1,2,3,4],"1":"sbose","2":"pranav","participants":{"4":"sbose56","5":"sbose78"}},"name":"shoubhik"}`
	remoteTrackerItem := TrackerItem{Item: jsonContent, RemoteItemID: "xyz", BatchID: "bxyz", TrackerQueryID: uint64(0)}

	githubRemoteWorkItem, ok := NewGitHubRemoteWorkItem(remoteTrackerItem)
	assert.Nil(t, ok)
	assert.Equal(t, githubRemoteWorkItem.Get("admins.0.name"), "aslak")
	assert.Equal(t, githubRemoteWorkItem.Get("name"), "shoubhik")
	assert.Equal(t, githubRemoteWorkItem.Get("assignee.complete"), true)
	assert.Equal(t, githubRemoteWorkItem.Get("assignee.participants.4"), "sbose56")

}

func TestNewJiraRemoteWorkItem(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	jsonContent := `{"admins":[{"name":"aslak"}],"name":"shoubhik", "assignee":{"fixes": 2, "complete" : true,"foo":[ 1,2,3,4],"1":"sbose","2":"pranav","participants":{"4":"sbose56","5":"sbose78"}},"name":"shoubhik"}`
	remoteTrackerItem := TrackerItem{Item: jsonContent, RemoteItemID: "xyz", BatchID: "bxyz", TrackerQueryID: uint64(0)}

	jiraRemoteWorkItem, ok := NewJiraRemoteWorkItem(remoteTrackerItem)
	assert.Nil(t, ok)
	assert.Equal(t, jiraRemoteWorkItem.Get("admins.0.name"), "aslak")
	assert.Equal(t, jiraRemoteWorkItem.Get("name"), "shoubhik")
	assert.Equal(t, jiraRemoteWorkItem.Get("assignee.complete"), true)
	assert.Equal(t, jiraRemoteWorkItem.Get("assignee.participants.4"), "sbose56")
}

func TestRemoteWorkItemImplRegistry(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	_, ok := RemoteWorkItemImplRegistry[ProviderGithub]
	assert.Equal(t, ok, true)

	_, ok = RemoteWorkItemImplRegistry[ProviderJira]
	assert.Equal(t, ok, true)

}
