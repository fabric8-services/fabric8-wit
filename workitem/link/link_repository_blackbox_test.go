package link_test

import (
	"context"
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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type linkRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo               link.WorkItemLinkRepository
	clean              func()
	ctx                context.Context
	testSpace          uuid.UUID
	testIdentity       account.Identity
	TestTreeLinkTypeID uuid.UUID
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
	s.repo = link.NewWorkItemLinkRepository(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "jdoe1", "test")
	s.testIdentity = testIdentity
	require.Nil(s.T(), err)

	// create a space
	spaceRepository := space.NewRepository(s.DB)
	spaceName := "test-space" + uuid.NewV4().String()
	testSpace, err := spaceRepository.Create(s.ctx, &space.Space{
		Name: spaceName,
	})
	s.testSpace = testSpace.ID
	require.Nil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TearDownTest() {
	s.clean()
}

// This creates a parent-child link between two workitems -> Parent1 and Child. It tests that when there is an attempt to create another parent (Parent2) of child, it should throw an error.
func (s *linkRepoBlackBoxTest) TestDisallowMultipleParents() {
	// create 3 workitems for linking
	workitemRepository := workitem.NewWorkItemRepository(s.DB)
	Parent1, err := workitemRepository.Create(
		s.ctx, s.testSpace, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Parent 1",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.testIdentity.ID)
	require.Nil(s.T(), err)
	Parent1ID, err := strconv.ParseUint(Parent1.ID, 10, 64)

	Parent2, err := workitemRepository.Create(
		s.ctx, s.testSpace, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Parent 2",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.testIdentity.ID)
	require.Nil(s.T(), err)
	Parent2ID, err := strconv.ParseUint(Parent2.ID, 10, 64)

	Child, err := workitemRepository.Create(
		s.ctx, s.testSpace, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Child",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.testIdentity.ID)
	require.Nil(s.T(), err)
	ChildID, err := strconv.ParseUint(Child.ID, 10, 64)
	require.Nil(s.T(), err)

	// Create a work item link category
	linkCategoryRepository := link.NewWorkItemLinkCategoryRepository(s.DB)
	categoryName := "test" + uuid.NewV4().String()
	categoryDescription := "Test Link Category"
	linkCategory, err := linkCategoryRepository.Create(s.ctx, &categoryName, &categoryDescription)
	require.Nil(s.T(), err)

	// create tree topology link type
	linkTypeRepository := link.NewWorkItemLinkTypeRepository(s.DB)
	TestTreeLinkType, err := linkTypeRepository.Create(s.ctx, "TestTreeLinkType", nil, workitem.SystemBug, workitem.SystemBug, "foo", "foo", "tree", linkCategory.ID, s.testSpace)
	require.Nil(s.T(), err)
	s.TestTreeLinkTypeID = TestTreeLinkType.ID

	// create a work item link
	linkRepository := link.NewWorkItemLinkRepository(s.DB)
	_, err = linkRepository.Create(s.ctx, Parent1ID, ChildID, s.TestTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)

	_, err = linkRepository.Create(s.ctx, Parent2ID, ChildID, s.TestTreeLinkTypeID, s.testIdentity.ID)
	require.NotNil(s.T(), err)
}
