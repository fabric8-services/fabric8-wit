package link_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type linkRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	workitemLinkRepo *link.GormWorkItemLinkRepository
}

func TestRunLinkRepoBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &linkRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml")})
}

func (s *linkRepoBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.workitemLinkRepo = link.NewWorkItemLinkRepository(s.DB)
}

func (s *linkRepoBlackBoxTest) TestList() {
	// tests total number of workitem children returned by list is equal to the
	// total number of workitem children created and total number of workitem
	// children in a page are equal to the "limit" specified
	s.T().Run("ok - count child work items", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(4), // parent + child 1-3
			tf.WorkItemLinkTypes(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemLinkTypes[idx].ForwardName = "parent of"
				return nil
			}),
			tf.WorkItemLinks(3, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemLinks[idx].SourceID = fxt.WorkItems[0].ID
				fxt.WorkItemLinks[idx].TargetID = fxt.WorkItems[idx+1].ID
				return nil
			}),
		)

		offset := 0
		limit := 1
		res, count, err := s.workitemLinkRepo.ListWorkItemChildren(s.Ctx, fxt.WorkItems[0].ID, &offset, &limit)
		require.Nil(t, err)
		require.Len(t, res, 1)
		require.Equal(t, 3, int(count))
	})
}

func (s *linkRepoBlackBoxTest) TestWorkItemHasChildren() {
	s.T().Run("work item has no child after deletion", func(t *testing.T) {
		// given a work item link
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(2), // parent + child 1
			tf.WorkItemLinkTypes(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemLinkTypes[idx].ForwardName = "parent of"
				return nil
			}),
			tf.WorkItemLinks(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemLinks[idx].SourceID = fxt.WorkItems[0].ID
				fxt.WorkItemLinks[idx].TargetID = fxt.WorkItems[idx+1].ID
				return nil
			}),
		)

		// when this work item link is deleted
		err := s.workitemLinkRepo.Delete(s.Ctx, fxt.WorkItemLinks[0].ID, fxt.Identities[0].ID)
		require.Nil(t, err)

		// then it must not have any child
		hasChildren, err := s.workitemLinkRepo.WorkItemHasChildren(s.Ctx, fxt.WorkItems[0].ID)
		// then
		require.Nil(t, err)
		require.False(t, hasChildren)
	})
}

func (s *linkRepoBlackBoxTest) TestValidateTopology() {
	// given 2 work items linked with one tree-topology link type
	fxt := tf.NewTestFixture(s.T(), s.DB,
		tf.WorkItems(3, tf.SetWorkItemTitles([]string{"parent", "child", "another-item"})),
		tf.WorkItemLinkTypes(2,
			tf.SetTopologies(link.TopologyTree, link.TopologyTree),
			tf.SetWorkItemLinkTypeNames([]string{"tree-type", "another-type"}),
		),
		tf.WorkItemLinks(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItemLinks[idx].SourceID = fxt.WorkItemByTitle("parent").ID
			fxt.WorkItemLinks[idx].TargetID = fxt.WorkItemByTitle("child").ID
			fxt.WorkItemLinks[idx].LinkTypeID = fxt.WorkItemLinkTypeByName("tree-type").ID
			return nil
		}),
	)

	s.T().Run("ok - no link", func(t *testing.T) {
		// given link type exists but no link to child item
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(1, tf.SetWorkItemTitles([]string{"someWorkItem"})),
			tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyTree), tf.SetWorkItemLinkTypeNames([]string{"tree-type"})),
		)
		// when
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, nil, fxt.WorkItemByTitle("someWorkItem").ID, *fxt.WorkItemLinkTypeByName("tree-type"))
		// then: there must be no error because no link exists
		require.Nil(t, err)
	})

	s.T().Run("ok - link exists but ignored", func(t *testing.T) {
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, &fxt.WorkItemByTitle("parent").ID, fxt.WorkItemByTitle("child").ID, *fxt.WorkItemLinkTypeByName("tree-type"))
		// then: there must be no error because the existing link was ignored
		require.Nil(t, err)
	})

	s.T().Run("ok - no link with same type", func(t *testing.T) {
		// when using another link type to validate
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, nil, fxt.WorkItemByTitle("child").ID, *fxt.WorkItemLinkTypeByName("another-type"))
		// then: there must be no error because no link of the same type exists
		require.Nil(t, err)
	})

	s.T().Run("fail - link exists", func(t *testing.T) {
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, nil, fxt.WorkItemByTitle("child").ID, *fxt.WorkItemLinkTypeByName("tree-type"))
		// then: there must be an error because a link of the same type already exists
		require.NotNil(t, err)
	})

	s.T().Run("fail - another link exists", func(t *testing.T) {
		err := s.workitemLinkRepo.ValidateTopology(s.Ctx, &fxt.WorkItemByTitle("another-item").ID, fxt.WorkItemByTitle("child").ID, *fxt.WorkItemLinkTypeByName("tree-type"))
		// then: there must be an error because a link of the same type already exists with another parent
		require.NotNil(t, err)
	})
}

func (s *linkRepoBlackBoxTest) TestCreate() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(2, tf.SetWorkItemTitles([]string{"parent", "child"})),
			tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyTree), tf.SetWorkItemLinkTypeNames([]string{"tree-type"})),
		)
		// when
		_, err := s.workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("parent").ID, fxt.WorkItemByTitle("child").ID, fxt.WorkItemLinkTypeByName("tree-type").ID, fxt.Identities[0].ID)
		// then
		require.Nil(t, err)
	})

	s.T().Run("fail - other parent-child-link exists", func(t *testing.T) {
		// given 2 work items linked with one tree-topology link type
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItems(3, tf.SetWorkItemTitles([]string{"parent", "child", "another-item"})),
			tf.WorkItemLinkTypes(1,
				tf.SetTopologies(link.TopologyTree),
				tf.SetWorkItemLinkTypeNames([]string{"tree-type"}),
			),
			tf.WorkItemLinks(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemLinks[idx].SourceID = fxt.WorkItemByTitle("parent").ID
				fxt.WorkItemLinks[idx].TargetID = fxt.WorkItemByTitle("child").ID
				return nil
			}),
		)
		// when try to link parent#2 to child
		_, err := s.workitemLinkRepo.Create(s.Ctx, fxt.WorkItemByTitle("another-item").ID, fxt.WorkItemByTitle("child").ID, fxt.WorkItemLinkTypeByName("tree-type").ID, fxt.Identities[0].ID)
		// then expect an error because a parent/link relation already exists with the child item
		require.NotNil(t, err)
	})
}

func (s *linkRepoBlackBoxTest) TestSave() {
	s.T().Run("ok", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinks(1))
		_, err := s.workitemLinkRepo.Save(s.Ctx, *fxt.WorkItemLinks[0], fxt.Identities[0].ID)
		require.Nil(t, err)
	})
}

func (s *linkRepoBlackBoxTest) TestExistsLink() {
	s.T().Run("link exists", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinks(1))
		err := s.workitemLinkRepo.CheckExists(s.Ctx, fxt.WorkItemLinks[0].ID.String())
		require.Nil(t, err)
	})

	s.T().Run("link doesn't exist", func(t *testing.T) {
		err := s.workitemLinkRepo.CheckExists(s.Ctx, uuid.NewV4().String())
		require.IsType(t, errors.NotFoundError{}, err)
	})
}

func (s *linkRepoBlackBoxTest) TestGetParentID() {
	// create 1 links between 2 work items having TopologyNetwork with ForwardName = "parent of"
	fixtures := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemLinks(1), tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyNetwork), func(fxt *tf.TestFixture, idx int) error {
		fxt.WorkItemLinkTypes[idx].ForwardName = "parent of"
		return nil
	}))
	parentID, err := s.workitemLinkRepo.GetParentID(s.Ctx, fixtures.WorkItems[1].ID)
	require.Nil(s.T(), err)
	assert.Equal(s.T(), fixtures.WorkItems[0].ID, *parentID)
}

func (s *linkRepoBlackBoxTest) TestGetParentIDNotExist() {
	// create 1 links between 2 work items having TopologyNetwork with ForwardName = "parent of"
	fixtures := tf.NewTestFixture(s.T(), s.DB, tf.WorkItemLinks(1), tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyNetwork), func(fxt *tf.TestFixture, idx int) error {
		fxt.WorkItemLinkTypes[idx].ForwardName = "parent of"
		return nil
	}))
	parentID, err := s.workitemLinkRepo.GetParentID(s.Ctx, fixtures.WorkItems[0].ID)
	require.NotNil(s.T(), err)
	assert.Nil(s.T(), parentID)
}
