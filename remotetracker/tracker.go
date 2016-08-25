package remotetracker

// Tracker represents a remote issue tracker
// for e.g github, trello, jira, bugzilla
type Tracker interface {
	Fetch(query string) error
	Import() error
}

type Item struct {
	ID          string
	Title       string
	Description string
	State       string
}
