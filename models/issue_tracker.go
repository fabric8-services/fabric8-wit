package models

import (
	"fmt"
	"golang.org/x/net/context"
)

// Configuration describes authentication model for the issue tracker.
type Configuration struct {
	ApiKey   string
	Token    string
	UserName string
}

// Issue gives information of every single issue of the issue trackers.
type Issue struct {
	Title       string
	Description string
}

func PrintIssue(issue Issue) {
	fmt.Println("title: ", issue.Title)
	fmt.Println("issue: ", issue.Description)
	fmt.Println("")
}

// Interface to Fetch data from issue trackers.
type IssueProvider interface {
	FetchData(ctx context.Context) (chan Issue, chan error)
}
