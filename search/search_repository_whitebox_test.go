package search

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSearchRepositoryWhitebox(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchRepositoryWhiteboxTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type searchRepositoryWhiteboxTestSuite struct {
	gormtestsupport.DBTestSuite
	sr *GormSearchRepository
}

func (s *searchRepositoryWhiteboxTestSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.sr = NewGormSearchRepository(s.DB)
}

func (s *searchRepositoryWhiteboxTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
}

type SearchTestDescriptor struct {
	fields         map[string]interface{}
	searchString   string
	minimumResults int
}

func (s *searchRepositoryWhiteboxTestSuite) TestSearch() {

	var start, limit int = 0, 100

	s.T().Run("Search accross title and description", func(t *testing.T) {

		t.Run("match title and descrition", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(10, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "test sbose title '12345678asdfgh'"
					fxt.WorkItems[idx].Fields[workitem.SystemDescription] = rendering.NewMarkupContentFromLegacy(`"description" for search test`)
				}
				return nil
			}))
			// when
			searchKeywords := Keywords{
				Words: []string{"Sbose", "deScription", "12345678asdfgh"},
			}
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.sr.SearchFullText(context.Background(), searchKeywords, &start, &limit, &spaceID)
			// then
			require.Nil(t, err)
			verify(t, searchKeywords, searchResults, 1)
		})

		t.Run("match title with description undefined", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(10, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "test nofield sbose title '12345678asdfgh'"
				}
				return nil
			}))
			// when
			searchKeywords := Keywords{
				Words: []string{"Sbose", "nofield"},
			}
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.sr.SearchFullText(context.Background(), searchKeywords, &start, &limit, &spaceID)
			// then
			require.Nil(t, err)
			verify(t, searchKeywords, searchResults, 1)
		})

		t.Run("match title with slash", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(10, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "add new error types in models/errors.go'"
					fxt.WorkItems[idx].Fields[workitem.SystemDescription] = rendering.NewMarkupContentFromLegacy(`Make sure remoteworkitem can access..`)
				}
				return nil
			}))
			// when
			searchKeywords := Keywords{
				Words: []string{"models/errors.go", "remoteworkitem"},
			}
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.sr.SearchFullText(context.Background(), searchKeywords, &start, &limit, &spaceID)
			// then
			require.Nil(t, err)
			verify(t, searchKeywords, searchResults, 1)
		})

		t.Run("match title with braces", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(10, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "Bug reported by administrator for input = (value)"
				}
				return nil
			}))
			// when
			searchKeywords := Keywords{
				Words: []string{"(value)"},
			}
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.sr.SearchFullText(context.Background(), searchKeywords, &start, &limit, &spaceID)
			// then
			require.Nil(t, err)
			verify(t, searchKeywords, searchResults, 1)

		})

		t.Run("match title with braces and brackets", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(10, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "trial for braces (pranav) {shoubhik} [aslak]"
				}
				return nil
			}))
			// when
			searchKeywords := Keywords{
				Words: []string{"(pranav)", "{shoubhik}", "[aslak]"},
			}
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.sr.SearchFullText(context.Background(), searchKeywords, &start, &limit, &spaceID)
			// then
			require.Nil(t, err)
			verify(t, searchKeywords, searchResults, 1)

		})

		t.Run("no match", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(10, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "test should return 0 results'"
				}
				return nil
			}))
			// when
			searchKeywords := Keywords{
				Words: []string{"negative", "case"},
			}
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.sr.SearchFullText(context.Background(), searchKeywords, &start, &limit, &spaceID)
			// then
			require.Nil(t, err)
			verify(t, searchKeywords, searchResults, 0)
		})
	})

	s.T().Run("Search by number", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(10))
		spaceID := fxt.Spaces[0].ID.String()

		t.Run("and by space", func(t *testing.T) {
			t.Run("single match", func(t *testing.T) {
				// given
				queryNumber := fxt.WorkItems[2].Number
				// when looking for `number:3`
				searchKeywords := Keywords{
					Number: []string{fmt.Sprintf("%d:*A", queryNumber)},
				}
				searchResults, _, err := s.sr.SearchFullText(context.Background(), searchKeywords, &start, &limit, &spaceID)
				// then there should be a single match
				require.Nil(t, err)
				require.Len(t, searchResults, 1)
				assert.Equal(t, queryNumber, searchResults[0].Number)
			})

			t.Run("multiple matches", func(t *testing.T) {
				// given
				queryNumber := fxt.WorkItems[0].Number
				// when looking for `number:1`
				searchKeywords := Keywords{
					Number: []string{fmt.Sprintf("%d:*A", queryNumber)},
				}
				searchResults, _, err := s.sr.SearchFullText(context.Background(), searchKeywords, &start, &limit, &spaceID)
				// then there should be 2 matches: `1` and `10`
				require.Nil(t, err)
				require.Len(t, searchResults, 2)
				for _, searchResult := range searchResults {
					// verifies that the number in the search result contains the query number
					assert.Contains(t, strconv.Itoa(searchResult.Number), strconv.Itoa(queryNumber))
				}
			})
			t.Run("not found", func(t *testing.T) {
				// given
				notExistingWINumber := 12345 // We only created one work item in that space, so that number should not exist
				searchKeywords := Keywords{
					Number: []string{strconv.Itoa(notExistingWINumber)},
				}
				// when
				workItemList, _, err := s.sr.SearchFullText(context.Background(), searchKeywords, &start, &limit, &spaceID)
				// then
				require.Nil(t, err)
				require.Len(t, workItemList, 0)
			})
		})
		t.Run("not by space", func(t *testing.T) {
			t.Run("single match", func(t *testing.T) {
				// given
				searchKeywords := Keywords{
					Number: []string{strconv.Itoa(fxt.WorkItems[0].Number)},
				}
				// when
				workItemList, _, err := s.sr.SearchFullText(context.Background(), searchKeywords, &start, &limit, nil)
				// then
				require.Nil(t, err)
				require.True(t, len(workItemList) >= 1, "at least one work item should be found for the given work item number")
				var found bool
				for _, wi := range workItemList {
					if wi.ID == fxt.WorkItems[0].ID {
						found = true
					}
				}
				require.True(t, found, "failed to found: %s", fxt.WorkItems[0].ID)
			})
			t.Run("not found", func(t *testing.T) {
				// given
				notExistingWINumber := math.MaxInt64 - 1 // That ID most likely does not exist at all
				searchKeywords := Keywords{
					Number: []string{strconv.Itoa(notExistingWINumber)},
				}
				// when
				workItemList, _, err := s.sr.SearchFullText(context.Background(), searchKeywords, &start, &limit, nil)
				// then
				require.Nil(t, err)
				require.Len(t, workItemList, 0)
			})
		})
	})

	s.T().Run("Search by URL - single match", func(t *testing.T) {
		t.Run("single match", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(10, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "test nofield sbose title '12345678asdfgh'"
				}
				return nil
			}))
			// when looking for `http://.../3` there should be a single match
			queryNumber := fxt.WorkItems[2].Number
			searchKeywords := Keywords{
				Words: []string{fmt.Sprintf("%[2]d:*A | %[1]s%[2]d", "demo.openshift.io/work-item/list/detail/", queryNumber)},
			}
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.sr.SearchFullText(context.Background(), searchKeywords, &start, &limit, &spaceID)
			// then
			require.Nil(t, err)
			require.Len(t, searchResults, 1)
			assert.Equal(t, queryNumber, searchResults[0].Number)
		})

		t.Run("multiple matches", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(10, func(fxt *tf.TestFixture, idx int) error {
				if idx == 0 {
					fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "test nofield sbose title '12345678asdfgh'"
				}
				return nil
			}))
			// when looking for `http://.../1` there should be a 2 matchs: `http://.../1` and `http://.../10``
			queryNumber := fxt.WorkItems[0].Number
			searchKeywords := Keywords{
				Words: []string{fmt.Sprintf("%[2]d:*A | %[1]s%[2]d", "demo.openshift.io/work-item/list/detail/", queryNumber)},
			}
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.sr.SearchFullText(context.Background(), searchKeywords, &start, &limit, &spaceID)
			// then
			require.Nil(t, err)
			require.Len(t, searchResults, 2)
			for _, searchResult := range searchResults {
				// verifies that the number in the search result contains the query number
				assert.Contains(t, strconv.Itoa(searchResult.Number), strconv.Itoa(queryNumber))
			}
		})
	})
}

// verify verifies that the search results match with the expected count and that the title or description contain all
// the terms of the search query
func verify(t *testing.T, searchKeywords Keywords, searchResults []workitem.WorkItem, expectedCount int) {
	// Since this test adds test data, whether or not other workitems exist
	// there must be at least 1 search result returned.
	if len(searchResults) == expectedCount && expectedCount == 0 {
		// no point checking further, we got what we wanted.
		return
	}
	require.Equal(t, expectedCount, len(searchResults), "invalid number of results in the search")

	// These keywords need a match in the textual part.
	allKeywords := searchKeywords.Words
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
	input := Keywords{
		Number: []string{"10", "99"},
		Words:  []string{"username", "title_substr", "desc_substr"},
	}
	expectedSQLParameter := "10 & 99 & username & title_substr & desc_substr"

	actualSQLParameter := generateSQLSearchInfo(input)
	assert.Equal(t, expectedSQLParameter, actualSQLParameter)
}

func TestGenerateSQLSearchStringIdOnly(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	input := Keywords{
		Number: []string{"10"},
		Words:  []string{},
	}
	expectedSQLParameter := "10"

	actualSQLParameter := generateSQLSearchInfo(input)
	assert.Equal(t, expectedSQLParameter, actualSQLParameter)
}
