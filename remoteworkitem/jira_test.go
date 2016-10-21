package remoteworkitem

import (
	"net/http"
	"testing"
	"time"

	"github.com/almighty/almighty-core/resource"
	jira "github.com/andygrunwald/go-jira"
	"github.com/dnaeon/go-vcr/recorder"
)

type fakeJiraIssueFetcher struct{}

func (f *fakeJiraIssueFetcher) listIssues(jql string, options *jira.SearchOptions) ([]jira.Issue, *jira.Response, error) {
	return []jira.Issue{{}}, &jira.Response{}, nil
}

func (f *fakeJiraIssueFetcher) getIssue(issueID string) (*jira.Issue, *jira.Response, error) {
	return &jira.Issue{ID: "1"}, &jira.Response{}, nil
}

func TestJiraFetch(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	f := fakeJiraIssueFetcher{}
	j := JiraTracker{URL: "", Query: ""}
	i := <-j.fetch(&f)
	if string(i.Content) != `{"id":"1"}` {
		t.Errorf("Content is not matching: %#v", string(i.Content))
	}

}

func TestJiraFetchWithRecording(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	r, err := recorder.New("../test/data/jira_fetch_test")
	if err != nil {
		t.Error(err)
	}
	defer r.Stop()

	h := &http.Client{
		Timeout:   100 * time.Second,
		Transport: r.Transport,
	}

	f := jiraIssueFetcher{}
	j := JiraTracker{URL: "https://issues.jboss.org", Query: "project = Arquillian AND status = Closed AND assignee = aslak AND fixVersion = 1.1.11.Final AND priority = Major ORDER BY created ASC"}
	client, _ := jira.NewClient(h, j.URL)
	f.client = client
	fetch := j.fetch(&f)

	i := <-fetch
	if i.ID != `"ARQ-1937"` {
		t.Errorf("ID is not matching: %#v", string(i.ID))
	}

	i = <-fetch
	if i.ID != `"ARQ-1956"` {
		t.Errorf("ID is not matching: %#v", string(i.ID))
	}

	i = <-fetch
	if i.ID != `"ARQ-1996"` {
		t.Errorf("ID is not matching: %#v", string(i.ID))
	}

	i = <-fetch
	if i.ID != `"ARQ-2009"` {
		t.Errorf("ID is not matching: %#v", string(i.ID))
	}
}
