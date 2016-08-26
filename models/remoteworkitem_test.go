package models

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
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
	remoteWorkItem := RemoteWorkItem{ID: "xyz", Content: []byte("{\"title\":\"abc\"}")}

	gh, err := NewGitHubRemoteWorkItem(remoteWorkItem)
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

	fmt.Println(workItem)
}

func TestGitHubIssueMapping(t *testing.T) {

	content, err := LoadTestData("github_issue_mapping.json", provideRemoteGithubData)
	if err != nil {
		t.Fatal(err)
	}

	workItemMap := workItemKeyMaps[ProviderGithub]
	remoteWorkItem := RemoteWorkItem{ID: "xyz", Content: []byte(content)}

	gh, err := NewGitHubRemoteWorkItem(remoteWorkItem)
	if err != nil {lWorkItemKey))
		}
		t.Fatal(err)
	}
	workItem, err := Map(gh, workItemMap)
	if err != nil {
		t.Fatal(err)
	}
	
	for _, localWorkItemKey := range workItemKeyMaps[ProviderGithub] {
		if workItem.Fields[localWorkItemKey] == nil {
			t.Error(fmt.Sprintf("%s not mapped", localWorkItemKey))
		}
	}
}

// TestDataProvider defines the simple funcion for returning data from a remote provider
type TestDataProvider func() ([]byte, error)

// LoadTestData attempt to load test data from local disk unless;
// * It does not exist or,
// * Variable REFRESH_DATA is present in ENV
//
// Data is stored under examples/test
// This is done to avoid always depending on remote systems, but also with an option
// to refresh/retest against the 'current' remote system data without manual copy/paste
func LoadTestData(filename string, provider TestDataProvider) ([]byte, error) {
	refreshLocalData := func(path string, refresh TestDataProvider) ([]byte, error) {
		content, err := refresh()
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(path, content, 0644)
		if err != nil {
			return nil, err
		}
		return content, nil
	}

	targetDir := "examples/test/"
	err := os.MkdirAll(targetDir, 0777)
	if err != nil {
		return nil, err
	}

	targetPath := targetDir + filename
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		// Call refresher if data does not exist locally
		return refreshLocalData(targetPath, provider)
	}
	if _, found := os.LookupEnv("REFRESH_DATA"); found {
		// Call refresher if force update of test data set in env
		return refreshLocalData(targetPath, provider)
	}

	return ioutil.ReadFile(targetPath)
}
