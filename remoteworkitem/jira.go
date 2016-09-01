package remoteworkitem

import (
	"encoding/json"

	"github.com/andygrunwald/go-jira"
)

// Fetch collects data from Jira
func (j *Jira) Fetch() chan map[string]string {
	item := make(chan map[string]string)
	go func() {
		client, _ := jira.NewClient(nil, j.URL)
		issues, _, _ := client.Issue.Search(j.Query, nil)
		bID := batchID()
		for l := range issues {
			i := make(map[string]string)
			id, _ := json.Marshal(issues[l].Key)
			issue, _, _ := client.Issue.Get(issues[l].Key)
			content, _ := json.Marshal(issue)
			i = map[string]string{"id": string(id), "content": string(content), "batch_id": bID}
			item <- i
		}
		close(item)
	}()
	return item
}
