package search

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	var err error

	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	if _, c := os.LookupEnv(resource.Database); c {
		db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())
		if err != nil {
			panic("Failed to connect database: " + err.Error())
		}
		defer db.Close()
	}
	os.Exit(m.Run())
}

func TestSearchByText(t *testing.T) {
	resource.Require(t, resource.Database)

	wir := models.NewWorkItemRepository(db)

	models.Transactional(db, func(tx *gorm.DB) error {

		workItem := app.WorkItem{Fields: make(map[string]interface{})}
		createdWorkItems := make([]string, 0, 3)

		workItem.Fields = map[string]interface{}{
			models.SystemTitle:       "test sbose title for search",
			models.SystemDescription: "description for search test",
			models.SystemCreator:     "sbose78",
			models.SystemAssignee:    "pranav",
			models.SystemState:       "closed",
		}

		workItemUrlInSearchString := "http://demo.almighty.io/detail/"
		searchString := "Sbose deScription "
		createdWorkItem, err := wir.Create(context.Background(), models.SystemBug, workItem.Fields)
		defer wir.Delete(context.Background(), createdWorkItem.ID)

		if err != nil {
			t.Fatal("Couldnt create test data")
		}

		// create the URL and use it in the search string
		workItemUrlInSearchString = workItemUrlInSearchString + createdWorkItem.ID

		// had to dynamically create this since I didn't now the URL/ID of the workitem
		// till the test data was created.
		searchString = searchString + workItemUrlInSearchString
		t.Log("using search string: " + searchString)

		createdWorkItems = append(createdWorkItems, createdWorkItem.ID)
		t.Log(createdWorkItem.ID)

		sr := NewGormSearchRepository(db)
		var start, limit int = 0, 100
		workItemList, _, err := sr.SearchFullText(context.Background(), searchString, &start, &limit)
		if err != nil {
			t.Fatal("Error getting search result ", err)
		}

		// Since this test adds test data, whether or not other workitems exist
		// there must be at least 1 search result returned.
		assert.NotEqual(t, 0, len(workItemList))

		// These keywords need a match in the textual part.
		allKeywords := []string{workItemUrlInSearchString, createdWorkItem.ID, "Sbose", "deScription"}

		// These keywords need a match
		optionalKeywords := []string{workItemUrlInSearchString, createdWorkItem.ID}

		// We will now check the legitimacy of the search results.
		// Iterate through all search results and see whether they meet the critera

		for _, workItemValue := range workItemList {
			t.Log("Found search result  ", workItemValue.ID)

			for _, keyWord := range allKeywords {

				workItemTitle := strings.ToLower(workItemValue.Fields[models.SystemTitle].(string))
				workItemDescription := strings.ToLower(workItemValue.Fields[models.SystemDescription].(string))
				keyWord = strings.ToLower(keyWord)

				if strings.Contains(workItemTitle, keyWord) || strings.Contains(workItemDescription, keyWord) {
					// Check if the search keyword is present as text in the title/description
					t.Logf("Found keyword %s in workitem %s", keyWord, workItemValue.ID)
				} else if stringInSlice(keyWord, optionalKeywords) && strings.Contains(keyWord, workItemValue.ID) {
					// If not present in title/description then it should be a URL or ID
					t.Logf("Found keyword %s as ID %s from the URL", keyWord, workItemValue.ID)
				} else {
					t.Errorf("%s neither found in title %s nor in the description: %s", keyWord, workItemValue.Fields[models.SystemTitle], workItemValue.Fields[models.SystemDescription])
				}
			}
			// defer wir.Delete(context.Background(), workItemValue.ID)
		}

		return err
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
	resource.Require(t, resource.Database)
	wir := models.NewWorkItemRepository(db)

	models.Transactional(db, func(tx *gorm.DB) error {

		workItem := app.WorkItem{Fields: make(map[string]interface{})}

		workItem.Fields = map[string]interface{}{
			models.SystemTitle:       "Search Test Sbose",
			models.SystemDescription: "Description",
			models.SystemCreator:     "sbose78",
			models.SystemAssignee:    "pranav",
			models.SystemState:       "closed",
		}

		createdWorkItem, err := wir.Create(context.Background(), models.SystemBug, workItem.Fields)
		if err != nil {
			t.Fatal("Couldnt create test data")
		}

		// Create a new workitem to have the ID in it's title. This should not come
		// up in search results

		workItem.Fields[models.SystemTitle] = "Search test sbose " + createdWorkItem.ID
		_, err = wir.Create(context.Background(), models.SystemBug, workItem.Fields)
		if err != nil {
			t.Fatal("Couldnt create test data")
		}

		sr := NewGormSearchRepository(db)

		var start, limit int = 0, 100
		workItemList, _, err := sr.SearchFullText(context.Background(), "id:"+createdWorkItem.ID, &start, &limit)
		if err != nil {
			t.Fatal("Error gettig search result ", err)
		}

		// ID is unique, hence search result set's length should be 1
		assert.Equal(t, len(workItemList), 1)
		for _, workItemValue := range workItemList {
			t.Log("Found search result for ID Search ", workItemValue.ID)
			assert.Equal(t, createdWorkItem.ID, workItemValue.ID)

			// clean it up if found, this effectively cleans up the test data created.
			// this for loop is always of 1 iteration, hence only 1 item gets deleted anyway.

			defer wir.Delete(context.Background(), workItemValue.ID)
		}
		return err
	})
}

func TestGenerateSQLSearchString(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	input := searchKeyword{
		id:    []string{"10", "99"},
		words: []string{"username", "title_substr", "desc_substr"},
	}
	expectedSQLParameter := "10 & 99 & username & title_substr & desc_substr"
	expectedSQLQuery := WhereClauseForSearchByText

	actualSQLQuery, actualSQLParameter := generateSQLSearchInfo(input)
	assert.Equal(t, expectedSQLParameter, actualSQLParameter)
	assert.Equal(t, expectedSQLQuery, actualSQLQuery)
}

func TestParseSearchString(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	input := "user input for search string with some ids like id:99 and id:400 but this is not id like 800"
	op := parseSearchString(input)
	expectedSearchRes := searchKeyword{
		id:    []string{"99", "400"},
		words: []string{"user", "input", "for", "search", "string", "with", "some", "ids", "like", "and", "but", "this", "is", "not", "id", "like", "800"},
	}
	assert.True(t, assert.ObjectsAreEqualValues(expectedSearchRes, op))
}

func TestParseSearchStringURL(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	input := "http://demo.almighty.io/detail/100"
	op := parseSearchString(input)

	expectedSearchRes := searchKeyword{
		id:    nil,
		words: []string{"100:* | demo.almighty.io/detail/100:*"},
	}
	fmt.Printf("\n%#v\n%#v", op, expectedSearchRes)

	assert.True(t, assert.ObjectsAreEqualValues(expectedSearchRes, op))
}

func TestParseSearchStringURLWithouID(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	input := "http://demo.almighty.io/detail/"
	op := parseSearchString(input)

	expectedSearchRes := searchKeyword{
		id:    nil,
		words: []string{"demo.almighty.io/detail:*"},
	}
	fmt.Printf("\n%#v\n%#v", op, expectedSearchRes)

	assert.True(t, assert.ObjectsAreEqualValues(expectedSearchRes, op))
}

func TestParseSearchStringDifferentURL(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	input := "http://demo.redhat.io"
	op := parseSearchString(input)
	expectedSearchRes := searchKeyword{
		id:    nil,
		words: []string{"demo.redhat.io:*"},
	}
	assert.True(t, assert.ObjectsAreEqualValues(expectedSearchRes, op))
}

func TestRegisterAsKnownURL(t *testing.T) {
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
	// getSearchQueryFromURLPattern
	// register urls
	// select pattern and pass search string
	// validate output with different scenarios like ID present not present
	urlRegex := `(?P<domain>google.me.io)(?P<path>/everything/)(?P<id>\d*)`
	routeName := "custom-test-route"
	RegisterAsKnownURL(routeName, urlRegex)

	searchQuery := getSearchQueryFromURLPattern(routeName, "google.me.io/everything/100")
	assert.Equal(t, "100:* | google.me.io/everything/100:*", searchQuery)

	searchQuery = getSearchQueryFromURLPattern(routeName, "google.me.io/everything/")
	assert.Equal(t, "google.me.io/everything/:*", searchQuery)

	// cleanup
	delete(knownURLs, routeName)
}

func TestGetSearchQueryFromURLString(t *testing.T) {
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
	assert.Equal(t, "100:* | google.me.io/everything/100:*", searchQuery)
}
