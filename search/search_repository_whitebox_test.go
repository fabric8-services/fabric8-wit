package search

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

func TestRunSearchRepositoryWhiteboxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchRepositoryWhiteboxTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

type searchRepositoryWhiteboxTest struct {
	gormsupport.DBTestSuite
	modifierID uuid.UUID
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
func (s *searchRepositoryWhiteboxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if _, c := os.LookupEnv(resource.Database); c {
		if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			ctx := migration.NewMigrationContext(context.Background())
			return migration.PopulateCommonTypes(ctx, tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
}

func (s *searchRepositoryWhiteboxTest) SetupTest() {
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "jdoe", "test")
	require.Nil(s.T(), err)
	s.modifierID = testIdentity.ID
}

type SearchTestDescriptor struct {
	wi             app.WorkItem
	searchString   string
	minimumResults int
}

func (s *searchRepositoryWhiteboxTest) TestSearchByText() {
	wir := workitem.NewWorkItemRepository(s.DB)

	testDataSet := []SearchTestDescriptor{
		{
			wi: app.WorkItem{
				Fields: map[string]interface{}{
					workitem.SystemTitle:       "test sbose title '12345678asdfgh'",
					workitem.SystemDescription: rendering.NewMarkupContentFromLegacy(`"description" for search test`),
					workitem.SystemCreator:     "sbose78",
					workitem.SystemAssignees:   []string{"pranav"},
					workitem.SystemState:       "closed",
				},
			},
			searchString:   `Sbose "deScription" '12345678asdfgh' `,
			minimumResults: 1,
		},
		{
			wi: app.WorkItem{
				Fields: map[string]interface{}{
					workitem.SystemTitle:       "add new error types in models/errors.go'",
					workitem.SystemDescription: rendering.NewMarkupContentFromLegacy(`Make sure remoteworkitem can access..`),
					workitem.SystemCreator:     "sbose78",
					workitem.SystemAssignees:   []string{"pranav"},
					workitem.SystemState:       "closed",
				},
			},
			searchString:   `models/errors.go remoteworkitem `,
			minimumResults: 1,
		},
		{
			wi: app.WorkItem{
				Fields: map[string]interface{}{
					workitem.SystemTitle:       "test sbose title '12345678asdfgh'",
					workitem.SystemDescription: rendering.NewMarkupContentFromLegacy(`"description" for search test`),
					workitem.SystemCreator:     "sbose78",
					workitem.SystemAssignees:   []string{"pranav"},
					workitem.SystemState:       "closed",
				},
			},
			searchString:   `Sbose "deScription" '12345678asdfgh' `,
			minimumResults: 1,
		},
		{
			wi: app.WorkItem{
				// will test behaviour when null fields are present. In this case, "system.description" is nil
				Fields: map[string]interface{}{
					workitem.SystemTitle:     "test nofield sbose title '12345678asdfgh'",
					workitem.SystemCreator:   "sbose78",
					workitem.SystemAssignees: []string{"pranav"},
					workitem.SystemState:     "closed",
				},
			},
			searchString:   `sbose nofield `,
			minimumResults: 1,
		},
		{
			wi: app.WorkItem{
				// will test behaviour when null fields are present. In this case, "system.description" is nil
				Fields: map[string]interface{}{
					workitem.SystemTitle:     "test should return 0 results'",
					workitem.SystemCreator:   "sbose78",
					workitem.SystemAssignees: []string{"pranav"},
					workitem.SystemState:     "closed",
				},
			},
			searchString:   `negative case `,
			minimumResults: 0,
		}, {
			wi: app.WorkItem{
				// search stirng with braces should be acceptable case
				Fields: map[string]interface{}{
					workitem.SystemTitle:     "Bug reported by administrator for input = (value)",
					workitem.SystemCreator:   "pgore",
					workitem.SystemAssignees: []string{"pranav"},
					workitem.SystemState:     "new",
				},
			},
			searchString:   `(value) `,
			minimumResults: 1,
		}, {
			wi: app.WorkItem{
				// search stirng with surrounding braces should be acceptable case
				Fields: map[string]interface{}{
					workitem.SystemTitle:     "trial for braces (pranav) {shoubhik} [aslak]",
					workitem.SystemCreator:   "pgore",
					workitem.SystemAssignees: []string{"pranav"},
					workitem.SystemState:     "new",
				},
			},
			searchString:   `(pranav) {shoubhik} [aslak] `,
			minimumResults: 1,
		},
	}

	models.Transactional(s.DB, func(tx *gorm.DB) error {

		for _, testData := range testDataSet {
			workItem := testData.wi
			searchString := testData.searchString
			minimumResults := testData.minimumResults
			workItemURLInSearchString := "http://demo.almighty.io/work-item/list/detail/"
			req := &http.Request{Host: "localhost"}
			params := url.Values{}
			ctx := goa.NewContext(context.Background(), nil, req, params)

			createdWorkItem, err := wir.Create(ctx, space.SystemSpace, workitem.SystemBug, workItem.Fields, s.modifierID)
			if err != nil {
				s.T().Fatal("Couldnt create test data")
			}

			defer wir.Delete(ctx, createdWorkItem.ID, s.modifierID)

			// create the URL and use it in the search string
			workItemURLInSearchString = workItemURLInSearchString + createdWorkItem.ID

			// had to dynamically create this since I didn't now the URL/ID of the workitem
			// till the test data was created.
			searchString = searchString + workItemURLInSearchString
			searchString = fmt.Sprintf("\"%s\"", searchString)
			s.T().Log("using search string: " + searchString)
			sr := NewGormSearchRepository(tx)
			var start, limit int = 0, 100
			workItemList, _, err := sr.SearchFullText(ctx, searchString, &start, &limit)
			if err != nil {
				s.T().Fatal("Error getting search result ", err)
			}
			searchString = strings.Trim(searchString, "\"")
			// Since this test adds test data, whether or not other workitems exist
			// there must be at least 1 search result returned.
			if len(workItemList) == minimumResults && minimumResults == 0 {
				// no point checking further, we got what we wanted.
				continue
			} else if len(workItemList) < minimumResults {
				s.T().Fatalf("At least %d search results was expected ", minimumResults)
			}

			// These keywords need a match in the textual part.
			allKeywords := strings.Fields(searchString)
			allKeywords = append(allKeywords, createdWorkItem.ID)
			//[]string{workItemURLInSearchString, createdWorkItem.ID, `"Sbose"`, `"deScription"`, `'12345678asdfgh'`}

			// These keywords need a match optionally either as URL string or ID
			optionalKeywords := []string{workItemURLInSearchString, createdWorkItem.ID}

			// We will now check the legitimacy of the search results.
			// Iterate through all search results and see whether they meet the criteria

			for _, workItemValue := range workItemList {
				s.T().Log("Found search result  ", workItemValue.ID)

				for _, keyWord := range allKeywords {

					workItemTitle := ""
					if workItemValue.Fields[workitem.SystemTitle] != nil {
						workItemTitle = strings.ToLower(workItemValue.Fields[workitem.SystemTitle].(string))
					}
					workItemDescription := ""
					if workItemValue.Fields[workitem.SystemDescription] != nil {
						descriptionField := workItemValue.Fields[workitem.SystemDescription].(rendering.MarkupContent)
						workItemDescription = strings.ToLower(descriptionField.Content)
					}
					keyWord = strings.ToLower(keyWord)

					if strings.Contains(workItemTitle, keyWord) || strings.Contains(workItemDescription, keyWord) {
						// Check if the search keyword is present as text in the title/description
						s.T().Logf("Found keyword %s in workitem %s", keyWord, workItemValue.ID)
					} else if stringInSlice(keyWord, optionalKeywords) && strings.Contains(keyWord, workItemValue.ID) {
						// If not present in title/description then it should be a URL or ID
						s.T().Logf("Found keyword %s as ID %s from the URL", keyWord, workItemValue.ID)
					} else {
						s.T().Errorf("%s neither found in title %s nor in the description: %s", keyWord, workItemTitle, workItemDescription)
					}
				}
				//defer wir.Delete(context.Background(), workItemValue.ID)
			}

		}
		return nil

	})
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

	models.Transactional(s.DB, func(tx *gorm.DB) error {
		req := &http.Request{Host: "localhost"}
		params := url.Values{}
		ctx := goa.NewContext(context.Background(), nil, req, params)

		wir := workitem.NewWorkItemRepository(tx)

		workItem := app.WorkItem{Fields: make(map[string]interface{})}

		workItem.Fields = map[string]interface{}{
			workitem.SystemTitle:       "Search Test Sbose",
			workitem.SystemDescription: rendering.NewMarkupContentFromLegacy("Description"),
			workitem.SystemCreator:     "sbose78",
			workitem.SystemAssignees:   []string{"pranav"},
			workitem.SystemState:       "closed",
		}

		createdWorkItem, err := wir.Create(ctx, space.SystemSpace, workitem.SystemBug, workItem.Fields, s.modifierID)
		if err != nil {
			s.T().Fatalf("Couldn't create test data: %+v", err)
		}
		defer wir.Delete(ctx, createdWorkItem.ID, s.modifierID)

		// Create a new workitem to have the ID in it's title. This should not come
		// up in search results

		workItem.Fields[workitem.SystemTitle] = "Search test sbose " + createdWorkItem.ID
		_, err = wir.Create(ctx, space.SystemSpace, workitem.SystemBug, workItem.Fields, s.modifierID)
		if err != nil {
			s.T().Fatalf("Couldn't create test data: %+v", err)
		}

		sr := NewGormSearchRepository(tx)

		var start, limit int = 0, 100
		searchString := "id:" + createdWorkItem.ID
		workItemList, _, err := sr.SearchFullText(ctx, searchString, &start, &limit)
		if err != nil {
			s.T().Fatal("Error gettig search result ", err)
		}

		// ID is unique, hence search result set's length should be 1
		assert.Equal(s.T(), len(workItemList), 1)
		for _, workItemValue := range workItemList {
			s.T().Log("Found search result for ID Search ", workItemValue.ID)
			assert.Equal(s.T(), createdWorkItem.ID, workItemValue.ID)
		}
		return errors.WithStack(err)
	})
}

func TestGenerateSQLSearchStringText(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	input := searchKeyword{
		id:    []string{"10", "99"},
		words: []string{"username", "title_substr", "desc_substr"},
	}
	expectedSQLParameter := "10 & 99 & username & title_substr & desc_substr"

	actualSQLParameter := generateSQLSearchInfo(input)
	assert.Equal(t, expectedSQLParameter, actualSQLParameter)
}

func TestGenerateSQLSearchStringIdOnly(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	input := searchKeyword{
		id:    []string{"10"},
		words: []string{},
	}
	expectedSQLParameter := "10"

	actualSQLParameter := generateSQLSearchInfo(input)
	assert.Equal(t, expectedSQLParameter, actualSQLParameter)
}

func TestParseSearchString(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	input := "user input for search string with some ids like id:99 and id:400 but this is not id like 800"
	op, _ := parseSearchString(input)
	expectedSearchRes := searchKeyword{
		id:    []string{"99:*A", "400:*A"},
		words: []string{"user:*", "input:*", "for:*", "search:*", "string:*", "with:*", "some:*", "ids:*", "like:*", "and:*", "but:*", "this:*", "is:*", "not:*", "id:*", "like:*", "800:*"},
	}
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
			id:    nil,
			words: []string{"(100:* | demo.almighty.io/work-item/list/detail/100:*)"},
		},
	}, {
		query: "http://demo.almighty.io/work-item/board/detail/100",
		expected: searchKeyword{
			id:    nil,
			words: []string{"(100:* | demo.almighty.io/work-item/board/detail/100:*)"},
		},
	}}

	for _, input := range inputSet {
		op, _ := parseSearchString(input.query)
		assert.True(t, assert.ObjectsAreEqualValues(input.expected, op))
	}
}

func TestParseSearchStringURLWithouID(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	inputSet := []searchTestData{{
		query: "http://demo.almighty.io/work-item/list/detail/",
		expected: searchKeyword{
			id:    nil,
			words: []string{"demo.almighty.io/work-item/list/detail:*"},
		},
	}, {
		query: "http://demo.almighty.io/work-item/board/detail/",
		expected: searchKeyword{
			id:    nil,
			words: []string{"demo.almighty.io/work-item/board/detail:*"},
		},
	}}

	for _, input := range inputSet {
		op, _ := parseSearchString(input.query)
		assert.True(t, assert.ObjectsAreEqualValues(input.expected, op))
	}

}

func TestParseSearchStringDifferentURL(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	input := "http://demo.redhat.io"
	op, _ := parseSearchString(input)
	expectedSearchRes := searchKeyword{
		id:    nil,
		words: []string{"demo.redhat.io:*"},
	}
	assert.True(t, assert.ObjectsAreEqualValues(expectedSearchRes, op))
}

func TestParseSearchStringCombination(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// do combination of ID, full text and URLs
	// check if it works as expected.
	input := "http://general.url.io http://demo.almighty.io/work-item/list/detail/100 id:300 golang book and           id:900 \t \n unwanted"
	op, _ := parseSearchString(input)
	expectedSearchRes := searchKeyword{
		id:    []string{"300:*A", "900:*A"},
		words: []string{"general.url.io:*", "(100:* | demo.almighty.io/work-item/list/detail/100:*)", "golang:*", "book:*", "and:*", "unwanted:*"},
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
