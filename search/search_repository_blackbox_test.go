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

func (s *searchRepositoryBlackboxTest) getTestFixture(extraRecipeFuncs ...tf.RecipeFunction) *tf.TestFixture {
	baseRecipes := []tf.RecipeFunction{
		tf.WorkItemLinkTypes(1, func(fxt *tf.TestFixture, idx int) error {
			wilt := fxt.WorkItemLinkTypes[idx]
			wilt.ForwardName = link.TypeParentOf
			return nil
		}),
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
	}
	allRecipes := append(baseRecipes, extraRecipeFuncs...)
	return tf.NewTestFixture(s.T(), s.DB, allRecipes...)

}

func (s *searchRepositoryBlackboxTest) TestSearchFullText() {

	s.T().Run("Filter by title", func(t *testing.T) {

		t.Run("matching title", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			wi1 := fxt.WorkItems[0]
			wi2 := fxt.WorkItems[1]
			// when
			res, count, err := s.searchRepo.SearchFullText(context.Background(), "TestRestrictByType", nil, nil, nil)
			// then
			assert.Nil(t, err)
			assert.Equal(t, uint64(2), count)
			assert.Equal(t, res[0].Fields["system.order"], wi2.Fields["system.order"])
			assert.Equal(t, res[1].Fields["system.order"], wi1.Fields["system.order"])
		})
		s.T().Run("unmatching title", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			_, count, err := s.searchRepo.SearchFullText(context.Background(), "TRBTgorxi type:"+fxt.WorkItemTypeByName("base").ID.String(), nil, nil, nil)
			// then
			require.Nil(t, err)
			assert.Equal(t, uint64(0), count)
		})
	})

	s.T().Run("Filter by title and types", func(t *testing.T) {

		t.Run("type sub1", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			wi1 := fxt.WorkItems[0]
			// when
			res, count, err := s.searchRepo.SearchFullText(context.Background(), "TestRestrictByType type:"+fxt.WorkItemTypeByName("sub1").ID.String(), nil, nil, nil)
			// then
			require.Nil(t, err)
			require.Equal(t, uint64(1), count)
			assert.Equal(t, wi1.ID, res[0].ID)
			assert.Equal(t, res[0].Fields["system.order"], wi1.Fields["system.order"])
		})

		t.Run("type sub2", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			wi2 := fxt.WorkItems[1]
			// when
			res, count, err := s.searchRepo.SearchFullText(context.Background(), "TestRestrictByType type:"+fxt.WorkItemTypeByName("sub2").ID.String(), nil, nil, nil)
			// then
			require.Nil(t, err)
			require.Equal(t, uint64(1), count)
			assert.Equal(t, wi2.ID, res[0].ID)
			assert.Equal(t, res[0].Fields["system.order"], wi2.Fields["system.order"])
		})

		t.Run("type base", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			wi1 := fxt.WorkItems[0]
			wi2 := fxt.WorkItems[1]
			// when
			res, count, err := s.searchRepo.SearchFullText(context.Background(), "TestRestrictByType type:"+fxt.WorkItemTypeByName("base").ID.String(), nil, nil, nil)
			// then
			require.Nil(t, err)
			require.Equal(t, uint64(2), count)
			assert.Equal(t, res[0].Fields["system.order"], wi2.Fields["system.order"])
			assert.Equal(t, res[1].Fields["system.order"], wi1.Fields["system.order"])
		})

		t.Run("types sub1+sub2", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			wi1 := fxt.WorkItems[0]
			wi2 := fxt.WorkItems[1]
			// when
			res, count, err := s.searchRepo.SearchFullText(context.Background(), "TestRestrictByType type:"+fxt.WorkItemTypeByName("sub2").ID.String()+" type:"+fxt.WorkItemTypeByName("sub1").ID.String(), nil, nil, nil)
			// then
			require.Nil(t, err)
			assert.Equal(t, uint64(2), count)
			assert.Equal(t, res[0].Fields["system.order"], wi2.Fields["system.order"])
			assert.Equal(t, res[1].Fields["system.order"], wi1.Fields["system.order"])
		})

		t.Run("types base+sub1", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			wi1 := fxt.WorkItems[0]
			wi2 := fxt.WorkItems[1]
			// when
			res, count, err := s.searchRepo.SearchFullText(context.Background(), "TestRestrictByType type:"+fxt.WorkItemTypeByName("base").ID.String()+" type:"+fxt.WorkItemTypeByName("sub1").ID.String(), nil, nil, nil)
			// then
			require.Nil(t, err)
			assert.Equal(t, uint64(2), count)
			assert.Equal(t, res[0].Fields["system.order"], wi2.Fields["system.order"])
			assert.Equal(t, res[1].Fields["system.order"], wi1.Fields["system.order"])
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
			require.Nil(t, err)
			assert.Equal(t, uint64(2), count)
			assert.Equal(t, 2, len(res))
		})

		t.Run("Filter with offset", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			start := 3
			res, count, err := s.searchRepo.Filter(context.Background(), filter, nil, &start, nil)
			// then
			require.Nil(t, err)
			assert.Equal(t, uint64(2), count)
			assert.Equal(t, 0, len(res))
		})

		t.Run("Filter with limit", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			limit := 1
			res, count, err := s.searchRepo.Filter(context.Background(), filter, nil, nil, &limit)
			// then
			require.Nil(s.T(), err)
			assert.Equal(t, uint64(2), count)
			assert.Equal(t, 1, len(res))
		})
	})

	s.T().Run("Filter with parent-exists filter", func(t *testing.T) {

		t.Run("no link created", func(t *testing.T) {
			// given
			fxt := s.getTestFixture()
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			parentExists := false
			res, count, err := s.searchRepo.Filter(context.Background(), filter, &parentExists, nil, nil)
			// then both work items should be returned
			require.Nil(t, err)
			assert.Equal(t, uint64(2), count)
			assert.Equal(t, 2, len(res))
		})

		t.Run("link created", func(t *testing.T) {
			// given
			fxt := s.getTestFixture(tf.WorkItemLinks(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemLinks[0].LinkTypeID = fxt.WorkItemLinkTypes[0].ID
				fxt.WorkItemLinks[0].SourceID = fxt.WorkItems[0].ID
				fxt.WorkItemLinks[0].TargetID = fxt.WorkItems[1].ID
				return nil
			}))
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			parentExists := false
			res, count, err := s.searchRepo.Filter(context.Background(), filter, &parentExists, nil, nil)
			// then only parent work item should be returned
			require.Nil(t, err)
			assert.Equal(t, uint64(1), count)
			assert.Equal(t, 1, len(res))
			assert.Equal(t, fxt.WorkItems[0].ID, res[0].ID)
		})

		t.Run("link deleted", func(t *testing.T) {
			// given
			fxt := s.getTestFixture(tf.WorkItemLinks(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemLinks[0].LinkTypeID = fxt.WorkItemLinkTypes[0].ID
				fxt.WorkItemLinks[0].SourceID = fxt.WorkItems[0].ID
				fxt.WorkItemLinks[0].TargetID = fxt.WorkItems[1].ID
				return nil
			}))
			linkRepo := link.NewWorkItemLinkRepository(s.DB)
			err := linkRepo.Delete(context.Background(), fxt.WorkItemLinks[0].ID, fxt.Identities[0].ID)
			require.Nil(t, err)
			// when
			filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, fxt.Spaces[0].ID)
			parentExists := false
			res, count, err := s.searchRepo.Filter(context.Background(), filter, &parentExists, nil, nil)
			// then both work items should be returned
			require.Nil(t, err)
			assert.Equal(t, uint64(2), count)
			assert.Equal(t, 2, len(res))
		})

	})

}
