package remotetracker

import (
	"encoding/json"

	"github.com/google/go-github/github"
)

// Github represents github remote issue tracker
type Github struct {
	items []Item
}

func (g *Github) Fetch(url, query string) error {
	client := github.NewClient(nil)
	result, _, _ := client.Search.Issues(query, nil)
	issues := result.Issues

	for l, _ := range issues {
		id, _ := json.Marshal(issues[l].ID)
		description, _ := json.Marshal(issues[l].Body)
		title, _ := json.Marshal(issues[l].Title)
		status, _ := json.Marshal(issues[l].State)
		g.items = append(g.items, Item{ID: string(id), Title: string(title), Description: string(description), State: string(status)})
	}
	return nil
}

func (g *Github) Import() error {
	return nil
}
