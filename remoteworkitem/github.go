package remoteworkitem

import (
	"encoding/json"
	"log"

	"github.com/google/go-github/github"
	"github.com/satori/go.uuid"
)

// Fetch tracker items from Github
func (g *Github) Fetch() chan map[string]string {
	item := make(chan map[string]string)
	go func() {
		bID := batchID()
		opts := &github.SearchOptions{
			ListOptions: github.ListOptions{
				PerPage: 20,
			},
		}
		client := github.NewClient(nil)
		for {
			result, response, err := client.Search.Issues(g.Query, opts)
			if _, ok := err.(*github.RateLimitError); ok {
				log.Println("reached rate limit", err)
				break
			}
			issues := result.Issues
			for _, l := range issues {
				i := make(map[string]string)
				id, _ := json.Marshal(l.URL)
				content, _ := json.Marshal(l)
				i = map[string]string{"id": string(id), "content": string(content), "batch_id": bID}
				item <- i
			}
			if response.NextPage == 0 {
				break
			}
			opts.ListOptions.Page = response.NextPage
		}
		close(item)
	}()
	return item
}

func batchID() string {
	u1 := uuid.NewV4().String()
	return u1
}
