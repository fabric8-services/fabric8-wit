package remoteworkitem

import (
	"encoding/json"
	"os"

	"github.com/andygrunwald/go-jira"
)

// Fetch collects data from Jira
func (j *Jira) Fetch() {
	client, _ := jira.NewClient(nil, j.URL)
	issues, _, _ := client.Issue.Search(j.Query, nil)
	bID := batchID()
	for l := range issues {
		i := make(map[string]string)
		id, _ := json.Marshal(issues[l].Key)
		issue, _, _ := client.Issue.Get(issues[l].Key)
		rawData := json.NewEncoder(os.Stdout).Encode(issue)
		data := rawData.Error()
		i = map[string]string{"id": string(id), "content": data, "batch_id": bID}
		j.Item <- i
	}
	close(j.Item)
}

// NextItem imports the items into database
func (j *Jira) NextItem() chan map[string]string {
	return j.Item
}
