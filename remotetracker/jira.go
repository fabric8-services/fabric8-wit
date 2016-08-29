package remotetracker

import (
	"encoding/json"
	"fmt"

	"github.com/andygrunwald/go-jira"
	"github.com/jinzhu/gorm"
)

// Fetch collects data from Jira
func fetchJira(url, query string, item chan map[string]interface{}) {
	client, _ := jira.NewClient(nil, url)
	issues, _, _ := client.Issue.Search(query, nil)

	for l := range issues {
		i := make(map[string]interface{})
		id, _ := json.Marshal(issues[l].Key)
		issue, _, _ := client.Issue.Get(issues[l].Key)
		title, _ := json.Marshal(issue.Fields.Summary)
		description, _ := json.Marshal(issue.Fields.Description)
		status, _ := json.Marshal(issue.Fields.Status.Name)
		i = map[string]interface{}{"id": string(id), "title": string(title), "description": string(description), "state": string(status)}
		item <- i
	}
	close(item)
}

// Import imports the items into database
func uploadJira(db *gorm.DB, tqID int, item map[string]interface{}) error {
	fmt.Println(item)
	return nil
}
