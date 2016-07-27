package github

import (
	"fmt"
	"github.com/google/go-github/github"
	"strings"
)

func (g GithubIssueProvider) FetchData(done chan String) (result chan Issue, err chan error) {
	result := make(chan Issue)
	client := github.NewClient(nil)
	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	res, _, err := client.Search.Issues(g.Query, opts)
	issues := res.Issues
	for l, _ := range issues {
		url := issues[l].URL
		title := issues[l].Title
		issueInstance := Issue{*title, *url}
		result <- issueInstance
	}
	close(result)
	if err != nil {
		fmt.Printf("error: %v\n\n", err)
	}
}
