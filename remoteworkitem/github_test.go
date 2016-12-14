package remoteworkitem

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/almighty/almighty-core/resource"
	"github.com/dnaeon/go-vcr/recorder"
	"github.com/google/go-github/github"
)

type fakeGithubIssueFetcher struct{}

// ListIssues list all issues
func (f *fakeGithubIssueFetcher) listIssues(query string, opts *github.SearchOptions) (*github.IssuesSearchResult, *github.Response, error) {
	if opts.ListOptions.Page == 0 {
		one := 1
		i := github.Issue{ID: &one}
		isr := &github.IssuesSearchResult{Issues: []github.Issue{i}}
		r := &github.Response{}
		r.NextPage = 1
		return isr, r, nil
	}
	isr := &github.IssuesSearchResult{}
	r := &github.Response{}
	r.NextPage = 0
	return isr, r, nil

}

func TestGithubFetch(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	f := fakeGithubIssueFetcher{}
	g := GithubTracker{URL: "", Query: ""}
	fetch := g.fetch(&f)
	i := <-fetch
	if string(i.Content) != `{"id":1}` {
		t.Errorf("Content is not matching: %#v", string(i.Content))
	}

}

type fakeGithubIssueFetcherWithRateLimit struct{}

// ListIssues list all issues
func (f *fakeGithubIssueFetcherWithRateLimit) listIssues(query string, opts *github.SearchOptions) (*github.IssuesSearchResult, *github.Response, error) {
	isr := &github.IssuesSearchResult{}
	r := &github.Response{}
	r.NextPage = 0
	e := &github.RateLimitError{}
	return isr, r, e

}

func TestGithubFetchWithRateLimit(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	f := fakeGithubIssueFetcherWithRateLimit{}
	g := GithubTracker{URL: "", Query: ""}
	fetch := g.fetch(&f)
	if len(fetch) > 0 {
		t.Error("Channel should not have any data")
	}
}

func TestGithubFetchWithRecording(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	r, err := recorder.New("../test/data/github_fetch_test")
	if err != nil {
		t.Error(err)
	}
	defer r.Stop()

	h := &http.Client{
		Timeout:   1000 * time.Second,
		Transport: r.Transport,
	}

	f := githubIssueFetcher{}
	f.client = github.NewClient(h)
	g := &GithubTracker{URL: "", Query: "is:open is:issue user:almighty-test"}
	fetch := g.fetch(&f)
	i := <-fetch
	if !strings.Contains(string(i.Content), `"html_url":"https://github.com/almighty-test/almighty-test-unit/issues/3"`) {
		t.Errorf("Content is not matching: %#v", string(i.Content))
	}
	i2 := <-fetch
	if !strings.Contains(string(i2.Content), `"html_url":"https://github.com/almighty-test/almighty-test-unit/issues/2"`) {
		t.Errorf("Content is not matching: %#v", string(i2.Content))
	}
}
