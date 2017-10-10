package search

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	_ "github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func init() {
	// While registering URLs do not include protocol because it will be removed before scanning starts
	// Please do not include trailing slashes because it will be removed before scanning starts
	RegisterAsKnownURL("test-work-item-list-details", `(?P<domain>demo.openshift.io)(?P<path>/work-item/list/detail/)(?P<id>\d*)`)
	RegisterAsKnownURL("test-work-item-board-details", `(?P<domain>demo.openshift.io)(?P<path>/work-item/board/detail/)(?P<id>\d*)`)
}

func TestRunSearchRepositoryWhiteboxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchRepositoryWhiteboxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type searchRepositoryWhiteboxTest struct {
	gormtestsupport.DBTestSuite
	modifierID uuid.UUID
}

func (s *searchRepositoryWhiteboxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
}

func (s *searchRepositoryWhiteboxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "jdoe", "test")
	require.Nil(s.T(), err)
	s.modifierID = testIdentity.ID
}

type SearchTestDescriptor struct {
	fields         map[string]interface{}
	searchString   string
	minimumResults int
}

func (s *searchRepositoryWhiteboxTest) TestSearch() {

	s.T().Run("Search accross title and description", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "test sbose title '12345678asdfgh'"
			fxt.WorkItems[idx].Fields[workitem.SystemDescription] = rendering.NewMarkupContentFromLegacy(`"description" for search test`)
			return nil
		}))
		// when
		searchQuery := `Sbose "deScription" '12345678asdfgh'`
		searchResults, err := s.searchFor(fxt.Spaces[0].ID, searchQuery)
		// then
		require.Nil(t, err)
		verify(t, searchQuery, searchResults, 1)
	})

	s.T().Run("Search accross title and description undefined", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "test nofield sbose title '12345678asdfgh'"
			return nil
		}))
		// when
		searchQuery := `sbose nofield`
		searchResults, err := s.searchFor(fxt.Spaces[0].ID, searchQuery)
		// then
		require.Nil(t, err)
		verify(t, searchQuery, searchResults, 1)
	})

	s.T().Run("Search accross title with slash", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "add new error types in models/errors.go'"
			fxt.WorkItems[idx].Fields[workitem.SystemDescription] = rendering.NewMarkupContentFromLegacy(`Make sure remoteworkitem can access..`)
			return nil
		}))
		// when
		searchQuery := `models/errors.go remoteworkitem`
		searchResults, err := s.searchFor(fxt.Spaces[0].ID, searchQuery)
		// then
		require.Nil(t, err)
		verify(t, searchQuery, searchResults, 1)
	})

	s.T().Run("Search accross title with braces", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "Bug reported by administrator for input = (value)"
			return nil
		}))
		// when
		searchQuery := `(value)`
		searchResults, err := s.searchFor(fxt.Spaces[0].ID, searchQuery)
		// then
		require.Nil(t, err)
		verify(t, searchQuery, searchResults, 1)

	})

	s.T().Run("Search accross title with braces and brackets", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "trial for braces (pranav) {shoubhik} [aslak]"
			return nil
		}))
		// when
		searchQuery := `(pranav) {shoubhik} [aslak]`
		searchResults, err := s.searchFor(fxt.Spaces[0].ID, searchQuery)
		// then
		require.Nil(t, err)
		verify(t, searchQuery, searchResults, 1)

	})

	s.T().Run("Search accross title and description undefined and no match", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "test should return 0 results'"
			return nil
		}))
		// when
		searchQuery := `negative case`
		searchResults, err := s.searchFor(fxt.Spaces[0].ID, searchQuery)
		// then
		require.Nil(t, err)
		verify(t, searchQuery, searchResults, 0)
	})

	s.T().Run("Search by number", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "test nofield sbose title '12345678asdfgh'"
			return nil
		}))
		// when
		searchQuery := fmt.Sprintf("number:%d", fxt.WorkItems[0].Number)
		searchResults, err := s.searchFor(fxt.Spaces[0].ID, searchQuery)
		// then
		require.Nil(t, err)
		require.Len(t, searchResults, 1)
		assert.Equal(t, fxt.WorkItems[0].ID, searchResults[0].ID)
	})

	s.T().Run("Search by URL", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "test nofield sbose title '12345678asdfgh'"
			return nil
		}))
		// when
		searchQuery := fmt.Sprintf("%s%d", "http://demo.openshift.io/work-item/list/detail/", fxt.WorkItems[0].Number)
		searchResults, err := s.searchFor(fxt.Spaces[0].ID, searchQuery)
		// then
		require.Nil(t, err)
		require.Len(t, searchResults, 1)
		assert.Equal(t, fxt.WorkItems[0].ID, searchResults[0].ID)
	})

}

func (s *searchRepositoryWhiteboxTest) searchFor(spaceID uuid.UUID, searchQuery string) ([]workitem.WorkItem, error) {
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)
	sr := NewGormSearchRepository(s.DB)
	var start, limit int = 0, 100
	spaceIDStr := spaceID.String()
	workItemList, _, err := sr.SearchFullText(ctx, fmt.Sprintf("\"%s\"", searchQuery), &start, &limit, &spaceIDStr)
	return workItemList, err
}

func verify(t *testing.T, searchQuery string, searchResults []workitem.WorkItem, expectedCount int) {
	// Since this test adds test data, whether or not other workitems exist
	// there must be at least 1 search result returned.
	if len(searchResults) == expectedCount && expectedCount == 0 {
		// no point checking further, we got what we wanted.
		return
	}
	require.Equal(t, expectedCount, len(searchResults), "invalid number of results in the search")

	// These keywords need a match in the textual part.
	allKeywords := strings.Fields(searchQuery)
	// These keywords need a match optionally either as URL string or ID		 +				keyWord = strings.ToLower(keyWord)
	// optionalKeywords := []string{workItemURLInSearchString, strconv.Itoa(fxt.WorkItems[idx].Number)}
	// We will now check the legitimacy of the search results.
	// Iterate through all search results and see whether they meet the criteria
	for _, searchResult := range searchResults {
		t.Logf("Examining workitem id=`%v` number=`%d` using keywords %v", searchResult.ID, searchResult.Number, allKeywords)
		for _, keyWord := range allKeywords {
			keyWord = strings.ToLower(keyWord)
			t.Logf("Verifying workitem id=`%v` number=`%d` for keyword `%s`...", searchResult.ID, searchResult.Number, keyWord)
			workItemTitle := ""
			if searchResult.Fields[workitem.SystemTitle] != nil {
				workItemTitle = strings.ToLower(searchResult.Fields[workitem.SystemTitle].(string))
			}
			workItemDescription := ""
			if searchResult.Fields[workitem.SystemDescription] != nil {
				descriptionField := searchResult.Fields[workitem.SystemDescription].(rendering.MarkupContent)
				workItemDescription = strings.ToLower(descriptionField.Content)
			}
			assert.True(t,
				strings.Contains(workItemTitle, keyWord) || strings.Contains(workItemDescription, keyWord),
				"`%s` neither found in title `%s` nor in the description `%s` for workitem #%d", keyWord, workItemTitle, workItemDescription, searchResult.Number)
		}
	}
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func TestGenerateSQLSearchStringText(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	input := searchKeyword{
		number: []string{"10", "99"},
		words:  []string{"username", "title_substr", "desc_substr"},
	}
	expectedSQLParameter := "10 & 99 & username & title_substr & desc_substr"

	actualSQLParameter := generateSQLSearchInfo(input)
	assert.Equal(t, expectedSQLParameter, actualSQLParameter)
}

func TestGenerateSQLSearchStringIdOnly(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	input := searchKeyword{
		number: []string{"10"},
		words:  []string{},
	}
	expectedSQLParameter := "10"

	actualSQLParameter := generateSQLSearchInfo(input)
	assert.Equal(t, expectedSQLParameter, actualSQLParameter)
}

func TestParseSearchString(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	input := "user input for search string with some ids like number:99 and number:400 but this is not id like 800"
	op, _ := parseSearchString(context.Background(), input)
	expectedSearchRes := searchKeyword{
		number: []string{"99:*A", "400:*A"},
		words:  []string{"user:*", "input:*", "for:*", "search:*", "string:*", "with:*", "some:*", "ids:*", "like:*", "and:*", "but:*", "this:*", "is:*", "not:*", "id:*", "like:*", "800:*"},
	}
	t.Log("Parsed search string: ", op)
	assert.True(t, assert.ObjectsAreEqualValues(expectedSearchRes, op))
}

type searchTestData struct {
	query    string
	expected searchKeyword
}

func TestParseSearchStringURL(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	inputSet := []searchTestData{{
		query: "http://demo.openshift.io/work-item/list/detail/100",
		expected: searchKeyword{
			number: nil,
			words:  []string{"(100:*A | demo.openshift.io/work-item/list/detail/100:*)"},
		},
	}, {
		query: "http://demo.openshift.io/work-item/board/detail/100",
		expected: searchKeyword{
			number: nil,
			words:  []string{"(100:*A | demo.openshift.io/work-item/board/detail/100:*)"},
		},
	}}

	for _, input := range inputSet {
		op, _ := parseSearchString(context.Background(), input.query)
		assert.Equal(t, input.expected, op)
	}
}

func TestParseSearchStringURLWithouID(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	inputSet := []searchTestData{{
		query: "http://demo.openshift.io/work-item/list/detail/",
		expected: searchKeyword{
			number: nil,
			words:  []string{"demo.openshift.io/work-item/list/detail:*"},
		},
	}, {
		query: "http://demo.openshift.io/work-item/board/detail/",
		expected: searchKeyword{
			number: nil,
			words:  []string{"demo.openshift.io/work-item/board/detail:*"},
		},
	}}

	for _, input := range inputSet {
		op, _ := parseSearchString(context.Background(), input.query)
		assert.True(t, assert.ObjectsAreEqualValues(input.expected, op))
	}

}

func TestParseSearchStringDifferentURL(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	input := "http://demo.redhat.io"
	op, _ := parseSearchString(context.Background(), input)
	expectedSearchRes := searchKeyword{
		number: nil,
		words:  []string{"demo.redhat.io:*"},
	}
	assert.True(t, assert.ObjectsAreEqualValues(expectedSearchRes, op))
}

func TestParseSearchStringCombination(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// do combination of ID, full text and URLs
	// check if it works as expected.
	input := "http://general.url.io http://demo.openshift.io/work-item/list/detail/100 number:300 golang book and           number:900 \t \n unwanted"
	op, _ := parseSearchString(context.Background(), input)
	expectedSearchRes := searchKeyword{
		number: []string{"300:*A", "900:*A"},
		words:  []string{"general.url.io:*", "(100:*A | demo.openshift.io/work-item/list/detail/100:*)", "golang:*", "book:*", "and:*", "unwanted:*"},
	}
	assert.True(t, assert.ObjectsAreEqualValues(expectedSearchRes, op))
}

func TestRegisterAsKnownURL(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// build 2 fake urls and cross check against RegisterAsKnownURL
	urlRegex := `(?P<domain>google.me.io)(?P<path>/everything/)(?P<param>.*)`
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
	// register few URLs and cross check is knwon or not one by one
	urlRegex := `(?P<domain>google.me.io)(?P<path>/everything/)(?P<param>.*)`
	routeName := "custom-test-route"
	RegisterAsKnownURL(routeName, urlRegex)
	known, patternName := isKnownURL("google.me.io/everything/v1/v2/q=1")
	assert.True(t, known)
	assert.Equal(t, routeName, patternName)

	known, patternName = isKnownURL("google.different.io/everything/v1/v2/q=1")
	assert.False(t, known)
	assert.Equal(t, "", patternName)

	// cleanup
	delete(knownURLs, routeName)
}

func TestGetSearchQueryFromURLPattern(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// getSearchQueryFromURLPattern
	// register urls
	// select pattern and pass search string
	// validate output with different scenarios like ID present not present
	urlRegex := `(?P<domain>google.me.io)(?P<path>/everything/)(?P<id>\d*)`
	routeName := "custom-test-route"
	RegisterAsKnownURL(routeName, urlRegex)

	searchQuery := getSearchQueryFromURLPattern(routeName, "google.me.io/everything/100")
	assert.Equal(t, "(100:*A | google.me.io/everything/100:*)", searchQuery)

	searchQuery = getSearchQueryFromURLPattern(routeName, "google.me.io/everything/")
	assert.Equal(t, "google.me.io/everything/:*", searchQuery)

	// cleanup
	delete(knownURLs, routeName)
}

func TestGetSearchQueryFromURLString(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// register few urls
	// call getSearchQueryFromURLString with different urls - both registered and non-registered
	searchQuery := getSearchQueryFromURLString("abcd.something.com")
	assert.Equal(t, "abcd.something.com:*", searchQuery)

	urlRegex := `(?P<domain>google.me.io)(?P<path>/everything/)(?P<id>\d*)`
	routeName := "custom-test-route"
	RegisterAsKnownURL(routeName, urlRegex)

	searchQuery = getSearchQueryFromURLString("google.me.io/everything/")
	assert.Equal(t, "google.me.io/everything/:*", searchQuery)

	searchQuery = getSearchQueryFromURLString("google.me.io/everything/100")
	assert.Equal(t, "(100:*A | google.me.io/everything/100:*)", searchQuery)
}
