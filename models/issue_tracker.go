package models

import (
	"fmt"
	"strings"
)

type Configuration struct {
	ApiKey   string
	Token    string
	UserName string
}

type TrelloIssueProvider struct {
	Configuration
	BoardId  string
	ListName string
}

type GithubIssueProvider struct {
	Query string
}

type Issue struct {
	title       string
	description string
}

func PrintIssue(issue Issue) {
	fmt.Println("title: ", issue.title)
	fmt.Println("issue: ", issue.description)
	fmt.Println("")
}

type IssueProvider interface {
	FetchData(chan String) (chan Issue, chan error)
}
