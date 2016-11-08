package remoteworkitem

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/almighty/almighty-core/models"
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

func provideRemoteData(dataURL string) ([]byte, error) {
	response, err := http.Get(dataURL)
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
		AttributeMapper{AttributeExpression("title"), AttributeConverter(StringConverter{})}: models.SystemTitle,
	}
	jsonContent := `{"title":"abc"}`
	remoteTrackerItem := TrackerItem{Item: jsonContent, RemoteItemID: "xyz", TrackerID: uint64(0)}

	remoteWorkItemImpl := RemoteWorkItemImplRegistry[ProviderGithub]
	gh, err := remoteWorkItemImpl(remoteTrackerItem)
	if err != nil {
		t.Fatal(err)
	}
	workItem, err := Map(gh, workItemMap)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, workItem.Fields[models.SystemTitle], fmt.Sprintf("%s not mapped", models.SystemTitle))
}

func TestGitHubIssueMapping(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	type githubData struct {
		inputFile      string
		expectedOutput bool
	}

	var gid = []githubData{
		{"github_issue_mapping.json", true},
	}

	for _, j := range gid {
		content, err := test.LoadTestData(j.inputFile, provideRemoteGithubDataWithAssignee)
		if err != nil {
			t.Fatal(err)
		}

		workItemMap := WorkItemKeyMaps[ProviderGithub]
		remoteTrackerkItem := TrackerItem{Item: string(content[:]), RemoteItemID: "xyz", TrackerID: uint64(0)}

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
			assert.Equal(t, ok, j.expectedOutput, fmt.Sprintf("%s not mapped", localWorkItemKey))
		}
	}
}
func TestJiraIssueMapping(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	type jiradata struct {
		inputFile      string
		expectedOutput bool
	}

	var jir = []jiradata{
		{"jira_issue_mapping.json", true},
	}

	for _, j := range jir {
		content, err := test.LoadTestData(j.inputFile, provideRemoteJiraDataWithAssignee)
		if err != nil {
			t.Fatal(err)
		}

		workItemMap := WorkItemKeyMaps[ProviderJira]
		remoteTrackerItem := TrackerItem{Item: string(content[:]), RemoteItemID: "xyz", TrackerID: uint64(0)}
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
			assert.Equal(t, ok, j.expectedOutput, fmt.Sprintf("%s not mapped", localWorkItemKey))
		}
	}
}

func TestFlattenGithubResponseMap(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	type githubData struct {
		inputFile      string
		expectedOutput bool
	}

	var gid = []githubData{
		{"github_issue_mapping.json", true},
	}

	for _, j := range gid {
		testString, err := test.LoadTestData(j.inputFile, provideRemoteGithubDataWithAssignee)
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
		_, ok := OneLevelMap[string(k.expression)]
		assert.Equal(t, ok, true, fmt.Sprintf("Could not access %s from the flattened map ", k))
		// Verifying if the new map is usable.
		for k := range githubKeyMap {
			_, ok := OneLevelMap[string(k.expression)]
			assert.Equal(t, ok, j.expectedOutput, fmt.Sprintf("Could not access %s from the flattened map ", k))
		}
	}
}

func TestFlattenGithubResponseMapWithoutAssignee(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	type githubData struct {
		inputFile      string
		expectedOutput bool
	}

	var gid = []githubData{
		{"github_issue_mapping.json", true},
	}

	for _, j := range gid {
		testString, err := test.LoadTestData(j.inputFile, provideRemoteGithubDataWithoutAssignee)
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
			_, ok := OneLevelMap[string(k.expression)]
			if k == GithubAssignee {
				continue
			}
			assert.Equal(t, ok, j.expectedOutput, fmt.Sprintf("Could not access %s from the flattened map ", k))
		}
	}
}

func TestFlattenJiraResponseMap(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	type jiradata struct {
		inputFile      string
		expectedOutput bool
	}

	var jir = []jiradata{
		{"jira_issue_mapping.json", true},
	}

	for _, j := range jir {

		testString, err := test.LoadTestData(j.inputFile, provideRemoteJiraDataWithAssignee)
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
			_, ok := OneLevelMap[string(k.expression)]
			assert.Equal(t, ok, j.expectedOutput, fmt.Sprintf("Could not access %s from the flattened map ", k))
		}
	}
}

func TestFlattenJiraResponseMapWithoutAssignee(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	type jiradata struct {
		inputFile      string
		expectedOutput bool
	}

	var jir = []jiradata{
		{"jira_issue_mapping.json", true},
	}

	for _, j := range jir {

		testString, err := test.LoadTestData(j.inputFile, provideRemoteJiraDataWithoutAssignee)
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
			_, ok := OneLevelMap[string(k.expression)]
			if k == JiraAssignee {
				continue
			}
			assert.Equal(t, ok, j.expectedOutput, fmt.Sprintf("Could not access %s from the flattened map ", k))
		}
	}
}

func TestNewGitHubRemoteWorkItem(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	jsonContent := `{"admins":[{"name":"aslak"}],"name":"shoubhik", "assignee":{"fixes": 2, "complete" : true,"foo":[ 1,2,3,4],"1":"sbose","2":"pranav","participants":{"4":"sbose56","5":"sbose78"}},"name":"shoubhik"}`
	remoteTrackerItem := TrackerItem{Item: jsonContent, RemoteItemID: "xyz", TrackerID: uint64(0)}

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
	remoteTrackerItem := TrackerItem{Item: jsonContent, RemoteItemID: "xyz", TrackerID: uint64(0)}

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
