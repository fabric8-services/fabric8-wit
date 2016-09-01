package remoteworkitem

import (
	"encoding/json"

	"github.com/google/go-github/github"
	"github.com/satori/go.uuid"
)

// Fetch tracker items from Github
func (g *Github) Fetch() {
	client := github.NewClient(nil)
	result, _, _ := client.Search.Issues(g.Query, nil)
	issues := result.Issues
	bID := batchID()
	for l := range issues {
		i := make(map[string]string)
		id, _ := json.Marshal(issues[l].URL)
		content, _ := json.Marshal(issues[l])
		i = map[string]string{"id": string(id), "content": string(content), "batch_id": bID}
		g.Item <- i
	}
	close(g.Item)
}

// NextItem tracker items from Github
func (g *Github) NextItem() chan map[string]string {
	return g.Item
}

func batchID() string {
	u1 := uuid.NewV4().String()
	return u1
}
