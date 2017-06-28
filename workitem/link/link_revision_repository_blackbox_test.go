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

func TestRunWorkItemLinkRevisionRepositoryBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &revisionRepositoryBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml")})
}

type revisionRepositoryBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repository         link.WorkItemLinkRepository
	revisionRepository link.RevisionRepository
	clean              func()
	ctx                context.Context
	testIdentity1      account.Identity
	testIdentity2      account.Identity
	testIdentity3      account.Identity
	sourceWorkItemID   uint64
	targetWorkItemID   uint64
	testLinkType1ID    uuid.UUID
	testLinkType2ID    uuid.UUID
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *revisionRepositoryBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)
}

func (s *revisionRepositoryBlackBoxTest) SetupTest() {
	s.repository = link.NewWorkItemLinkRepository(s.DB)
	s.revisionRepository = link.NewRevisionRepository(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	testIdentity1, err := testsupport.CreateTestIdentity(s.DB, "jdoe1", "test")
	require.Nil(s.T(), err)
	s.testIdentity1 = testIdentity1
	testIdentity2, err := testsupport.CreateTestIdentity(s.DB, "jdoe2", "test")
	require.Nil(s.T(), err)
	s.testIdentity2 = testIdentity2
	testIdentity3, err := testsupport.CreateTestIdentity(s.DB, "jdoe3", "test")
	require.Nil(s.T(), err)
	s.testIdentity3 = testIdentity3
	// create a space
	spaceRepository := space.NewRepository(s.DB)
	spaceName := testsupport.CreateRandomValidTestName("test-space")
	testSpace, err := spaceRepository.Create(s.ctx, &space.Space{
		Name: spaceName,
	})
	require.Nil(s.T(), err)
	// create source and target work items before linking them
	workitemRepository := workitem.NewWorkItemRepository(s.DB)
	wi, err := workitemRepository.Create(
		s.ctx, testSpace.ID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Source",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.testIdentity1.ID)
	require.Nil(s.T(), err)
	sourceWorkItemID, err := strconv.ParseUint(wi.ID, 10, 64)
	require.Nil(s.T(), err)
	s.sourceWorkItemID = sourceWorkItemID
	wi, err = workitemRepository.Create(
		s.ctx, testSpace.ID, workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Target",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.testIdentity1.ID)
	require.Nil(s.T(), err)
	targetWorkItemID, err := strconv.ParseUint(wi.ID, 10, 64)
	require.Nil(s.T(), err)
	s.targetWorkItemID = targetWorkItemID

	// Create a work item link category
	linkCategoryRepository := link.NewWorkItemLinkCategoryRepository(s.DB)
	categoryName := testsupport.CreateRandomValidTestName("test-category")
	categoryDescription := "testing work item link revisions"
	linkCategory := link.WorkItemLinkCategory{
		Name:        categoryName,
		Description: &categoryDescription,
	}
	_, err = linkCategoryRepository.Create(s.ctx, &linkCategory)
	require.Nil(s.T(), err)
	// create link types
	linkTypeRepository := link.NewWorkItemLinkTypeRepository(s.DB)
	linkTypeModel1 := link.WorkItemLinkType{
		Name:           "test link type 1",
		ForwardName:    "foo",
		ReverseName:    "foo",
		Topology:       "dependency",
		LinkCategoryID: linkCategory.ID,
		SpaceID:        testSpace.ID,
	}
	linkType1, err := linkTypeRepository.Create(s.ctx, &linkTypeModel1)
	require.Nil(s.T(), err)
	s.testLinkType1ID = linkType1.ID
	linkTypeModel2 := link.WorkItemLinkType{
		Name:           "test link type 2",
		ForwardName:    "bar",
		ReverseName:    "bar",
		Topology:       "dependency",
		LinkCategoryID: linkCategory.ID,
		SpaceID:        testSpace.ID,
	}
	linkType2, err := linkTypeRepository.Create(s.ctx, &linkTypeModel2)
	require.Nil(s.T(), err)
	s.testLinkType2ID = linkType2.ID
}

func (s *revisionRepositoryBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *revisionRepositoryBlackBoxTest) TestStoreWorkItemLinkRevisions() {
	// given
	linkRepository := link.NewWorkItemLinkRepository(s.DB)
	// create a work item link
	workitemLink, err := linkRepository.Create(s.ctx, s.sourceWorkItemID, s.targetWorkItemID, s.testLinkType1ID, s.testIdentity1.ID)
	require.Nil(s.T(), err)
	// modify the work item link
	s.T().Log(fmt.Sprintf("setting workitem link type from %s to %s", workitemLink.LinkTypeID, s.testLinkType2ID))
	workitemLink.LinkTypeID = s.testLinkType2ID
	workitemLink, err = linkRepository.Save(s.ctx, *workitemLink, s.testIdentity2.ID)
	require.Nil(s.T(), err)
	// delete the work item link
	err = linkRepository.Delete(s.ctx, workitemLink.ID, s.testIdentity3.ID)
	require.Nil(s.T(), err)
	// when
	workitemLinkRevisions, err := s.revisionRepository.List(s.ctx, workitemLink.ID)
	// then
	require.Nil(s.T(), err)
	require.Len(s.T(), workitemLinkRevisions, 3)
	// revision 1
	revision1 := workitemLinkRevisions[0]
	assert.Equal(s.T(), workitemLink.ID, revision1.WorkItemLinkID)
	assert.Equal(s.T(), link.RevisionTypeCreate, revision1.Type)
	assert.Equal(s.T(), s.testIdentity1.ID, revision1.ModifierIdentity)
	assert.Equal(s.T(), s.sourceWorkItemID, revision1.WorkItemLinkSourceID)
	assert.Equal(s.T(), s.targetWorkItemID, revision1.WorkItemLinkTargetID)
	assert.Equal(s.T(), s.testLinkType1ID, revision1.WorkItemLinkTypeID)
	// revision 2
	revision2 := workitemLinkRevisions[1]
	assert.Equal(s.T(), workitemLink.ID, revision2.WorkItemLinkID)
	assert.Equal(s.T(), link.RevisionTypeUpdate, revision2.Type)
	assert.Equal(s.T(), s.testIdentity2.ID, revision2.ModifierIdentity)
	assert.Equal(s.T(), s.sourceWorkItemID, revision2.WorkItemLinkSourceID)
	assert.Equal(s.T(), s.targetWorkItemID, revision2.WorkItemLinkTargetID)
	assert.Equal(s.T(), s.testLinkType2ID, revision2.WorkItemLinkTypeID)
	// revision 3
	revision3 := workitemLinkRevisions[2]
	assert.Equal(s.T(), workitemLink.ID, revision3.WorkItemLinkID)
	assert.Equal(s.T(), link.RevisionTypeDelete, revision3.Type)
	assert.Equal(s.T(), s.testIdentity3.ID, revision3.ModifierIdentity)
	assert.Equal(s.T(), s.sourceWorkItemID, revision3.WorkItemLinkSourceID)
	assert.Equal(s.T(), s.targetWorkItemID, revision3.WorkItemLinkTargetID)
	assert.Equal(s.T(), s.testLinkType2ID, revision3.WorkItemLinkTypeID)
}

func (s *revisionRepositoryBlackBoxTest) TestStoreWorkItemLinkRevisionsWhenDeletingWorkItem() {
	// given
	linkRepository := link.NewWorkItemLinkRepository(s.DB)
	// create a work item link
	workitemLink, err := linkRepository.Create(s.ctx, s.sourceWorkItemID, s.targetWorkItemID, s.testLinkType1ID, s.testIdentity1.ID)
	require.Nil(s.T(), err)
	// delete the source work item
	sourceWorkItemID := strconv.FormatUint(s.sourceWorkItemID, 10)
	err = linkRepository.DeleteRelatedLinks(s.ctx, sourceWorkItemID, s.testIdentity3.ID)
	require.Nil(s.T(), err)
	// when
	workitemLinkRevisions, err := s.revisionRepository.List(s.ctx, workitemLink.ID)
	// then
	require.Nil(s.T(), err)
	require.Len(s.T(), workitemLinkRevisions, 2)
	// revision 1
	revision1 := workitemLinkRevisions[0]
	assert.Equal(s.T(), workitemLink.ID, revision1.WorkItemLinkID)
	assert.Equal(s.T(), link.RevisionTypeCreate, revision1.Type)
	assert.Equal(s.T(), s.testIdentity1.ID, revision1.ModifierIdentity)
	assert.Equal(s.T(), s.sourceWorkItemID, revision1.WorkItemLinkSourceID)
	assert.Equal(s.T(), s.targetWorkItemID, revision1.WorkItemLinkTargetID)
	assert.Equal(s.T(), s.testLinkType1ID, revision1.WorkItemLinkTypeID)
	// revision 2
	revision2 := workitemLinkRevisions[1]
	assert.Equal(s.T(), workitemLink.ID, revision2.WorkItemLinkID)
	assert.Equal(s.T(), link.RevisionTypeDelete, revision2.Type)
	assert.Equal(s.T(), s.testIdentity3.ID, revision2.ModifierIdentity)
	assert.Equal(s.T(), s.sourceWorkItemID, revision2.WorkItemLinkSourceID)
	assert.Equal(s.T(), s.targetWorkItemID, revision2.WorkItemLinkTargetID)
	assert.Equal(s.T(), s.testLinkType1ID, revision2.WorkItemLinkTypeID)
}
