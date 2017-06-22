package link_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type linkRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	workitemLinkRepo         *link.GormWorkItemLinkRepository
	workitemLinkTypeRepo     link.WorkItemLinkTypeRepository
	workitemLinkCategoryRepo link.WorkItemLinkCategoryRepository
	workitemRepo             workitem.WorkItemRepository
	clean                    func()
	ctx                      context.Context
	testSpace                uuid.UUID
	testIdentity             account.Identity
	linkCategoryID           uuid.UUID
	testTreeLinkTypeID       uuid.UUID
	parent1                  *workitem.WorkItem
	parent1ID                uint64
	parent2                  *workitem.WorkItem
	parent2ID                uint64
	child                    *workitem.WorkItem
	childID                  uint64
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *linkRepoBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)
}

func TestRunLinkRepoBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &linkRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml")})
}

func (s *linkRepoBlackBoxTest) SetupTest() {
	s.workitemRepo = workitem.NewWorkItemRepository(s.DB)
	s.workitemLinkRepo = link.NewWorkItemLinkRepository(s.DB)
	s.workitemLinkTypeRepo = link.NewWorkItemLinkTypeRepository(s.DB)
	s.workitemLinkCategoryRepo = link.NewWorkItemLinkCategoryRepository(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "jdoe1", "test")
	s.testIdentity = testIdentity
	require.Nil(s.T(), err)

	// create a space
	spaceRepository := space.NewRepository(s.DB)
	spaceName := testsupport.CreateRandomValidTestName("test-space")
	testSpace, err := spaceRepository.Create(s.ctx, &space.Space{
		Name: spaceName,
	})
	s.testSpace = testSpace.ID
	require.Nil(s.T(), err)

	// Create a work item link category
	categoryName := "test" + uuid.NewV4().String()
	categoryDescription := "Test Link Category"
	linkCategoryModel1 := link.WorkItemLinkCategory{
		Name:        categoryName,
		Description: &categoryDescription,
	}
	linkCategory, err := s.workitemLinkCategoryRepo.Create(s.ctx, &linkCategoryModel1)
	require.Nil(s.T(), err)
	s.linkCategoryID = linkCategory.ID

	// create tree topology link type
	treeLinkTypeModel := link.WorkItemLinkType{
		Name:           "Parent child item",
		SourceTypeID:   workitem.SystemBug,
		TargetTypeID:   workitem.SystemBug,
		ForwardName:    "parent of",
		ReverseName:    "child of",
		Topology:       "tree",
		LinkCategoryID: linkCategory.ID,
		SpaceID:        s.testSpace,
	}
	testTreeLinkType, err := s.workitemLinkTypeRepo.Create(s.ctx, &treeLinkTypeModel)
	require.Nil(s.T(), err)
	s.testTreeLinkTypeID = testTreeLinkType.ID
	// create 3 workitems for linking (or not) during the tests
	s.parent1, err = s.createWorkitem(workitem.SystemBug, "Parent 1", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	s.parent1ID, err = strconv.ParseUint(s.parent1.ID, 10, 64)
	s.parent2, err = s.createWorkitem(workitem.SystemBug, "Parent 2", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	s.parent2ID, err = strconv.ParseUint(s.parent2.ID, 10, 64)
	require.Nil(s.T(), err)
	s.child, err = s.createWorkitem(workitem.SystemBug, "Child", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	s.childID, err = strconv.ParseUint(s.child.ID, 10, 64)
	require.Nil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *linkRepoBlackBoxTest) createWorkitem(wiType uuid.UUID, title, state string) (*workitem.WorkItem, error) {
	return s.workitemRepo.Create(
		s.ctx, s.testSpace, wiType,
		map[string]interface{}{
			workitem.SystemTitle: title,
			workitem.SystemState: state,
		}, s.testIdentity.ID)
}

// This creates a parent-child link between two workitems -> parent1 and Child. It tests that when there is an attempt to create another parent (parent2) of child, it should throw an error.
func (s *linkRepoBlackBoxTest) TestDisallowMultipleParents() {
	// create a work item link
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1ID, s.childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	_, err = s.workitemLinkRepo.Create(s.ctx, s.parent2ID, s.childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	// then
	require.NotNil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestExistsLink() {
	// create a parent and a child workitem, then link them together
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	wil, err := s.workitemLinkRepo.Create(s.ctx, s.parent1ID, s.childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	exists, err := s.workitemLinkRepo.Exists(s.ctx, wil.ID.String())
	// then
	require.Nil(s.T(), err)
	require.True(s.T(), exists)
}

// TestCountChildWorkitems tests total number of workitem children returned by list is equal to the total number of workitem children created
// and total number of workitem children in a page are equal to the "limit" specified
func (s *linkRepoBlackBoxTest) TestCountChildWorkitems() {
	// create 3 workitems for linking as children to parent workitem
	child1, err := s.createWorkitem(workitem.SystemBug, "Child 1", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	child1ID, err := strconv.ParseUint(child1.ID, 10, 64)
	require.Nil(s.T(), err)
	child2, err := s.createWorkitem(workitem.SystemBug, "Child 2", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	child2ID, err := strconv.ParseUint(child2.ID, 10, 64)
	require.Nil(s.T(), err)
	child3, err := s.createWorkitem(workitem.SystemBug, "Child 3", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	child3ID, err := strconv.ParseUint(child3.ID, 10, 64)
	require.Nil(s.T(), err)

	// link the children workitems to parent
	_, err = s.workitemLinkRepo.Create(s.ctx, s.parent1ID, child1ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)

	_, err = s.workitemLinkRepo.Create(s.ctx, s.parent1ID, child2ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)

	_, err = s.workitemLinkRepo.Create(s.ctx, s.parent1ID, child3ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)

	offset := 0
	limit := 1
	res, count, err := s.workitemLinkRepo.ListWorkItemChildren(s.ctx, s.parent1.ID, &offset, &limit)
	require.Nil(s.T(), err)
	require.Len(s.T(), res, 1)
	require.Equal(s.T(), 3, int(count))
}

func (s *linkRepoBlackBoxTest) TestWorkItemHasNoChildAfterDeletion() {
	// given
	// create a work item link...
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	wil, err := s.workitemLinkRepo.Create(s.ctx, s.parent1ID, s.childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// ... and remove it
	err = s.workitemLinkRepo.Delete(s.ctx, wil.ID, s.testIdentity.ID)
	require.Nil(s.T(), err)

	// when
	hasChildren, err := s.workitemLinkRepo.WorkItemHasChildren(s.ctx, s.parent1.ID)
	// then
	assert.Nil(s.T(), err)
	assert.False(s.T(), hasChildren)
}

func (s *linkRepoBlackBoxTest) TestValidateTopologyOkNoLink() {
	// given link type exists but no link to child item
	linkType, err := s.workitemLinkTypeRepo.Load(s.ctx, s.testTreeLinkTypeID)
	require.Nil(s.T(), err)
	// when
	err = s.workitemLinkRepo.ValidateTopology(s.ctx, nil, s.childID, *linkType)
	// then: there must be no error because no link exists
	assert.Nil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestValidateTopologyOkLinkExistsButIgnored() {
	// given
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1ID, s.childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	linkType, err := s.workitemLinkTypeRepo.Load(s.ctx, s.testTreeLinkTypeID)
	require.Nil(s.T(), err)
	// when
	err = s.workitemLinkRepo.ValidateTopology(s.ctx, &s.parent1ID, s.childID, *linkType)
	// then: there must be no error because the existing link was ignored
	assert.Nil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestValidateTopologyOkNoLinkWithSameType() {
	// given
	// link 2 workitems together
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1ID, s.childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// use another link type to validate
	linkTypeModel := link.WorkItemLinkType{
		Name:           "foo/bar relationship",
		SourceTypeID:   workitem.SystemBug,
		TargetTypeID:   workitem.SystemBug,
		ForwardName:    "foo",
		ReverseName:    "bar",
		Topology:       "tree",
		LinkCategoryID: s.linkCategoryID,
		SpaceID:        s.testSpace,
	}
	foobarLinkType, err := s.workitemLinkTypeRepo.Create(s.ctx, &linkTypeModel)
	require.Nil(s.T(), err)
	// when
	err = s.workitemLinkRepo.ValidateTopology(s.ctx, nil, s.childID, *foobarLinkType)
	// then: there must be no error because no link of the same type exists
	assert.Nil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestValidateTopologyErrorLinkExists() {
	// given
	// link 2 workitems together
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1ID, s.childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	linkType, err := s.workitemLinkTypeRepo.Load(s.ctx, s.testTreeLinkTypeID)
	require.Nil(s.T(), err)
	// when checking the child *without* excluding the parent item
	err = s.workitemLinkRepo.ValidateTopology(s.ctx, nil, s.childID, *linkType)
	// then: there must be an error because a link of the same type already exists
	assert.NotNil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestValidateTopologyErrorAnotherLinkExists() {
	// given
	// link 2 workitems together
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1ID, s.childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	linkType, err := s.workitemLinkTypeRepo.Load(s.ctx, s.testTreeLinkTypeID)
	require.Nil(s.T(), err)
	parent2, err := s.createWorkitem(workitem.SystemBug, "Parent", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	parent2ID, err := strconv.ParseUint(parent2.ID, 10, 64)
	require.Nil(s.T(), err)
	// when checking the child  while excluding the parent item
	err = s.workitemLinkRepo.ValidateTopology(s.ctx, &parent2ID, s.childID, *linkType)
	// then: there must be an error because a link of the same type already exists with another parent
	assert.NotNil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestCreateLinkOK() {
	// given
	// when
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1ID, s.childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	// then
	require.Nil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestUpdateLinkOK() {
	// given
	// link 2 workitems together
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	wiLink, err := s.workitemLinkRepo.Create(s.ctx, s.parent1ID, s.childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	s.T().Log(fmt.Sprintf("updating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err = s.workitemLinkRepo.Save(s.ctx, *wiLink, s.testIdentity.ID)
	// then
	require.Nil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestCreateLinkErrorOtherParentChildLinkExist() {
	// given
	// link 2 workitems together
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1ID, s.childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when try to link parent#2 to child
	_, err = s.workitemLinkRepo.Create(s.ctx, s.parent2ID, s.childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	// then expect an error because a parent/link relation already exists with the child item
	require.NotNil(s.T(), err)
}
