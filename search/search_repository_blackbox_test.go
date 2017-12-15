package search_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
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
	suite.Run(t, &searchRepositoryBlackboxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
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
		tf.WorkItemTypes(4, func(fxt *tf.TestFixture, idx int) error {
			wit := fxt.WorkItemTypes[idx]
			wit.ID = uuid.NewV4()
			switch idx {
			case 0:
				wit.Name = "base"
				wit.Path = workitem.LtreeSafeID(wit.ID)
			case 1:
				wit.Name = "sub1"
				wit.Path = fxt.WorkItemTypes[0].Path + workitem.GetTypePathSeparator() + workitem.LtreeSafeID(wit.ID)
			case 2:
				wit.Name = "sub2"
				wit.Path = fxt.WorkItemTypes[0].Path + workitem.GetTypePathSeparator() + workitem.LtreeSafeID(wit.ID)
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
			assert.Equal(t, uint64(2), count)
			assert.Condition(t, containsAllWorkItems(res, *fxt.WorkItems[1], *fxt.WorkItems[0]))
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
			assert.Equal(t, uint64(0), count)
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
			require.Equal(t, uint64(1), count)
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
			require.Equal(t, uint64(1), count)
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
			require.Equal(t, uint64(2), count)
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
			assert.Equal(t, uint64(2), count)
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
			assert.Equal(t, uint64(2), count)
			assert.Condition(t, containsAllWorkItems(res, *fxt.WorkItems[1], *fxt.WorkItems[0]))
		})
	})

	s.T().Run("Filter with limits", func(t *testing.T) {

		t.Run("none", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			res, count, err := s.searchRepo.Filter(context.Background(), filter, nil, nil, nil)
			// when
			require.NoError(t, err)
			assert.Equal(t, uint64(2), count)
			assert.Equal(t, 2, len(res))
		})

		t.Run("with offset", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			start := 3
			res, count, err := s.searchRepo.Filter(context.Background(), filter, nil, &start, nil)
			// then
			require.NoError(t, err)
			assert.Equal(t, uint64(2), count)
			assert.Equal(t, 0, len(res))
		})

		t.Run("with limit", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			limit := 1
			res, count, err := s.searchRepo.Filter(context.Background(), filter, nil, nil, &limit)
			// then
			require.NoError(s.T(), err)
			assert.Equal(t, uint64(2), count)
			assert.Equal(t, 1, len(res))
		})
	})

	s.T().Run("with parent-exists filter", func(t *testing.T) {

		t.Run("no link created", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(3))
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			parentExists := false
			res, count, err := s.searchRepo.Filter(context.Background(), filter, &parentExists, nil, nil)
			// then both work items should be returned
			require.NoError(t, err)
			assert.Equal(t, uint64(3), count)
			assert.Equal(t, 3, len(res))
		})

		t.Run("link created", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB,
				tf.WorkItemLinkTypes(1, func(fxt *tf.TestFixture, idx int) error {
					// need an explicit 'parent-of' type of link
					fxt.WorkItemLinkTypes[idx].ForwardName = link.TypeParentOf
					fxt.WorkItemLinkTypes[idx].Topology = link.TopologyTree
					return nil
				}),
				tf.WorkItems(3),
				tf.WorkItemLinks(1))
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			parentExists := false
			res, count, err := s.searchRepo.Filter(context.Background(), filter, &parentExists, nil, nil)
			// then only parent work item should be returned
			require.NoError(t, err)
			assert.Equal(t, uint64(2), count)
			require.Equal(t, 2, len(res))
			// item #0 is parent of #1 and item #2 is not linked to any otjer item
			assert.Condition(t, containsAllWorkItems(res, *fxt.WorkItems[2], *fxt.WorkItems[0]))
		})

		t.Run("link deleted", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB,
				tf.WorkItemLinkTypes(1, func(fxt *tf.TestFixture, idx int) error {
					// need an explicit 'parent-of' type of link
					fxt.WorkItemLinkTypes[idx].ForwardName = link.TypeParentOf
					fxt.WorkItemLinkTypes[idx].Topology = link.TopologyTree
					return nil
				}),
				tf.WorkItems(3),
				tf.WorkItemLinks(1))
			linkRepo := link.NewWorkItemLinkRepository(s.DB)
			err := linkRepo.Delete(context.Background(), fxt.WorkItemLinks[0].ID, fxt.Identities[0].ID)
			require.NoError(t, err)
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			parentExists := false
			res, count, err := s.searchRepo.Filter(context.Background(), filter, &parentExists, nil, nil)
			// then both work items should be returned
			require.NoError(t, err)
			assert.Equal(t, uint64(3), count)
			assert.Equal(t, 3, len(res))
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
