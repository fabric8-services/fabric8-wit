package test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/fabric8-services/fabric8-wit/rest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DummyHttpClient struct {
	Response      *http.Response
	Error         error
	AssertRequest func(req *http.Request)
}

type DummyHttpDoer struct {
	*rest.HttpClientDoer
	Client *DummyHttpClient
}

func (c *DummyHttpClient) Do(req *http.Request) (*http.Response, error) {
	if c.AssertRequest != nil {
		c.AssertRequest(req)
	}
	return c.Response, c.Error
}

func NewDummyHttpDoer() *DummyHttpDoer {
	client := &DummyHttpClient{}
	doer := &rest.HttpClientDoer{HttpClient: client}
	return &DummyHttpDoer{HttpClientDoer: doer, Client: client}
}

func EqualURLs(t *testing.T, expected string, actual string) {
	expectedURL, err := url.Parse(expected)
	require.Nil(t, err)
	actualURL, err := url.Parse(actual)
	require.Nil(t, err)
	assert.Equal(t, expectedURL.Scheme, actualURL.Scheme)
	assert.Equal(t, expectedURL.Host, actualURL.Host)
	assert.Equal(t, len(expectedURL.Query()), len(actualURL.Query()))
	for name, value := range expectedURL.Query() {
		assert.Equal(t, value, actualURL.Query()[name])
	}
}
