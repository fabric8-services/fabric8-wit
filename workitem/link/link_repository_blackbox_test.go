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
	workitemLinkRepo         link.WorkItemLinkRepository
	workitemLinkTypeRepo     link.WorkItemLinkTypeRepository
	workitemLinkCategoryRepo link.WorkItemLinkCategoryRepository
	workitemRepo             workitem.WorkItemRepository
	clean                    func()
	ctx                      context.Context
	testSpace                uuid.UUID
	testIdentity             account.Identity
	testTreeLinkTypeID       uuid.UUID
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

	// create tree topology link type
	linkTypeModel1 := link.WorkItemLinkType{
		Name:           "Parent child item",
		SourceTypeID:   workitem.SystemBug,
		TargetTypeID:   workitem.SystemBug,
		ForwardName:    "parent of",
		ReverseName:    "child of",
		Topology:       "tree",
		LinkCategoryID: linkCategory.ID,
		SpaceID:        s.testSpace,
	}
	TestTreeLinkType, err := s.workitemLinkTypeRepo.Create(s.ctx, &linkTypeModel1)
	require.Nil(s.T(), err)
	s.testTreeLinkTypeID = TestTreeLinkType.ID
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

func (s *linkRepoBlackBoxTest) TestExistsLink() {
	// create a parent and a child workitem, then link them together
	// create 2 workitems for linking
	parent, err := s.createWorkitem(workitem.SystemBug, "Parent", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	parentID, err := strconv.ParseUint(parent.ID, 10, 64)
	require.Nil(s.T(), err)
	// create 3 workitems for linking as children to parent workitem
	child, err := s.createWorkitem(workitem.SystemBug, "Child", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	childID, err := strconv.ParseUint(child.ID, 10, 64)
	require.Nil(s.T(), err)
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	wil, err := s.workitemLinkRepo.Create(s.ctx, parentID, childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	exists, err := s.workitemLinkRepo.Exists(s.ctx, wil.ID.String())
	// then
	require.Nil(s.T(), err)
	require.True(s.T(), exists)
}

// This creates a parent-child link between two workitems -> parent1 and Child. It tests that when there is an attempt to create another parent (parent2) of child, it should throw an error.
func (s *linkRepoBlackBoxTest) TestDisallowMultipleParents() {
	// create 3 workitems for linking
	parent1, err := s.createWorkitem(workitem.SystemBug, "Parent 1", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	parent1ID, err := strconv.ParseUint(parent1.ID, 10, 64)
	parent2, err := s.createWorkitem(workitem.SystemBug, "Parent 2", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	parent2ID, err := strconv.ParseUint(parent2.ID, 10, 64)
	require.Nil(s.T(), err)
	child, err := s.createWorkitem(workitem.SystemBug, "Child", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	childID, err := strconv.ParseUint(child.ID, 10, 64)
	require.Nil(s.T(), err)
	// create a work item link
	_, err = s.workitemLinkRepo.Create(s.ctx, parent1ID, childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	_, err = s.workitemLinkRepo.Create(s.ctx, parent2ID, childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	// then
	require.NotNil(s.T(), err)
}

// TestCountChildWorkitems tests total number of workitem children returned by list is equal to the total number of workitem children created
// and total number of workitem children in a page are equal to the "limit" specified
func (s *linkRepoBlackBoxTest) TestCountChildWorkitems() {
	// create a parent workitem
	parent, err := s.createWorkitem(workitem.SystemBug, "Parent", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	parentID, err := strconv.ParseUint(parent.ID, 10, 64)
	require.Nil(s.T(), err)
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
	_, err = s.workitemLinkRepo.Create(s.ctx, parentID, child1ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)

	_, err = s.workitemLinkRepo.Create(s.ctx, parentID, child2ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)

	_, err = s.workitemLinkRepo.Create(s.ctx, parentID, child3ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)

	offset := 0
	limit := 1
	res, count, err := s.workitemLinkRepo.ListWorkItemChildren(s.ctx, parent.ID, &offset, &limit)
	require.Nil(s.T(), err)
	require.Len(s.T(), res, 1)
	require.Equal(s.T(), 3, int(count))
}

func (s *linkRepoBlackBoxTest) TestWorkItemHasNoChildAfterDeletion() {
	// given
	// create 2 workitems for linking
	parent, err := s.createWorkitem(workitem.SystemBug, "Parent", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	parentID, err := strconv.ParseUint(parent.ID, 10, 64)
	require.Nil(s.T(), err)
	// create 3 workitems for linking as children to parent workitem
	child, err := s.createWorkitem(workitem.SystemBug, "Child", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	childID, err := strconv.ParseUint(child.ID, 10, 64)
	require.Nil(s.T(), err)

	// create a work item link...
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	wil, err := s.workitemLinkRepo.Create(s.ctx, parentID, childID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// ... and remove it
	err = s.workitemLinkRepo.Delete(s.ctx, wil.ID, s.testIdentity.ID)
	require.Nil(s.T(), err)

	// when
	hasChildren, err := s.workitemLinkRepo.WorkItemHasChildren(s.ctx, parent.ID)
	// then
	assert.Nil(s.T(), err)
	assert.False(s.T(), hasChildren)
}
