package remoteworkitem_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func provideRemoteData(dataURL string) ([]byte, error) {
	response, err := http.Get(dataURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer response.Body.Close()
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return responseData, nil
}

func TestWorkItemMapping(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	workItemMap := remoteworkitem.RemoteWorkItemMap{
		remoteworkitem.AttributeMapper{remoteworkitem.AttributeExpression("title"), remoteworkitem.AttributeConverter(remoteworkitem.StringConverter{})}: workitem.SystemTitle,
	}
	jsonContent := `{"title":"abc"}`
	remoteTrackerItem := remoteworkitem.TrackerItem{Item: jsonContent, RemoteItemID: "xyz", TrackerID: uuid.NewV4()}
	remoteWorkItemImpl := remoteworkitem.RemoteWorkItemImplRegistry[remoteworkitem.ProviderGithub]
	gh, err := remoteWorkItemImpl(remoteTrackerItem)
	require.NoError(t, err)
	// when
	workItem, err := remoteworkitem.Map(gh, workItemMap)
	// then
	require.NoError(t, err)
	assert.NotNil(t, workItem.Fields[workitem.SystemTitle], fmt.Sprintf("%s not mapped", workitem.SystemTitle))
}

// remoteData struct define test file and test url
type remoteData struct {
	inputFile      string
	expectedOutput bool
	inputURL       string
}

// Table driven tests for the Mapping of Github issues
func TestGitHubIssueMapping(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	var gitData = []remoteData{
		// JSON data file of Github issue with assignee to test that the data is getting correctly Mapped through the Map function
		// Github Issue API URL for the respective JSON data file to update the cache
		{"github_issue_with_assignee.json", true, "http://api.github.com/repos/fabric8-wit-test/fabric8-wit-test-unit/issues/2"},
		// JSON data file of Github issue with assignee and label
		// Issue API URL for the respective JSON file to update the cache
		{"github_issue_with_assignee_labels.json", true, "https://api.github.com/repos/fabric8-wit-unit-test/fabric8-wit-test/issues/1"},
	}
	// when/then
	for _, j := range gitData {
		doTestIssueMapping(t, j, remoteworkitem.ProviderGithub)
	}
}

// Table driven tests for the Mapping of Jira issues
func TestJiraIssueMapping(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	// JSON data to test the issue mapping for jira
	var jiraData = []remoteData{
		// JSON data file of Jira issue with null assignee
		// Issue API URL for the respective JSON file to update the cache
		{"jira_issue_without_assignee.json", true, "http://jira.atlassian.com/rest/api/latest/issue/JRA-9"},
		// JSON data file of Jira issue
		// Issue API URL for the respective JSON file to update the cache
		{"jira_issue_mapping_data.json", true, "https://jira.atlassian.com/rest/api/latest/issue/JRA-3"},
	}
	// when/then
	for _, j := range jiraData {
		doTestIssueMapping(t, j, remoteworkitem.ProviderJira)
	}
}

func doTestIssueMapping(t *testing.T, data remoteData, provider string) {
	// given
	content, err := test.LoadTestData(data.inputFile, func() ([]byte, error) {
		return provideRemoteData(data.inputURL)
	})
	require.NoError(t, err)
	workItemMap := remoteworkitem.RemoteWorkItemKeyMaps[provider]
	remoteTrackerItem := remoteworkitem.TrackerItem{Item: string(content[:]), RemoteItemID: "xyz", TrackerID: uuid.NewV4()}
	remoteWorkItemImpl := remoteworkitem.RemoteWorkItemImplRegistry[remoteworkitem.ProviderJira]
	issue, err := remoteWorkItemImpl(remoteTrackerItem)
	require.NoError(t, err)
	// when
	workItem, err := remoteworkitem.Map(issue, workItemMap)
	require.NoError(t, err)
	// then
	for _, localWorkItemKey := range workItemMap {
		t.Log("Mapping ", localWorkItemKey)
		_, ok := workItem.Fields[localWorkItemKey]
		assert.Equal(t, ok, data.expectedOutput, fmt.Sprintf("%s not mapped", localWorkItemKey))
	}
}

// Table driven tests for Flattening the Github response data with assignee
func TestFlattenGithubResponseMap(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	// JSON data to test the issue mapping for github
	var gitData = []remoteData{
		// JSON data file of Github issue with assignee to test that the data
		// is getting correctly Mapped through the Map function
		// Github Issue API URL for the respective JSON data file to update the cache
		{"github_issue_with_assignee.json", true, "http://api.github.com/repos/fabric8-wit-test/fabric8-wit-test-unit/issues/2"},
		// Github issue with assignee and label
		{"github_issue_with_assignee_labels.json", true, "https://api.github.com/repos/fabric8-wit-unit-test/fabric8-wit-test/issues/1"},
		// The Github issue URL doesn't exist. So, the mapping will not happen
		// The map created from the Flatten will be empty
		{"github_issue_invalid.json", false, "https://api.github.com/repos/fabric8-wit-unit-test/fabric8-wit-test/issues/255"},
	}

	for _, data := range gitData {
		doTestFlattenResponseMap(t, data, remoteworkitem.ProviderGithub)
	}
}

func TestFlattenJiraResponseMap(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	// JSON data to test the issue mapping for jira
	var jir = []remoteData{
		// JSON data file of Jira issue, test assertion, Issue API URL for the respective JSON file to update the cache
		{"jira_issue_mapping_data.json", true, "https://jira.atlassian.com/rest/api/latest/issue/JRA-3"},
	}

	for _, data := range jir {
		doTestFlattenResponseMap(t, data, remoteworkitem.ProviderJira)
	}
}

// Table driven tests for Flattening the Github response data without assignee
func TestFlattenGithubResponseMapWithoutAssignee(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	// JSON data to test the issue mapping for github
	var gitData = []remoteData{
		// Github data with labels and without assignee
		// assignees field is skipped if that is an empty array
		{"github_issue_with_labels.json", true, "https://api.github.com/repos/fabric8-wit-test/fabric8-wit-test-unit/issues/3"},
		// The Github issue URL doesn't exist. So, the mapping will not happen
		// The map created from the Flatten will be empty
		{"github_issue_invalid.json", false, "https://api.github.com/repos/fabric8-wit-unit-test/fabric8-wit-test/issues/255"},
	}
	// when/then
	for _, data := range gitData {
		// skipping assignees login and URL since the test data contain no assignee
		doTestFlattenResponseMap(t, data, remoteworkitem.ProviderGithub, remoteworkitem.GithubAssigneesLogin, remoteworkitem.GithubAssigneesProfileURL)
	}
}

func TestFlattenJiraResponseMapWithoutAssignee(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	// JSON data to test the issue mapping for jira
	var jir = []remoteData{
		// JSON data file of Jira issue with null assignee, test assertion, Issue API URL for the respective JSON file to update the cache
		{"jira_issue_without_assignee.json", true, "http://jira.atlassian.com/rest/api/latest/issue/JRA-10"},
		// JSON data file of Jira issue with null assignee, test assertion, Issue API URL for the respective JSON file to update the cache
		{"jira_issue_without_assignee.json", true, "http://jira.atlassian.com/rest/api/latest/issue/JRA-9"},
		// JSON data file of Jira issue with null assignee, test assertion, Issue API URL for the respective JSON file to update the cache
		{"jira_issue_with_null_assignee.json", true, "https://jira.atlassian.com/rest/api/latest/issue/JRA-1300"},
	}

	for _, data := range jir {
		// skipping assignee login and URL since the test data contain no assignee
		doTestFlattenResponseMap(t, data, remoteworkitem.ProviderJira, remoteworkitem.JiraAssigneeLogin, remoteworkitem.JiraAssigneeProfileURL)
	}
}

func doTestFlattenResponseMap(t *testing.T, data remoteData, provider string, skipFields ...string) {
	// given
	testString, err := test.LoadTestData(data.inputFile, func() ([]byte, error) {
		return provideRemoteData(data.inputURL)
	})
	require.NoError(t, err)
	var nestedMap map[string]interface{}
	err = json.Unmarshal(testString, &nestedMap)
	require.NoError(t, err, "Incorrect dataset %s", testString)
	// when
	oneLevelMap := remoteworkitem.Flatten(nestedMap)
	// then: verifying that the newly converted map contains all expected keys
KEYS:
	for k := range remoteworkitem.RemoteWorkItemKeyMaps[provider] {
		key := string(k.Expression)
		for _, skipField := range skipFields {
			if skipField == key {
				// skip the key
				continue KEYS
			}
		}
		v, exists := oneLevelMap[key]
		switch v.(type) {
		case string:
			value := v.(string)
			l := int(math.Min(float64(60), float64(len(value))))
			t.Log(fmt.Sprintf("\t '%s'='%v' (expected=%v)", key, value[0:l], data.expectedOutput))
		}
		assert.Equal(t, exists, data.expectedOutput, fmt.Sprintf("Could not access '%s' from the flattened map ", key))
	}
}

func TestNewGitHubRemoteWorkItem(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	jsonContent := `
		{
			"admins": [
				{
					"name": "aslak"
				}
			],
			"name": "shoubhik",
			"assignee": {
				"fixes": 2,
				"complete": true,
				"foo": [1, 2, 3, 4],
				"1": "sbose",
				"2": "pranav",
				"participants": {
					"4": "sbose56",
					"5": "sbose78"
			 	}
			}
		}`
	// when
	remoteTrackerItem := remoteworkitem.TrackerItem{Item: jsonContent, RemoteItemID: "xyz", TrackerID: uuid.NewV4()}
	githubRemoteWorkItem, err := remoteworkitem.NewGitHubRemoteWorkItem(remoteTrackerItem)
	// then
	require.NoError(t, err)
	assert.Equal(t, githubRemoteWorkItem.Get("admins.0.name"), "aslak")
	assert.Equal(t, githubRemoteWorkItem.Get("name"), "shoubhik")
	assert.Equal(t, githubRemoteWorkItem.Get("assignee.complete"), true)
	assert.Equal(t, githubRemoteWorkItem.Get("assignee.participants.4"), "sbose56")
}

func TestNewJiraRemoteWorkItem(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	jsonContent := `
			{
			"admins": [
				{
					"name": "aslak"
				}
			],
			"name": "shoubhik",
			"assignee": {
				"fixes": 2,
				"complete": true,
				"foo": [1, 2, 3, 4 ],
				"1": "sbose",
				"2": "pranav",
				"participants": {
					"4": "sbose56",
					"5": "sbose78"
				}
			}
		}`
	// when
	remoteTrackerItem := remoteworkitem.TrackerItem{Item: jsonContent, RemoteItemID: "xyz", TrackerID: uuid.NewV4()}
	jiraRemoteWorkItem, err := remoteworkitem.NewJiraRemoteWorkItem(remoteTrackerItem)
	// then
	require.NoError(t, err)
	assert.Equal(t, jiraRemoteWorkItem.Get("admins.0.name"), "aslak")
	assert.Equal(t, jiraRemoteWorkItem.Get("name"), "shoubhik")
	assert.Equal(t, jiraRemoteWorkItem.Get("assignee.complete"), true)
	assert.Equal(t, jiraRemoteWorkItem.Get("assignee.participants.4"), "sbose56")
}

func TestRemoteWorkItemImplRegistry(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	// when
	_, ok := remoteworkitem.RemoteWorkItemImplRegistry[remoteworkitem.ProviderGithub]
	// then
	assert.True(t, ok)
	// when
	_, ok = remoteworkitem.RemoteWorkItemImplRegistry[remoteworkitem.ProviderJira]
	// then
	assert.True(t, ok)
}

func TestPatternConverter(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// given
	content := make(map[string]interface{})
	content[remoteworkitem.GithubState] = "open"
	content["assignees.0.login"] = "foo0"
	content["assignees.1.login"] = "foo1"
	content["assignees.2.login"] = "foo2"
	content["assignees.0.url"] = "/foo0"
	content["assignees.1.url"] = "/foo1"
	content["assignees.2.url"] = "/foo2"
	workItem := TestWorkItem{
		content: content,
	}
	workItemMap := remoteworkitem.RemoteWorkItemKeyMaps[remoteworkitem.ProviderGithub]
	// when
	result, err := remoteworkitem.Map(workItem, workItemMap)
	// then
	require.NoError(t, err)
	require.NotNil(t, result.Fields[remoteworkitem.RemoteAssigneeLogins])
	assert.Contains(t, result.Fields[remoteworkitem.RemoteAssigneeLogins], content["assignees.0.login"])
	assert.Contains(t, result.Fields[remoteworkitem.RemoteAssigneeLogins], content["assignees.1.login"])
	assert.Contains(t, result.Fields[remoteworkitem.RemoteAssigneeLogins], content["assignees.2.login"])
	require.NotNil(t, result.Fields[remoteworkitem.RemoteAssigneeProfileURLs])
	assert.Contains(t, result.Fields[remoteworkitem.RemoteAssigneeProfileURLs], content["assignees.0.url"])
	assert.Contains(t, result.Fields[remoteworkitem.RemoteAssigneeProfileURLs], content["assignees.1.url"])
	assert.Contains(t, result.Fields[remoteworkitem.RemoteAssigneeProfileURLs], content["assignees.2.url"])
}

func TestPatternConverterWithNoValue(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// given
	content := make(map[string]interface{})
	content[remoteworkitem.GithubState] = "open"
	workItem := TestWorkItem{
		content: content,
	}
	workItemMap := remoteworkitem.RemoteWorkItemKeyMaps[remoteworkitem.ProviderGithub]
	// when
	result, err := remoteworkitem.Map(workItem, workItemMap)
	// then
	require.NoError(t, err)
	require.NotNil(t, result.Fields[remoteworkitem.RemoteAssigneeLogins])
	require.Empty(t, result.Fields[remoteworkitem.RemoteAssigneeLogins])
	require.NotNil(t, result.Fields[remoteworkitem.RemoteAssigneeProfileURLs])
	require.Empty(t, result.Fields[remoteworkitem.RemoteAssigneeProfileURLs])
}

type TestWorkItem struct {
	content map[string]interface{}
}

func (t TestWorkItem) Get(field remoteworkitem.AttributeExpression) interface{} {
	return t.content[string(field)]
}

func TestListConverter(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// given
	content := make(map[string]interface{})
	content[remoteworkitem.JiraState] = "open"
	content[remoteworkitem.JiraAssigneeLogin] = "foo0"
	content[remoteworkitem.JiraAssigneeProfileURL] = "/foo/1"
	workItem := TestWorkItem{
		content: content,
	}
	workItemMap := remoteworkitem.RemoteWorkItemKeyMaps[remoteworkitem.ProviderJira]
	// when
	result, err := remoteworkitem.Map(workItem, workItemMap)
	// then
	require.NoError(t, err)
	require.NotNil(t, result.Fields[remoteworkitem.RemoteAssigneeLogins])
	assert.Contains(t, result.Fields[remoteworkitem.RemoteAssigneeLogins], content[remoteworkitem.JiraAssigneeLogin])
	require.NotNil(t, result.Fields[remoteworkitem.RemoteAssigneeProfileURLs])
	assert.Contains(t, result.Fields[remoteworkitem.RemoteAssigneeProfileURLs], content[remoteworkitem.JiraAssigneeProfileURL])
}

func TestListConverterWithNoValue(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// given
	content := make(map[string]interface{})
	content[remoteworkitem.JiraState] = "open"
	workItem := TestWorkItem{
		content: content,
	}
	workItemMap := remoteworkitem.RemoteWorkItemKeyMaps[remoteworkitem.ProviderJira]
	// when
	result, err := remoteworkitem.Map(workItem, workItemMap)
	// then
	require.NoError(t, err)
	require.NotNil(t, result.Fields[remoteworkitem.RemoteAssigneeLogins])
	require.Empty(t, result.Fields[remoteworkitem.RemoteAssigneeLogins])
	require.NotNil(t, result.Fields[remoteworkitem.RemoteAssigneeProfileURLs])
	require.Empty(t, result.Fields[remoteworkitem.RemoteAssigneeProfileURLs])
}
