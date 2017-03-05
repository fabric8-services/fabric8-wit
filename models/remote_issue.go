package models

// RemoteIssue defines the additional attributes an issue from a remote provider would have.
type RemoteIssue struct {
	Issue
	sourceURL string
}

// GithubIssue defines how an issue from Github would look like.
type GithubIssue struct {
	RemoteIssue
}

// JiraIssue defines how an issue from Jira would look like
type JiraIssue struct {
	RemoteIssue
}

// IssueMapper creates the contract that derivatives of RemoteIssue needs to implement.
type IssueMapper interface {
	toLocalIssue() WorkItem
}

/*

// TODO: Implement the IssueMapper method(s)

func (issue *GithubIssue) toLocalIssue() WorkItem {
	// Add code to convert the Github Issue to an ALM WorkItem
}

func (issue *JiraIssue) toLocalIssue() WorkItem {
	// Add code to convert the Jira Issue to an ALM WorkItem
}

*/
