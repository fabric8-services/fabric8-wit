package remotetracker

import "github.com/andygrunwald/go-jira"

// Jira represents Jira remote issue tracker
type Jira struct {
	items []Item
}

// Fetch collects data from Jira
func (j *Jira) Fetch(url, query string) error {
	client, _ := jira.NewClient(nil, url)
	issues, _, _ := client.Issue.Search(query, nil)

	for l := range issues {
		id := issues[l].Key
		i, _, _ := client.Issue.Get(issues[l].Key)
		title := i.Fields.Summary
		description := i.Fields.Description
		status := i.Fields.Status.Name
		j.items = append(j.items, Item{ID: id, Title: title, Description: description, State: status})
	}
	return nil
}

// Import imports the items into database
func (j *Jira) Import() error {
	return nil
}
