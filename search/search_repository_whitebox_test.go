package search

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/transaction"
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
	ts := models.NewGormTransactionSupport(db)
	witr := models.NewWorkItemTypeRepository(ts)
	wir := models.NewWorkItemRepository(ts, witr)

	transaction.Do(ts, func() error {

		workItem := app.WorkItem{Fields: make(map[string]interface{})}
		createdWorkItems := make([]string, 0, 3)

		workItem.Fields = map[string]interface{}{
			models.SystemTitle:       "test sbose title for search",
			models.SystemDescription: "description for search test",
			models.SystemCreator:     "sbose78",
			models.SystemAssignee:    "pranav",
			models.SystemState:       "closed",
		}

		searchString := "Sbose deScription"
		createdWorkItem, err := wir.Create(context.Background(), models.SystemBug, workItem.Fields)
		defer wir.Delete(context.Background(), createdWorkItem.ID)

		if err != nil {
			t.Fatal("Couldnt create test data")
		}
		createdWorkItems = append(createdWorkItems, createdWorkItem.ID)
		t.Log(createdWorkItem.ID)

		sr := NewGormSearchRepository(ts, witr)
		workItemList, err := sr.SearchFullText(context.Background(), searchString)
		if err != nil {
			t.Fatal("Error getting search result ", err)
		}

		mandatoryKeyWords := strings.Split(searchString, " ")
		for _, workItemValue := range workItemList {
			t.Log("Found search result  ", workItemValue.ID)

			for _, keyWord := range mandatoryKeyWords {

				workItemTitle := strings.ToLower(workItemValue.Fields[models.SystemTitle].(string))
				workItemDescription := strings.ToLower(workItemValue.Fields[models.SystemDescription].(string))
				keyWord = strings.ToLower(keyWord)

				if strings.Contains(workItemTitle, keyWord) || strings.Contains(workItemDescription, keyWord) {
					t.Logf("Found keyword %s in workitem %s", keyWord, workItemValue.ID)
				} else {
					t.Errorf("%s neither found in title %s nor in the description: %s", keyWord, workItemValue.Fields[models.SystemTitle], workItemValue.Fields[models.SystemDescription])
				}
			}
			// defer wir.Delete(context.Background(), workItemValue.ID)
		}

		return err
	})
}

func TestSearchByID(t *testing.T) {
	resource.Require(t, resource.Database)
	ts := models.NewGormTransactionSupport(db)
	witr := models.NewWorkItemTypeRepository(ts)
	wir := models.NewWorkItemRepository(ts, witr)

	transaction.Do(ts, func() error {

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

		sr := NewGormSearchRepository(ts, witr)

		workItemList, err := sr.SearchFullText(context.Background(), "id:"+createdWorkItem.ID)
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
		words: []string{"username title_substr desc_substr"},
	}
	expectedSqlParameter := strings.Join(input.id, " & ") + strings.Join(input.words, " & ")
	expectedSqlQuery := testText

	actualSqlQuery, actualSqlParameter := generateSQLSearchInfo(input)
	assert.Equal(t, expectedSqlParameter, actualSqlParameter)
	assert.Equal(t, expectedSqlQuery, actualSqlQuery)
}

func TestParseSearchString(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	input := "user input for search string with some ids like id:99 and id:400 but this is not id like 800"
	op := parseSearchString(input)
	expectedSearchRes := searchKeyword{
		id:    []string{"99", "400"},
		words: []string{"user", "input", "for", "search", "string", "with", "some", "ids", "like", "and", "but", "this", "is", "not", "id", "like", "800"},
	}
	assert.ObjectsAreEqualValues(expectedSearchRes, op)
}
