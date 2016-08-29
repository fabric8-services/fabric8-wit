package remotetracker

import (
	"encoding/json"
	"fmt"

	"github.com/google/go-github/github"
	"github.com/jinzhu/gorm"
)

// fetch collects data from Github
func fetchGithub(url, query string, item chan map[string]interface{}) {
	client := github.NewClient(nil)
	result, _, _ := client.Search.Issues(query, nil)
	issues := result.Issues
	for l := range issues {
		i := make(map[string]interface{})
		id, _ := json.Marshal(issues[l].URL)
		description, _ := json.Marshal(issues[l].Body)
		title, _ := json.Marshal(issues[l].Title)
		status, _ := json.Marshal(issues[l].State)
		i = map[string]interface{}{"id": string(id), "title": string(title), "description": string(description), "state": string(status)}
		item <- i
	}
	close(item)
}

// upload imports the items into database
func uploadGithub(db *gorm.DB, item map[string]interface{}) error {
	fmt.Println(item)
	return nil
}
