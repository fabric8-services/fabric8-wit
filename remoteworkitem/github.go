package remoteworkitem

import (
	"encoding/json"
	"log"
	"time"

	"github.com/almighty/almighty-core/configuration"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// githubFetcher provides issue listing
type githubFetcher interface {
	listIssues(query string, opts *github.SearchOptions) (*github.IssuesSearchResult, *github.Response, error)
}

// GithubTracker represents the Github tracker provider
type GithubTracker struct {
	URL   string
	Query string
}

// GithubIssueFetcher fetch issues from github
type githubIssueFetcher struct {
	client *github.Client
}

// ListIssues list all issues
func (f *githubIssueFetcher) listIssues(query string, opts *github.SearchOptions) (*github.IssuesSearchResult, *github.Response, error) {
	return f.client.Search.Issues(query, opts)
}

// Fetch tracker items from Github
func (g *GithubTracker) Fetch() chan TrackerItemContent {
	f := githubIssueFetcher{}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: configuration.GetGithubAuthToken()},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	f.client = github.NewClient(tc)
	return g.fetch(&f)
}

func (g *GithubTracker) fetch(f githubFetcher) chan TrackerItemContent {
	item := make(chan TrackerItemContent)
	go func() {
		opts := &github.SearchOptions{
			Sort:  "updated",
			Order: "desc",
			ListOptions: github.ListOptions{
				PerPage: 20,
			},
		}
		for {
			result, response, err := f.listIssues(g.Query, opts)
			if _, ok := err.(*github.RateLimitError); ok {
				log.Println("reached rate limit", err)
				break
			}
			issues := result.Issues
			for _, l := range issues {
				id, _ := json.Marshal(l.URL)
				lu, _ := json.Marshal(l.UpdatedAt)
				lut, _ := time.Parse("\"2006-01-02T15:04:05Z\"", string(lu))
				content, _ := json.Marshal(l)
				item <- TrackerItemContent{ID: string(id), Content: content, LastUpdated: &lut}
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
