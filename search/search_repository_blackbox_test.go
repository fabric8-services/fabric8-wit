package search_test

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/search"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRunSearchRepositoryBlackboxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchRepositoryBlackboxTest{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

type searchRepositoryBlackboxTest struct {
	gormtestsupport.DBTestSuite
	searchRepo *search.GormSearchRepository
}

func (s *searchRepositoryBlackboxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.searchRepo = search.NewGormSearchRepository(s.DB)
}

func (s *searchRepositoryBlackboxTest) getTestFixture() *tf.TestFixture {
	return tf.NewTestFixture(s.T(), s.DB,
		tf.WorkItemTypes(3, func(fxt *tf.TestFixture, idx int) error {
			wit := fxt.WorkItemTypes[idx]
			wit.ID = uuid.NewV4()
			switch idx {
			case 0:
				wit.Name = "base"
			case 1:
				wit.Name = "sub1"
				wit.Extends = fxt.WorkItemTypeByName("base").ID
			case 2:
				wit.Name = "sub2"
				wit.Extends = fxt.WorkItemTypeByName("base").ID
			}
			return nil
		}),
		tf.WorkItems(2, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			switch idx {
			case 0:
				wi.Type = fxt.WorkItemTypes[1].ID
				wi.Fields[workitem.SystemTitle] = "Test TestRestrictByType"
			case 1:
				wi.Type = fxt.WorkItemTypes[2].ID
				wi.Fields[workitem.SystemTitle] = "Test TestRestrictByType 2"
			}
			return nil
		}),
	)
}

func (s *searchRepositoryBlackboxTest) TestSearchWithJoin() {
	s.T().Run("join iterations", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB,
			tf.Iterations(2),
			tf.WorkItems(10, func(fxt *tf.TestFixture, idx int) error {
				switch idx {
				case 0, 1, 2, 3, 4, 5, 6:
					fxt.WorkItems[idx].Fields[workitem.SystemIteration] = fxt.Iterations[0].ID.String()
				default:
					fxt.WorkItems[idx].Fields[workitem.SystemIteration] = fxt.Iterations[1].ID.String()
				}
				return nil
			}),
		)
		t.Run("matching name", func(t *testing.T) {
			// when
			filter := fmt.Sprintf(`{"iteration.name": "%s"}`, fxt.Iterations[0].Name)
			res, count, _, _, err := s.searchRepo.Filter(context.Background(), filter, nil, nil, nil)
			// then
			require.NoError(t, err)
			assert.Equal(t, 7, count)
			toBeFound := id.Slice{
				fxt.WorkItems[0].ID,
				fxt.WorkItems[1].ID,
				fxt.WorkItems[2].ID,
				fxt.WorkItems[3].ID,
				fxt.WorkItems[4].ID,
				fxt.WorkItems[5].ID,
				fxt.WorkItems[6].ID,
			}.ToMap()
			for _, wi := range res {
				_, ok := toBeFound[wi.ID]
				require.True(t, ok, "unknown work item found: %s", wi.ID)
				delete(toBeFound, wi.ID)
			}
			require.Empty(t, toBeFound, "failed to found all work items: %+s", toBeFound)
		})
	})
	s.T().Run("join work item type groups", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItemTypes(2),
			tf.WorkItemTypeGroups(2, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemTypeGroups[idx].TypeList = []uuid.UUID{fxt.WorkItemTypes[idx].ID}
				return nil
			}),
			tf.WorkItems(4, func(fxt *tf.TestFixture, idx int) error {
				switch idx {
				case 0, 1, 2:
					fxt.WorkItems[idx].Type = fxt.WorkItemTypes[0].ID
				default:
					fxt.WorkItems[idx].Type = fxt.WorkItemTypes[1].ID
				}
				return nil
			}),
		)
		t.Run("matching name", func(t *testing.T) {
			// when
			filter := fmt.Sprintf(`{"typegroup.name": "%s"}`, fxt.WorkItemTypeGroups[0].Name)
			res, count, _, _, err := s.searchRepo.Filter(context.Background(), filter, nil, nil, nil)
			// then
			require.NoError(t, err)
			toBeFound := id.MapFromSlice(id.Slice{
				fxt.WorkItems[0].ID,
				fxt.WorkItems[1].ID,
				fxt.WorkItems[2].ID,
			})
			assert.Equal(t, len(toBeFound), count)
			for _, wi := range res {
				_, ok := toBeFound[wi.ID]
				require.True(t, ok, "unknown work item found: %s", wi.ID)
				delete(toBeFound, wi.ID)
			}
			require.Empty(t, toBeFound, "failed to found all work items: %+s", toBeFound)
		})
	})
}

func (s *searchRepositoryBlackboxTest) TestSearchFullText() {

	s.T().Run("Filter by title", func(t *testing.T) {

		t.Run("matching title", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			spaceID := fxt.Spaces[0].ID.String()
			query := "TestRestrictByType"
			res, count, err := s.searchRepo.SearchFullText(context.Background(), query, nil, nil, &spaceID)
			// then
			require.NoError(t, err)
			assert.Equal(t, 2, count)
			assert.Condition(t, containsAllWorkItems(res, *fxt.WorkItems[1], *fxt.WorkItems[0]))
			assert.NotNil(t, res[0].Fields[workitem.SystemNumber])
			assert.NotNil(t, res[1].Fields[workitem.SystemNumber])
		})
		s.T().Run("unmatching title", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			spaceID := fxt.Spaces[0].ID.String()
			query := "TRBTgorxi type:" + fxt.WorkItemTypeByName("base").ID.String()
			_, count, err := s.searchRepo.SearchFullText(context.Background(), query, nil, nil, &spaceID)
			// then
			require.NoError(t, err)
			assert.Equal(t, 0, count)
		})
	})

	s.T().Run("SearchFullText by title and types", func(t *testing.T) {

		t.Run("type sub1", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			spaceID := fxt.Spaces[0].ID.String()
			query := "TestRestrictByType type:" + fxt.WorkItemTypeByName("sub1").ID.String()
			res, count, err := s.searchRepo.SearchFullText(context.Background(), query, nil, nil, &spaceID)
			// then
			require.NoError(t, err)
			require.Equal(t, 1, count)
			assert.Condition(t, containsAllWorkItems(res, *fxt.WorkItems[0]))
		})

		t.Run("type sub2", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			spaceID := fxt.Spaces[0].ID.String()
			query := "TestRestrictByType type:" + fxt.WorkItemTypeByName("sub2").ID.String()
			res, count, err := s.searchRepo.SearchFullText(context.Background(), query, nil, nil, &spaceID)
			// then
			require.NoError(t, err)
			require.Equal(t, 1, count)
			assert.Condition(t, containsAllWorkItems(res, *fxt.WorkItems[1]))
		})

		t.Run("type base", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			spaceID := fxt.Spaces[0].ID.String()
			query := "TestRestrictByType type:" + fxt.WorkItemTypeByName("base").ID.String()
			res, count, err := s.searchRepo.SearchFullText(context.Background(), query, nil, nil, &spaceID)
			// then
			require.NoError(t, err)
			require.Equal(t, 2, count)
			assert.Condition(t, containsAllWorkItems(res, *fxt.WorkItems[1], *fxt.WorkItems[0]))
		})

		t.Run("types sub1+sub2", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			spaceID := fxt.Spaces[0].ID.String()
			query := "TestRestrictByType type:" + fxt.WorkItemTypeByName("sub2").ID.String() + " type:" + fxt.WorkItemTypeByName("sub1").ID.String()
			res, count, err := s.searchRepo.SearchFullText(context.Background(), query, nil, nil, &spaceID)
			// then
			require.NoError(t, err)
			assert.Equal(t, 2, count)
			assert.Condition(t, containsAllWorkItems(res, *fxt.WorkItems[1], *fxt.WorkItems[0]))
		})

		t.Run("types base+sub1", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			spaceID := fxt.Spaces[0].ID.String()
			query := "TestRestrictByType type:" + fxt.WorkItemTypeByName("base").ID.String() + " type:" + fxt.WorkItemTypeByName("sub1").ID.String()
			res, count, err := s.searchRepo.SearchFullText(context.Background(), query, nil, nil, &spaceID)
			// then
			require.NoError(t, err)
			assert.Equal(t, 2, count)
			assert.Condition(t, containsAllWorkItems(res, *fxt.WorkItems[1], *fxt.WorkItems[0]))
		})
	})

	s.T().Run("Filter with limits", func(t *testing.T) {

		t.Run("none", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			res, count, ancestors, childLinks, err := s.searchRepo.Filter(context.Background(), filter, nil, nil, nil)
			// when
			require.NoError(t, err)
			assert.Equal(t, 2, count)
			assert.Equal(t, 2, len(res))
			assert.Empty(t, ancestors)
			assert.Empty(t, childLinks)
		})

		t.Run("with offset", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			start := 3
			res, count, ancestors, childLinks, err := s.searchRepo.Filter(context.Background(), filter, nil, &start, nil)
			// then
			require.NoError(t, err)
			assert.Equal(t, 2, count)
			assert.Equal(t, 0, len(res))
			assert.Empty(t, ancestors)
			assert.Empty(t, childLinks)
		})

		t.Run("with limit", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			limit := 1
			res, count, ancestors, childLinks, err := s.searchRepo.Filter(context.Background(), filter, nil, nil, &limit)
			// then
			require.NoError(s.T(), err)
			assert.Equal(t, 2, count)
			assert.Equal(t, 1, len(res))
			assert.Empty(t, ancestors)
			assert.Empty(t, childLinks)
		})
	})

	s.T().Run("with parent-exists filter", func(t *testing.T) {

		t.Run("no link created", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(3))
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			parentExists := false
			res, count, ancestors, childLinks, err := s.searchRepo.Filter(context.Background(), filter, &parentExists, nil, nil)
			// then both work items should be returned
			require.NoError(t, err)
			assert.Equal(t, 3, count)
			assert.Equal(t, 3, len(res))
			assert.Empty(t, ancestors)
			assert.Empty(t, childLinks)
		})

		t.Run("link created", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB,
				tf.WorkItems(3),
				tf.WorkItemLinks(1, func(fxt *tf.TestFixture, idx int) error {
					fxt.WorkItemLinks[idx].LinkTypeID = link.SystemWorkItemLinkTypeParentChildID
					return nil
				}),
			)
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			parentExists := false
			res, count, ancestors, childLinks, err := s.searchRepo.Filter(context.Background(), filter, &parentExists, nil, nil)
			// then only parent work item should be returned
			require.NoError(t, err)
			assert.Equal(t, 2, count)
			require.Equal(t, 2, len(res))
			// item #0 is parent of #1 and item #2 is not linked to any otjer item
			assert.Condition(t, containsAllWorkItems(res, *fxt.WorkItems[2], *fxt.WorkItems[0]))
			assert.Empty(t, ancestors)
			assert.Empty(t, childLinks)
		})

		t.Run("link deleted", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB,
				tf.WorkItems(3),
				tf.WorkItemLinks(1, func(fxt *tf.TestFixture, idx int) error {
					fxt.WorkItemLinks[idx].LinkTypeID = link.SystemWorkItemLinkTypeParentChildID
					return nil
				}),
			)
			linkRepo := link.NewWorkItemLinkRepository(s.DB)
			err := linkRepo.Delete(context.Background(), fxt.WorkItemLinks[0].ID, fxt.Identities[0].ID)
			require.NoError(t, err)
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			parentExists := false
			res, count, ancestors, childLinks, err := s.searchRepo.Filter(context.Background(), filter, &parentExists, nil, nil)
			// then both work items should be returned
			require.NoError(t, err)
			assert.Equal(t, 3, count)
			assert.Equal(t, 3, len(res))
			assert.Empty(t, ancestors)
			assert.Empty(t, childLinks)
		})

	})
}

// containsAllWorkItems verifies that the `expectedWorkItems` array contains all `actualWorkitems` in the _given order_,
// by comparing the lengths and each ID,
func containsAllWorkItems(expectedWorkitems []workitem.WorkItem, actualWorkitems ...workitem.WorkItem) assert.Comparison {
	return func() bool {
		if len(expectedWorkitems) != len(actualWorkitems) {
			return false
		}
		for i, expectedWorkitem := range expectedWorkitems {
			if !uuid.Equal(expectedWorkitem.ID, actualWorkitems[i].ID) {
				return false
			}
		}
		return true
	}
}

func (s *searchRepositoryBlackboxTest) TestSearch() {

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
			searchQuery := `Sbose "deScription" '12345678asdfgh'`
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.searchRepo.SearchFullText(context.Background(), searchQuery, &start, &limit, &spaceID)
			// then
			require.NoError(t, err)
			verify(t, searchQuery, searchResults, 1)
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
			searchQuery := `sbose nofield`
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.searchRepo.SearchFullText(context.Background(), searchQuery, &start, &limit, &spaceID)
			// then
			require.NoError(t, err)
			verify(t, searchQuery, searchResults, 1)
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
			searchQuery := `models/errors.go remoteworkitem`
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.searchRepo.SearchFullText(context.Background(), searchQuery, &start, &limit, &spaceID)
			// then
			require.NoError(t, err)
			verify(t, searchQuery, searchResults, 1)
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
			searchQuery := `(value)`
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.searchRepo.SearchFullText(context.Background(), searchQuery, &start, &limit, &spaceID)
			// then
			require.NoError(t, err)
			verify(t, searchQuery, searchResults, 1)

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
			searchQuery := `(pranav) {shoubhik} [aslak]`
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.searchRepo.SearchFullText(context.Background(), searchQuery, &start, &limit, &spaceID)
			// then
			require.NoError(t, err)
			verify(t, searchQuery, searchResults, 1)

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
			searchQuery := `negative case`
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.searchRepo.SearchFullText(context.Background(), searchQuery, &start, &limit, &spaceID)
			// then
			require.NoError(t, err)
			verify(t, searchQuery, searchResults, 0)
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
				searchQuery := fmt.Sprintf("number:%d", queryNumber)
				searchResults, _, err := s.searchRepo.SearchFullText(context.Background(), searchQuery, &start, &limit, &spaceID)
				// then there should be a single match
				require.NoError(t, err)
				require.Len(t, searchResults, 1)
				assert.Equal(t, queryNumber, searchResults[0].Number)
			})

			t.Run("multiple matches", func(t *testing.T) {
				// given
				queryNumber := fxt.WorkItems[0].Number
				// when looking for `number:1`
				searchQuery := fmt.Sprintf("number:%d", queryNumber)
				searchResults, _, err := s.searchRepo.SearchFullText(context.Background(), searchQuery, &start, &limit, &spaceID)
				// then there should be 2 matches: `1` and `10`
				require.NoError(t, err)
				require.Len(t, searchResults, 2)
				for _, searchResult := range searchResults {
					// verifies that the number in the search result contains the query number
					assert.Contains(t, strconv.Itoa(searchResult.Number), strconv.Itoa(queryNumber))
				}
			})
			t.Run("not found", func(t *testing.T) {
				// given
				notExistingWINumber := 12345 // We only created one work item in that space, so that number should not exist
				searchString := "number:" + strconv.Itoa(notExistingWINumber)
				// when
				workItemList, _, err := s.searchRepo.SearchFullText(context.Background(), searchString, &start, &limit, &spaceID)
				// then
				require.NoError(t, err)
				require.Len(t, workItemList, 0)
			})
		})
		t.Run("not by space", func(t *testing.T) {
			t.Run("single match", func(t *testing.T) {
				// given
				searchString := "number:" + strconv.Itoa(fxt.WorkItems[0].Number)
				// when
				workItemList, _, err := s.searchRepo.SearchFullText(context.Background(), searchString, &start, &limit, nil)
				// then
				require.NoError(t, err)
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
				searchString := "number:" + strconv.Itoa(notExistingWINumber)
				// when
				workItemList, _, err := s.searchRepo.SearchFullText(context.Background(), searchString, &start, &limit, nil)
				// then
				require.NoError(t, err)
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
			searchQuery := fmt.Sprintf("%s%d", "http://demo.openshift.io/work-item/list/detail/", queryNumber)
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.searchRepo.SearchFullText(context.Background(), searchQuery, &start, &limit, &spaceID)
			// then
			require.NoError(t, err)
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
			searchQuery := fmt.Sprintf("%s%d", "http://demo.openshift.io/work-item/list/detail/", queryNumber)
			spaceID := fxt.Spaces[0].ID.String()
			searchResults, _, err := s.searchRepo.SearchFullText(context.Background(), searchQuery, &start, &limit, &spaceID)
			// then
			require.NoError(t, err)
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
