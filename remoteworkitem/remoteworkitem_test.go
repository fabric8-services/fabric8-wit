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

// var GithubIssueWithAssignee = []string{"http://api.github.com/repos/almighty-test/almighty-test-unit/issues/2", "https://api.github.com/repos/almighty-unit-test/almighty-test/issues/1"}
// var GithubIssueWithoutAssignee = []string{"http://api.github.com/repos/almighty-test/almighty-test-unit/issues/1", "https://api.github.com/repos/almighty-test/almighty-test-unit/issues/3"}
// var JiraIssueWithAssignee = []string{"http://jira.atlassian.com/rest/api/latest/issue/JRA-9"}
// var JiraIssueWithoutAssignee = []string{"http://jira.atlassian.com/rest/api/latest/issue/JRA-10"}

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

func provideRemoteGithubDataWithAssignee(input string) ([]byte, error) {
	return provideRemoteData(input)
}

func provideRemoteJiraDataWithAssignee(input string) ([]byte, error) {
	return provideRemoteData(input)
}

func provideRemoteGithubDataWithoutAssignee(input string) ([]byte, error) {
	return provideRemoteData(input)
}

func provideRemoteJiraDataWithoutAssignee(input string) ([]byte, error) {
	return provideRemoteData(input)
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
		inputURL       string
	}

	var gitData = []githubData{
		{"github_issue_mapping.json", true, "http://api.github.com/repos/almighty-test/almighty-test-unit/issues/2"},
		{"github_test2_data.json", true, "https://api.github.com/repos/almighty-unit-test/almighty-test/issues/1"},
	}

	for _, j := range gitData {
		content, err := test.LoadTestData(j.inputFile, func() ([]byte, error) {
			return provideRemoteGithubDataWithAssignee(j.inputURL)
		})
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

	type jiraData struct {
		inputFile      string
		expectedOutput bool
		inputURL       string
	}

	var jir = []jiraData{
		{"jira_issue_mapping.json", true, "http://jira.atlassian.com/rest/api/latest/issue/JRA-9"},
	}

	for _, j := range jir {
		content, err := test.LoadTestData(j.inputFile, func() ([]byte, error) {
			return provideRemoteJiraDataWithAssignee(j.inputURL)
		})
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
		inputURL       string
	}

	var gitData = []githubData{
		{"github_issue_mapping.json", true, "http://api.github.com/repos/almighty-test/almighty-test-unit/issues/2"},
		{"github_test2_data.json", true, "https://api.github.com/repos/almighty-unit-test/almighty-test/issues/1"},
		{"github_test3_data.json", false, "https://api.github.com/repos/almighty-unit-test/almighty-test/issues/255"},
	}

	for _, j := range gitData {
		testString, err := test.LoadTestData(j.inputFile, func() ([]byte, error) {
			return provideRemoteGithubDataWithAssignee(j.inputURL)
		})
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
			assert.Equal(t, ok, j.expectedOutput, fmt.Sprintf("Could not access %s from the flattened map ", k))
		}
	}
}

func TestFlattenGithubResponseMapWithoutAssignee(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	type githubData struct {
		inputFile      string
		expectedOutput bool
		inputURL       string
	}

	var gitData = []githubData{
		{"github_issue_mapping.json", true, "http://api.github.com/repos/almighty-test/almighty-test-unit/issues/2"},
		{"github_test_data.json", true, "https://api.github.com/repos/almighty-test/almighty-test-unit/issues/3"},
		{"github_test3_data.json", false, "https://api.github.com/repos/almighty-unit-test/almighty-test/issues/255"},
	}

	for _, j := range gitData {
		testString, err := test.LoadTestData(j.inputFile, func() ([]byte, error) {
			return provideRemoteGithubDataWithoutAssignee(j.inputURL)
		})
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
			if k.expression == GithubAssignee {
				continue
			}
			assert.Equal(t, ok, j.expectedOutput, fmt.Sprintf("Could not access %s from the flattened map ", k))
		}
	}
}

func TestFlattenJiraResponseMap(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	type jiraData struct {
		inputFile      string
		expectedOutput bool
		inputURL       string
	}

	var jir = []jiraData{
		{"jira_issue_mapping.json", true, "http://jira.atlassian.com/rest/api/latest/issue/JRA-9"},
	}

	for _, j := range jir {

		testString, err := test.LoadTestData(j.inputFile, func() ([]byte, error) {
			return provideRemoteJiraDataWithAssignee(j.inputURL)
		})
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

	type jiraData struct {
		inputFile      string
		expectedOutput bool
		inputURL       string
	}

	var jir = []jiraData{
		{"jira_issue_mapping.json", true, "http://jira.atlassian.com/rest/api/latest/issue/JRA-10"},
	}

	for _, j := range jir {

		testString, err := test.LoadTestData(j.inputFile, func() ([]byte, error) {
			return provideRemoteJiraDataWithoutAssignee(j.inputURL)
		})
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
			if k.expression == JiraAssignee {
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
