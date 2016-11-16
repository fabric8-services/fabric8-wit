package search

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

var DB *gorm.DB

func TestMain(m *testing.M) {
	var err error

	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	if _, c := os.LookupEnv(resource.Database); c {
		DB, err = gorm.Open("postgres", configuration.GetPostgresConfigString())
		if err != nil {
			panic("Failed to connect database: " + err.Error())
		}
		defer DB.Close()
	}
	os.Exit(m.Run())
}

type SearchTestDescriptor struct {
	wi             app.WorkItem
	searchString   string
	minimumResults int
}

func TestSearchByText(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.Database)

	wir := models.NewWorkItemRepository(DB)

	testDataSet := []SearchTestDescriptor{
		{
			wi: app.WorkItem{
				Fields: map[string]interface{}{
					models.SystemTitle:       "test sbose title '12345678asdfgh'",
					models.SystemDescription: `"description" for search test`,
					models.SystemCreator:     "sbose78",
					models.SystemAssignee:    "pranav",
					models.SystemState:       "closed",
				},
			},
			searchString:   `Sbose "deScription" '12345678asdfgh' `,
			minimumResults: 1,
		},
		{
			wi: app.WorkItem{
				Fields: map[string]interface{}{
					models.SystemTitle:       "add new error types in models/errors.go'",
					models.SystemDescription: `Make sure remoteworkitem can access..`,
					models.SystemCreator:     "sbose78",
					models.SystemAssignee:    "pranav",
					models.SystemState:       "closed",
				},
			},
			searchString:   `models/errors.go remoteworkitem `,
			minimumResults: 1,
		},
		{
			wi: app.WorkItem{
				Fields: map[string]interface{}{
					models.SystemTitle:       "test sbose title '12345678asdfgh'",
					models.SystemDescription: `"description" for search test`,
					models.SystemCreator:     "sbose78",
					models.SystemAssignee:    "pranav",
					models.SystemState:       "closed",
				},
			},
			searchString:   `Sbose "deScription" '12345678asdfgh' `,
			minimumResults: 1,
		},
		{
			wi: app.WorkItem{
				// will test behaviour when null fields are present. In this case, "system.description" is nil
				Fields: map[string]interface{}{
					models.SystemTitle:    "test nofield sbose title '12345678asdfgh'",
					models.SystemCreator:  "sbose78",
					models.SystemAssignee: "pranav",
					models.SystemState:    "closed",
				},
			},
			searchString:   `sbose nofield `,
			minimumResults: 1,
		},
		{
			wi: app.WorkItem{
				// will test behaviour when null fields are present. In this case, "system.description" is nil
				Fields: map[string]interface{}{
					models.SystemTitle:    "test should return 0 results'",
					models.SystemCreator:  "sbose78",
					models.SystemAssignee: "pranav",
					models.SystemState:    "closed",
				},
			},
			searchString:   `negative case `,
			minimumResults: 0,
		},
	}

	models.Transactional(DB, func(tx *gorm.DB) error {

		for _, testData := range testDataSet {
			workItem := testData.wi
			searchString := testData.searchString
			minimumResults := testData.minimumResults
			workItemURLInSearchString := "http://demo.almighty.io/work-item-list/detail/"

			createdWorkItem, err := wir.Create(context.Background(), models.SystemBug, workItem.Fields, account.TestIdentity.ID.String())
			if err != nil {
				t.Fatal("Couldnt create test data")
			}

			defer wir.Delete(context.Background(), createdWorkItem.ID)

			// create the URL and use it in the search string
			workItemURLInSearchString = workItemURLInSearchString + createdWorkItem.ID

			// had to dynamically create this since I didn't now the URL/ID of the workitem
			// till the test data was created.
			searchString = searchString + workItemURLInSearchString
			searchString = fmt.Sprintf("\"%s\"", searchString)
			t.Log("using search string: " + searchString)
			sr := NewGormSearchRepository(tx)
			var start, limit int = 0, 100
			workItemList, _, err := sr.SearchFullText(context.Background(), searchString, &start, &limit)
			if err != nil {
				t.Fatal("Error getting search result ", err)
			}
			searchString = strings.Trim(searchString, "\"")
			// Since this test adds test data, whether or not other workitems exist
			// there must be at least 1 search result returned.
			if len(workItemList) == minimumResults && minimumResults == 0 {
				// no point checking further, we got what we wanted.
				continue
			} else if len(workItemList) < minimumResults {
				t.Fatalf("At least %d search results was expected ", minimumResults)
			}

			// These keywords need a match in the textual part.
			allKeywords := strings.Fields(searchString)
			allKeywords = append(allKeywords, createdWorkItem.ID)
			//[]string{workItemURLInSearchString, createdWorkItem.ID, `"Sbose"`, `"deScription"`, `'12345678asdfgh'`}

			// These keywords need a match optionally either as URL string or ID
			optionalKeywords := []string{workItemURLInSearchString, createdWorkItem.ID}

			// We will now check the legitimacy of the search results.
			// Iterate through all search results and see whether they meet the critera

			for _, workItemValue := range workItemList {
				t.Log("Found search result  ", workItemValue.ID)

				for _, keyWord := range allKeywords {

					workItemTitle := ""
					if workItemValue.Fields[models.SystemTitle] != nil {
						workItemTitle = strings.ToLower(workItemValue.Fields[models.SystemTitle].(string))
					}
					workItemDescription := ""
					if workItemValue.Fields[models.SystemDescription] != nil {
						workItemDescription = strings.ToLower(workItemValue.Fields[models.SystemDescription].(string))
					}
					keyWord = strings.ToLower(keyWord)

					if strings.Contains(workItemTitle, keyWord) || strings.Contains(workItemDescription, keyWord) {
						// Check if the search keyword is present as text in the title/description
						t.Logf("Found keyword %s in workitem %s", keyWord, workItemValue.ID)
					} else if stringInSlice(keyWord, optionalKeywords) && strings.Contains(keyWord, workItemValue.ID) {
						// If not present in title/description then it should be a URL or ID
						t.Logf("Found keyword %s as ID %s from the URL", keyWord, workItemValue.ID)
					} else {
						t.Errorf("%s neither found in title %s nor in the description: %s", keyWord, workItemTitle, workItemDescription)
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

func TestSearchByID(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.Database)
	wir := models.NewWorkItemRepository(DB)

	models.Transactional(DB, func(tx *gorm.DB) error {

		workItem := app.WorkItem{Fields: make(map[string]interface{})}

		workItem.Fields = map[string]interface{}{
			models.SystemTitle:       "Search Test Sbose",
			models.SystemDescription: "Description",
			models.SystemCreator:     "sbose78",
			models.SystemAssignee:    "pranav",
			models.SystemState:       "closed",
		}

		createdWorkItem, err := wir.Create(context.Background(), models.SystemBug, workItem.Fields, account.TestIdentity.ID.String())
		if err != nil {
			t.Fatal("Couldnt create test data")
		}
		defer wir.Delete(context.Background(), createdWorkItem.ID)

		// Create a new workitem to have the ID in it's title. This should not come
		// up in search results

		workItem.Fields[models.SystemTitle] = "Search test sbose " + createdWorkItem.ID
		_, err = wir.Create(context.Background(), models.SystemBug, workItem.Fields, account.TestIdentity.ID.String())
		if err != nil {
			t.Fatal("Couldnt create test data")
		}

		sr := NewGormSearchRepository(tx)

		var start, limit int = 0, 100
		searchString := "id:" + createdWorkItem.ID
		workItemList, _, err := sr.SearchFullText(context.Background(), searchString, &start, &limit)
		if err != nil {
			t.Fatal("Error gettig search result ", err)
		}

		// ID is unique, hence search result set's length should be 1
		assert.Equal(t, len(workItemList), 1)
		for _, workItemValue := range workItemList {
			t.Log("Found search result for ID Search ", workItemValue.ID)
			assert.Equal(t, createdWorkItem.ID, workItemValue.ID)
		}
		return err
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

func TestParseSearchStringURL(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	input := "http://demo.almighty.io/detail/100"
	op, _ := parseSearchString(input)

	expectedSearchRes := searchKeyword{
		id:    nil,
		words: []string{"(100:* | demo.almighty.io/work-item-list/detail/100:*)"},
	}

	assert.True(t, assert.ObjectsAreEqualValues(expectedSearchRes, op))
}

func TestParseSearchStringURLWithouID(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	input := "http://demo.almighty.io/detail/"
	op, _ := parseSearchString(input)

	expectedSearchRes := searchKeyword{
		id:    nil,
		words: []string{"demo.almighty.io/work-item-list/detail:*"},
	}

	assert.True(t, assert.ObjectsAreEqualValues(expectedSearchRes, op))
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
	input := "http://general.url.io http://demo.almighty.io/detail/100 id:300 golang book and           id:900 \t \n unwanted"
	op, _ := parseSearchString(input)
	expectedSearchRes := searchKeyword{
		id:    []string{"300:*A", "900:*A"},
		words: []string{"general.url.io:*", "(100:* | demo.almighty.io/work-item-list/detail/100:*)", "golang:*", "book:*", "and:*", "unwanted:*"},
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
		urlRegex:          urlRegex,
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
