package controller

import (
	"context"
	"regexp"
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// While registering URLs do not include protocol because it will be removed before scanning starts
	// Please do not include trailing slashes because it will be removed before scanning starts
	RegisterAsKnownURL("test-work-item-list-details", `"(?P<domain>[^/]+)/(?P<org>[^/]+)/(?P<space>[^/]+)/(?P<path>plan/detail)/(?P<number>.*)"`)
	RegisterAsKnownURL("test-work-item-board-details", `"(?P<domain>[^/]+)/(?P<org>[^/]+)/(?P<space>[^/]+)/(?P<path>board/detail)/(?P<number>.*)"`)
}

type searchTestData struct {
	query    string
	expected search.Keywords
}

func TestParseSearchString(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	t.Run("string terms", func(t *testing.T) {
		t.Parallel()
		input := "user input for search string with some ids like number:99 and number:400 but this is not id like 800"
		op, _ := parseSearchString(context.Background(), input)
		expectedSearchRes := search.Keywords{
			Number: []string{"99:*A", "400:*A"},
			Words:  []string{"user:*", "input:*", "for:*", "search:*", "string:*", "with:*", "some:*", "ids:*", "like:*", "and:*", "but:*", "this:*", "is:*", "not:*", "id:*", "like:*", "800:*"},
		}
		t.Log("Parsed search string: ", op)
		assert.True(t, assert.ObjectsAreEqualValues(expectedSearchRes, op))
	})

	t.Run("URL terms", func(t *testing.T) {
		t.Parallel()
		t.Run("plan/detail URL with ID", func(t *testing.T) {
			t.Parallel()
			op, err := parseSearchString(context.Background(), "http://demo.openshift.io/username/spacename/plan/detail/100")
			require.Nil(t, err)
			assert.Equal(t, search.Keywords{
				Words: []string{"(100:*A | demo.openshift.io/username/spacename/plan/detail/100:*)"},
			}, op)
		})
		t.Run("board/detail URL with ID", func(t *testing.T) {
			t.Parallel()
			op, err := parseSearchString(context.Background(), "http://demo.openshift.io/username/spacename/board/detail/100")
			require.Nil(t, err)
			assert.Equal(t, search.Keywords{
				Words: []string{"(100:*A | demo.openshift.io/username/spacename/board/detail/100:*)"},
			}, op)
		})
		t.Run("plan/detail URL without ID", func(t *testing.T) {
			t.Parallel()
			op, err := parseSearchString(context.Background(), "http://demo.openshift.io/username/spacename/plan/detail")
			require.Nil(t, err)
			assert.Equal(t, search.Keywords{
				Words: []string{"demo.openshift.io/username/spacename/plan/detail:*"},
			}, op)
		})
		t.Run("board/detail URL without ID", func(t *testing.T) {
			t.Parallel()
			op, err := parseSearchString(context.Background(), "http://demo.openshift.io/username/spacename/board/detail")
			require.Nil(t, err)
			assert.Equal(t, search.Keywords{
				Words: []string{"demo.openshift.io/username/spacename/board/detail:*"},
			}, op)
		})
		t.Run("different detail URL", func(t *testing.T) {
			t.Parallel()
			resource.Require(t, resource.UnitTest)
			input := "http://demo.redhat.io"
			op, _ := parseSearchString(context.Background(), input)
			expectedSearchRes := search.Keywords{
				Number: nil,
				Words:  []string{"demo.redhat.io:*"},
			}
			assert.True(t, assert.ObjectsAreEqualValues(expectedSearchRes, op))
		})
		t.Run("mixing strings and URL terms", func(t *testing.T) {
			t.Parallel()
			resource.Require(t, resource.UnitTest)
			// do combination of ID, full text and URLs
			// check if it works as expected.
			input := "http://general.url.io http://demo.openshift.io/username/spacename/plan/detail/100 number:300 golang book and           number:900 \t \n unwanted"
			op, _ := parseSearchString(context.Background(), input)
			expectedSearchRes := search.Keywords{
				Number: []string{"300:*A", "900:*A"},
				Words:  []string{"general.url.io:*", "(100:*A | demo.openshift.io/username/spacename/plan/detail/100:*)", "golang:*", "book:*", "and:*", "unwanted:*"},
			}
			assert.True(t, assert.ObjectsAreEqualValues(expectedSearchRes, op))

		})
	})

}

func TestRegisterAsKnownURL(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// build 2 fake urls and cross check against RegisterAsKnownURL
	urlRegex := `(?P<domain>openshift.io)(?P<path>/everything/)(?P<param>.*)`
	routeName := "custom-test-route"
	RegisterAsKnownURL(routeName, urlRegex)
	compiledRegex := regexp.MustCompile(urlRegex)
	groupNames := compiledRegex.SubexpNames()
	var expected = make(map[string]KnownURL)
	expected[routeName] = KnownURL{
		URLRegex:          urlRegex,
		compiledRegex:     regexp.MustCompile(urlRegex),
		groupNamesInRegex: groupNames,
	}
	assert.True(t, assert.ObjectsAreEqualValues(expected[routeName], knownURLs[routeName]))
	//cleanup
	delete(knownURLs, routeName)
}

func TestIsKnownURL(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// verifies that the URL below are known/unknown from the controller, using its default settings
	t.Run("known URL", func(t *testing.T) {
		url := "https://openshift.io/org_foo/space_bar/plan/detail/1"
		known, patternName := isKnownURL(url)
		require.True(t, known, "URL should be known: %s", url)
		assert.Equal(t, search.HostRegistrationKeyForListWI, *patternName)
	})

	t.Run("known Demo URL", func(t *testing.T) {
		url := "https://demo.openshift.io/org_foo/space_bar/board/detail/1"
		known, patternName := isKnownURL(url)
		require.True(t, known, "URL should be known: %s", url)
		assert.Equal(t, search.HostRegistrationKeyForBoardWI, *patternName)
	})

	t.Run("unknown URL", func(t *testing.T) {
		url := "different.io/everything/v1/v2/q=1"
		known, patternName := isKnownURL(url)
		require.False(t, known, "URL should not be known: %s", url)
		assert.Nil(t, patternName)
	})
}

func TestGetSearchQueryFromURLPattern(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// getSearchQueryFromURLPattern
	// register urls
	// select pattern and pass search string
	// validate output with different scenarios like ID present not present
	urlRegex := `(?P<domain>[^/]+)/(?P<org>[^/]+)/(?P<space>[^/]+)/(?P<path>plan/detail)/(?P<number>.*)`
	routeName := "custom-test-route"
	RegisterAsKnownURL(routeName, urlRegex)
	t.Run("search with work item number in URL", func(t *testing.T) {
		// when
		searchQuery, isKnownURL := getSearchQueryFromURLString("openshift.io/userfoo/spacebar/plan/detail/100")
		// then
		assert.True(t, isKnownURL)
		assert.Equal(t, "(100:*A | openshift.io/userfoo/spacebar/plan/detail/100:*)", searchQuery)
	})

	t.Run("search without work item number in URL", func(t *testing.T) {
		// when
		searchQuery, isKnownURL := getSearchQueryFromURLString("openshift.io/userfoo/spacebar/plan/detail/")
		// then
		assert.True(t, isKnownURL)
		assert.Equal(t, "openshift.io/userfoo/spacebar/plan/detail/:*", searchQuery)
	})
	t.Run("search with invalid URL", func(t *testing.T) {
		// when
		searchQuery, isKnownURL := getSearchQueryFromURLString("openshift.io/plan/detail")
		// then
		assert.False(t, isKnownURL)
		assert.Equal(t, "openshift.io/plan/detail:*", searchQuery)
	})
	// cleanup
	delete(knownURLs, routeName)
}

func TestGetSearchQueryFromURLString(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// register few urls
	// call getSearchQueryFromURLString with different urls - both registered and non-registered

	t.Run("unknown URL pattern", func(t *testing.T) {
		// when
		searchQuery, knownURL := getSearchQueryFromURLString("abcd.something.com")
		// then
		assert.Equal(t, "abcd.something.com:*", searchQuery)
		assert.False(t, knownURL)
	})

	t.Run("known URL pattern", func(t *testing.T) {
		// given
		urlRegex := `(?P<domain>openshift.io)/(?P<path>everything)/(?P<number>\d*)`
		routeName := "custom-test-route"
		RegisterAsKnownURL(routeName, urlRegex)

		t.Run("without work item number", func(t *testing.T) {
			// when
			searchQuery, knownURL := getSearchQueryFromURLString("openshift.io/everything/")
			// then
			assert.Equal(t, "openshift.io/everything/:*", searchQuery)
			assert.True(t, knownURL)
		})

		t.Run("without work item number", func(t *testing.T) {
			// when
			searchQuery, knownURL := getSearchQueryFromURLString("openshift.io/everything/100")
			// then
			assert.Equal(t, "(100:*A | openshift.io/everything/100:*)", searchQuery)
			assert.True(t, knownURL)
		})
	})
}
