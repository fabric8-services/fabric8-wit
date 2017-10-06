package search

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
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
	// While registering URLs do not include protocol because it will be removed before scanning starts
	// Please do not include trailing slashes because it will be removed before scanning starts
	RegisterAsKnownURL("test-work-item-list-details", `(?P<domain>demo.openshift.io)(?P<path>/work-item/list/detail/)(?P<id>\d*)`)
	RegisterAsKnownURL("test-work-item-board-details", `(?P<domain>demo.openshift.io)(?P<path>/work-item/board/detail/)(?P<id>\d*)`)

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

func (s *searchRepositoryWhiteboxTest) setupTestDataSet() ([]SearchTestDescriptor, *tf.TestFixture) {
	// given
	testDataSet := []SearchTestDescriptor{
		{
			fields: map[string]interface{}{
				workitem.SystemTitle:       "test sbose title '12345678asdfgh'",
				workitem.SystemDescription: rendering.NewMarkupContentFromLegacy(`"description" for search test`),
				workitem.SystemCreator:     "sbose78",
				workitem.SystemAssignees:   []string{"pranav"},
				workitem.SystemState:       "closed",
			},
			searchString:   `Sbose "deScription" '12345678asdfgh' `,
			minimumResults: 1,
		},
		{
			fields: map[string]interface{}{
				workitem.SystemTitle:       "add new error types in models/errors.go'",
				workitem.SystemDescription: rendering.NewMarkupContentFromLegacy(`Make sure remoteworkitem can access..`),
				workitem.SystemCreator:     "sbose78",
				workitem.SystemAssignees:   []string{"pranav"},
				workitem.SystemState:       "closed",
			},
			searchString:   `models/errors.go remoteworkitem `,
			minimumResults: 1,
		},
		{
			fields: map[string]interface{}{
				workitem.SystemTitle:       "test sbose title '12345678asdfgh'",
				workitem.SystemDescription: rendering.NewMarkupContentFromLegacy(`"description" for search test`),
				workitem.SystemCreator:     "sbose78",
				workitem.SystemAssignees:   []string{"pranav"},
				workitem.SystemState:       "closed",
			},
			searchString:   `Sbose "deScription" '12345678asdfgh' `,
			minimumResults: 1,
		},
		{
			// will test behaviour when null fields are present. In this case, "system.description" is nil
			fields: map[string]interface{}{
				workitem.SystemTitle:     "test nofield sbose title '12345678asdfgh'",
				workitem.SystemCreator:   "sbose78",
				workitem.SystemAssignees: []string{"pranav"},
				workitem.SystemState:     "closed",
			},
			searchString:   `sbose nofield `,
			minimumResults: 1,
		},
		{
			// will test behaviour when null fields are present. In this case, "system.description" is nil
			fields: map[string]interface{}{
				workitem.SystemTitle:     "test should return 0 results'",
				workitem.SystemCreator:   "sbose78",
				workitem.SystemAssignees: []string{"pranav"},
				workitem.SystemState:     "closed",
			},
			searchString:   `negative case `,
			minimumResults: 0,
		}, {
			// search stirng with braces should be acceptable case
			fields: map[string]interface{}{
				workitem.SystemTitle:     "Bug reported by administrator for input = (value)",
				workitem.SystemCreator:   "pgore",
				workitem.SystemAssignees: []string{"pranav"},
				workitem.SystemState:     "new",
			},
			searchString:   `(value) `,
			minimumResults: 1,
		}, {
			// search stirng with surrounding braces should be acceptable case
			fields: map[string]interface{}{
				workitem.SystemTitle:     "trial for braces (pranav) {shoubhik} [aslak]",
				workitem.SystemCreator:   "pgore",
				workitem.SystemAssignees: []string{"pranav"},
				workitem.SystemState:     "new",
			},
			searchString:   `(pranav) {shoubhik} [aslak] `,
			minimumResults: 1,
		},
	}
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Identities(2), tf.WorkItems(len(testDataSet), func(fxt *tf.TestFixture, idx int) error {
		fxt.WorkItems[idx].SpaceID = fxt.Spaces[0].ID
		fxt.WorkItems[idx].Type = fxt.WorkItemTypes[0].ID
		fxt.WorkItems[idx].Fields = testDataSet[idx].fields
		fxt.WorkItems[idx].Fields[workitem.SystemCreator] = fxt.Identities[0].ID.String()
		fxt.WorkItems[idx].Fields[workitem.SystemAssignees] = []string{fxt.Identities[1].ID.String()}
		return nil
	}))
	return testDataSet, fxt
}

// TestSearchByText verifies search on title or description
func (s *searchRepositoryWhiteboxTest) TestSearchByText() {
	// given
	testDataSet, fxt := s.setupTestDataSet()
	//
	for idx, testData := range testDataSet {
		minimumResults := testData.minimumResults
		req := &http.Request{Host: "localhost"}
		params := url.Values{}
		ctx := goa.NewContext(context.Background(), nil, req, params)
		// had to dynamically create this since I didn't now the URL/ID of the workitem
		// till the test data was created.
		searchString := testData.searchString
		workItemURLInSearchString := fmt.Sprintf("%s%d", "http://demo.openshift.io/work-item/list/detail/", fxt.WorkItems[idx].Number)
		searchString = fmt.Sprintf("\"%s %s\"", searchString, workItemURLInSearchString)
		sr := NewGormSearchRepository(s.DB)
		var start, limit int = 0, 100
		spaceID := fxt.Spaces[0].ID.String()
		workItemList, _, err := sr.SearchFullText(ctx, searchString, &start, &limit, &spaceID)
		require.Nil(s.T(), err, "failed to get search result")
		searchString = strings.Trim(searchString, "\"")
		s.T().Logf("TestData #%d: using search string: %s -> %d matches", (idx + 1), searchString, len(workItemList))
		// Since this test adds test data, whether or not other workitems exist
		// there must be at least 1 search result returned.
		if len(workItemList) == minimumResults && minimumResults == 0 {
			// no point checking further, we got what we wanted.
			continue
		} else if len(workItemList) < minimumResults {
			s.T().Fatalf("At least %d search result(s) was|were expected ", minimumResults)
		}

		// These keywords need a match in the textual part.
		allKeywords := strings.Fields(searchString)
		// These keywords need a match optionally either as URL string or ID		 +				keyWord = strings.ToLower(keyWord)
		optionalKeywords := []string{workItemURLInSearchString, strconv.Itoa(fxt.WorkItems[idx].Number)}
		// We will now check the legitimacy of the search results.
		// Iterate through all search results and see whether they meet the criteria
		for _, workItemValue := range workItemList {
			s.T().Logf("Examining workitem id=`%v` number=`%d` using keywords %v", workItemValue.ID, workItemValue.Number, allKeywords)
			for _, keyWord := range allKeywords {
				keyWord = strings.ToLower(keyWord)
				s.T().Logf("Verifying workitem id=`%v` number=`%d` for keyword `%s`...", workItemValue.ID, workItemValue.Number, keyWord)
				workItemTitle := ""
				if workItemValue.Fields[workitem.SystemTitle] != nil {
					workItemTitle = strings.ToLower(workItemValue.Fields[workitem.SystemTitle].(string))
				}
				workItemDescription := ""
				if workItemValue.Fields[workitem.SystemDescription] != nil {
					descriptionField := workItemValue.Fields[workitem.SystemDescription].(rendering.MarkupContent)
					workItemDescription = strings.ToLower(descriptionField.Content)
				}
				assert.True(s.T(),
					strings.Contains(workItemTitle, keyWord) || strings.Contains(workItemDescription, keyWord) ||
						(stringInSlice(keyWord, optionalKeywords) && strings.Contains(keyWord, strconv.Itoa(workItemValue.Number))),
					"`%s` neither found in title `%s` nor in the description `%s` for workitem #%d", keyWord, workItemTitle, workItemDescription, workItemValue.Number)
			}
		}

	}
}

// TestSearchByText verifies search on number
func (s *searchRepositoryWhiteboxTest) TestSearchByNumber() {
	// given
	testDataSet, fxt := s.setupTestDataSet()
	//
	for idx, testData := range testDataSet {
		number := fxt.WorkItems[idx].Number
		minimumResults := testData.minimumResults
		req := &http.Request{Host: "localhost"}
		params := url.Values{}
		ctx := goa.NewContext(context.Background(), nil, req, params)

		// had to dynamically create this since I didn't now the URL/ID of the workitem
		// till the test data was created.
		searchString := fmt.Sprintf("\"number:%d\"", number)
		sr := NewGormSearchRepository(s.DB)
		var start, limit int = 0, 100
		spaceID := fxt.Spaces[0].ID.String()
		workItemList, _, err := sr.SearchFullText(ctx, searchString, &start, &limit, &spaceID)
		require.Nil(s.T(), err, "failed to get search result")
		searchString = strings.Trim(searchString, "\"")
		s.T().Logf("TestData #%d: using search string: %s -> %d matches", idx, searchString, len(workItemList))
		// Since this test adds test data, whether or not other workitems exist
		// there must be at least 1 search result returned.
		if len(workItemList) == minimumResults && minimumResults == 0 {
			// no point checking further, we got what we wanted.
			continue
		} else if len(workItemList) < minimumResults {
			s.T().Fatalf("At least %d search result(s) was|were expected ", minimumResults)
		}

		// These keywords need a match in the textual part.
		allKeywords := strings.Fields(searchString)
		//[]string{createdWorkItem.ID, `"Sbose"`, `"deScription"`, `'12345678asdfgh'`}

		// We will now check the legitimacy of the search results.
		// Iterate through all search results and see whether they meet the criteria
		for _, workItemValue := range workItemList {
			s.T().Logf("Examining workitem id=`%v` number=`%d` using keywords %v", workItemValue.ID, workItemValue.Number, allKeywords)
			for _, keyWord := range allKeywords {
				keyWord = strings.ToLower(keyWord)
				s.T().Logf("Verifying workitem id=`%v` number=`%d` for keyword `%s`...", workItemValue.ID, workItemValue.Number, keyWord)
				assert.Equal(s.T(), number, workItemValue.Number,
					"workitem #%d did not have the expected number", workItemValue.Number, number)
			}
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

func (s *searchRepositoryWhiteboxTest) TestSearchByID() {
	// given
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)
	// create 2 work items, the second one having the number of the first one in its title
	fxt, err := tf.NewFixture(s.DB, tf.WorkItems(2, func(fxt *tf.TestFixture, idx int) error {
		fxt.WorkItems[idx].Type = workitem.SystemBug
		fxt.WorkItems[idx].Fields = map[string]interface{}{
			workitem.SystemTitle:       "Search Test Sbose",
			workitem.SystemDescription: rendering.NewMarkupContentFromLegacy("Description"),
			workitem.SystemCreator:     s.modifierID.String(),
			workitem.SystemAssignees:   []string{"pranav"},
			workitem.SystemState:       "closed",
		}
		fxt.WorkItems[idx].SpaceID = fxt.Spaces[0].ID
		// for the second work item, use the number of the first work item
		if idx == 1 {
			fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "Search Test Sbose" + strconv.Itoa(fxt.WorkItems[0].Number)
		}
		return nil
	}))
	require.Nil(s.T(), err, "Couldn't create test data")
	sr := NewGormSearchRepository(s.DB)
	// when
	var start, limit int = 0, 100
	searchString := "number:" + strconv.Itoa(fxt.WorkItems[0].Number)
	spaceID := fxt.Spaces[0].ID.String() // make sure the search is limited to the space to avoid collision with other existing data
	workItemList, _, err := sr.SearchFullText(ctx, searchString, &start, &limit, &spaceID)
	// then
	require.Nil(s.T(), err)
	// Number is unique, hence search result set's length should be 1
	require.Equal(s.T(), len(workItemList), 1)
	s.T().Log("Found search result for ID Search ", workItemList[0].ID)
	assert.Equal(s.T(), fxt.WorkItems[0].ID, workItemList[0].ID)
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
		query: "http://demo.almighty.io/work-item/list/detail/100",
		expected: searchKeyword{
			number: nil,
			words:  []string{"(100:* | demo.almighty.io/work-item/list/detail/100:*)"},
		},
	}, {
		query: "http://demo.almighty.io/work-item/board/detail/100",
		expected: searchKeyword{
			number: nil,
			words:  []string{"(100:* | demo.almighty.io/work-item/board/detail/100:*)"},
		},
	}}

	for _, input := range inputSet {
		op, _ := parseSearchString(context.Background(), input.query)
		assert.True(t, assert.ObjectsAreEqualValues(input.expected, op))
	}
}

func TestParseSearchStringURLWithouID(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	inputSet := []searchTestData{{
		query: "http://demo.almighty.io/work-item/list/detail/",
		expected: searchKeyword{
			number: nil,
			words:  []string{"demo.almighty.io/work-item/list/detail:*"},
		},
	}, {
		query: "http://demo.almighty.io/work-item/board/detail/",
		expected: searchKeyword{
			number: nil,
			words:  []string{"demo.almighty.io/work-item/board/detail:*"},
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
	input := "http://general.url.io http://demo.almighty.io/work-item/list/detail/100 number:300 golang book and           number:900 \t \n unwanted"
	op, _ := parseSearchString(context.Background(), input)
	expectedSearchRes := searchKeyword{
		number: []string{"300:*A", "900:*A"},
		words:  []string{"general.url.io:*", "(100:* | demo.almighty.io/work-item/list/detail/100:*)", "golang:*", "book:*", "and:*", "unwanted:*"},
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
	assert.Equal(t, "(100:* | google.me.io/everything/100:*)", searchQuery)

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
	assert.Equal(t, "(100:* | google.me.io/everything/100:*)", searchQuery)
}
