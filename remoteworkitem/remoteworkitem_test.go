package remoteworkitem

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"
	"github.com/stretchr/testify/assert"
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

func TestWorkItemMapping(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	workItemMap := WorkItemMap{
		AttributeMapper{AttributeExpression("title"), AttributeConverter(StringConverter{})}: workitem.SystemTitle,
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

	assert.NotNil(t, workItem.Fields[workitem.SystemTitle], fmt.Sprintf("%s not mapped", workitem.SystemTitle))
}

// Table driven tests for the Mapping of Github issues
func TestGitHubIssueMapping(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	// githubData struct define test file and test url
	type githubData struct {
		inputFile      string
		expectedOutput bool
		inputURL       string
	}

	var gitData = []githubData{
		// JSON data file of Github issue with assignee to test that the data is getting correctly Mapped through the Map funtion
		// Github Issue API URL for the respective JSON data file to update the cache
		{"github_issue_with_assignee.json", true, "http://api.github.com/repos/almighty-test/almighty-test-unit/issues/2"},
		// JSON data file of Github issue with assignee and label
		// Issue API URL for the respective JSON file to update the cache
		{"github_issue_with_assignee_labels.json", true, "https://api.github.com/repos/almighty-unit-test/almighty-test/issues/1"},
	}

	for _, j := range gitData {
		content, err := test.LoadTestData(j.inputFile, func() ([]byte, error) {
			return provideRemoteData(j.inputURL)
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

// Table driven tests for the Mapping of Jira issues
func TestJiraIssueMapping(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	// jiraData struct define test file and test url
	type jiraData struct {
		inputFile      string
		expectedOutput bool
		inputURL       string
	}
	// JSON data to test the issue mapping for jira
	var jir = []jiraData{
		// JSON data file of Jira issue with null assignee
		// Issue API URL for the respective JSON file to update the cache
		{"jira_issue_without_assignee.json", true, "http://jira.atlassian.com/rest/api/latest/issue/JRA-9"},
		// JSON data file of Jira issue
		// Issue API URL for the respective JSON file to update the cache
		{"jira_issue_mapping_data.json", true, "https://jira.atlassian.com/rest/api/latest/issue/JRA-3"},
	}

	for _, j := range jir {
		content, err := test.LoadTestData(j.inputFile, func() ([]byte, error) {
			return provideRemoteData(j.inputURL)
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

// Table driven tests for Flattening the Github response data with assignee
func TestFlattenGithubResponseMap(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	// githubData struct define test file and test url
	type githubData struct {
		inputFile      string
		expectedOutput bool
		inputURL       string
	}

	// JSON data to test the issue mapping for github
	var gitData = []githubData{
		// JSON data file of Github issue with assignee to test that the data
		// is getting correctly Mapped through the Map funtion
		// Github Issue API URL for the respective JSON data file to update the cache
		{"github_issue_with_assignee.json", true, "http://api.github.com/repos/almighty-test/almighty-test-unit/issues/2"},
		// Github issue with assignee and label
		{"github_issue_with_assignee_labels.json", true, "https://api.github.com/repos/almighty-unit-test/almighty-test/issues/1"},
		// The Github issue URL doesn't exist. So, the mapping will not happen
		// The map created from the Flatten will be empty
		{"github_issue_invalid.json", false, "https://api.github.com/repos/almighty-unit-test/almighty-test/issues/255"},
	}

	for _, j := range gitData {
		testString, err := test.LoadTestData(j.inputFile, func() ([]byte, error) {
			return provideRemoteData(j.inputURL)
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

// Table driven tests for Flattening the Github response data without assignee
func TestFlattenGithubResponseMapWithoutAssignee(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	// githubData struct define test file and test url
	type githubData struct {
		inputFile      string
		expectedOutput bool
		inputURL       string
	}

	// JSON data to test the issue mapping for github
	var gitData = []githubData{
		// Github data with assignee to map local workItem to remote workItem
		{"github_issue_with_assignee.json", true, "http://api.github.com/repos/almighty-test/almighty-test-unit/issues/2"},
		// Github data with labels and without assignee
		// assignee field is skipped if that is null
		{"github_issue_with_labels.json", true, "https://api.github.com/repos/almighty-test/almighty-test-unit/issues/3"},
		// The Github issue URL doesn't exist. So, the mapping will not happen
		// The map created from the Flatten will be empty
		{"github_issue_invalid.json", false, "https://api.github.com/repos/almighty-unit-test/almighty-test/issues/255"},
	}

	for _, j := range gitData {
		testString, err := test.LoadTestData(j.inputFile, func() ([]byte, error) {
			return provideRemoteData(j.inputURL)
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

	// jiraData struct define test file and test url
	type jiraData struct {
		inputFile      string
		expectedOutput bool
		inputURL       string
	}

	// JSON data to test the issue mapping for jira
	var jir = []jiraData{
		// JSON data file of Jira issue with null assignee, test assertion, Issue API URL for the respective JSON file to update the cache
		{"jira_issue_without_assignee.json", true, "http://jira.atlassian.com/rest/api/latest/issue/JRA-9"},
		// JSON data file of Jira issue with null assignee, test assertion, Issue API URL for the respective JSON file to update the cache
		{"jira_issue_with_null_assignee.json", true, "https://jira.atlassian.com/rest/api/latest/issue/JRA-1300"},
	}

	for _, j := range jir {

		testString, err := test.LoadTestData(j.inputFile, func() ([]byte, error) {
			return provideRemoteData(j.inputURL)
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

	// jiraData struct define test file and test url
	type jiraData struct {
		inputFile      string
		expectedOutput bool
		inputURL       string
	}

	// JSON data to test the issue mapping for jira
	var jir = []jiraData{
		// JSON data file of Jira issue with null assignee, test assertion, Issue API URL for the respective JSON file to update the cache
		{"jira_issue_without_assignee.json", true, "http://jira.atlassian.com/rest/api/latest/issue/JRA-10"},
		// JSON data file of Jira issue, test assertion, Issue API URL for the respective JSON file to update the cache
		{"jira_issue_mapping_data.json", true, "https://jira.atlassian.com/rest/api/latest/issue/JRA-3"},
	}

	for _, j := range jir {

		testString, err := test.LoadTestData(j.inputFile, func() ([]byte, error) {
			return provideRemoteData(j.inputURL)
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
