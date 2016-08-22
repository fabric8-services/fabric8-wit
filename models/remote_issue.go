package models

type RemoteIssue struct{
    Issue
    sourceURL string
    convertRemoteIssueToLocalIssue() WorkItem
}


type GithubIssue struct {
    RemoteIssue
}

type JiraIssue struct {
    RemoteIssue
}

func (issue *GithubIssue) convertRemoteIssueToLocalIssue() WorkItem{
    // Add code to convert the Github Issue to an ALM WorkItem   
}

func (issue *JiraIssue) convertRemoteIssueToLocalIssue() WorkItem{
    // Add code to convert the Jira Issue to an ALM WorkItem   
}