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
	ID string
	Title       string
	Description string
	Status      string
}



func PrintIssue(issue Issue) {
	fmt.Println("ID: ", issue.ID)
	fmt.Println("title: ", issue.Title)
	fmt.Println("issue: ", issue.Description)
	fmt.Println("status: ", issue.Status)
	fmt.Println("")
}

// Interface to Fetch data from issue trackers.
type IssueProvider interface {
	FetchData(ctx context.Context) (chan Issue, chan error)
}
