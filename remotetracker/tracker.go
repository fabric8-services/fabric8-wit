package remotetracker

// Tracker represents a remote issue tracker
// for e.g github, trello, jira, bugzilla
type Tracker interface{
  Fetch(query string) error
  Import() error
}
