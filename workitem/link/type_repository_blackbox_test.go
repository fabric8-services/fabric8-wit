package link_test

import (
	"context"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type linkTypeRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	linkTypeRepo link.WorkItemLinkTypeRepository
	clean        func()
	ctx          context.Context
	testSpace    uuid.UUID
	testIdentity account.Identity
	testLinkCat  uuid.UUID
}

func TestRunLinkTypeRepoBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &linkTypeRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *linkTypeRepoBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)
}

func (s *linkTypeRepoBlackBoxTest) SetupTest() {
	s.linkTypeRepo = link.NewWorkItemLinkTypeRepository(s.DB)
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

	// create a link category
	linkCatRepo := link.NewWorkItemLinkCategoryRepository(s.DB)
	linkCat, err := linkCatRepo.Create(s.ctx, &link.WorkItemLinkCategory{
		Name: testsupport.CreateRandomValidTestName("work item link category"),
	})
	require.Nil(s.T(), err)
	s.testLinkCat = linkCat.ID
}

func (s *linkTypeRepoBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *linkTypeRepoBlackBoxTest) Test_WorkItemLinkType_Create() {
	desc := "Bar"
	testWilt := link.WorkItemLinkType{
		Name:           testsupport.CreateRandomValidTestName("work item link type"),
		Description:    &desc,
		ForwardName:    "forward name",
		ReverseName:    "reverse name",
		Topology:       link.TopologyNetwork,
		LinkCategoryID: s.testLinkCat,
		SpaceID:        s.testSpace,
	}

	s.T().Run("creation okay", func(t *testing.T) {
		// given
		wilt := testWilt
		// when
		linkType, err := s.linkTypeRepo.Create(s.ctx, &wilt)
		// then
		require.Nil(t, err)
		require.Equal(t, testWilt.Name, linkType.Name)
		require.Equal(t, testWilt.Description, linkType.Description)
		require.Equal(t, testWilt.ForwardName, linkType.ForwardName)
		require.Equal(t, testWilt.ReverseName, linkType.ReverseName)
		require.Equal(t, testWilt.Topology, linkType.Topology)
		require.Equal(t, testWilt.LinkCategoryID, linkType.LinkCategoryID)
		require.Equal(t, testWilt.SpaceID, linkType.SpaceID)
	})
	s.T().Run("missing link category", func(t *testing.T) {
		// given
		wilt := testWilt
		wilt.LinkCategoryID = uuid.Nil
		// when
		_, err := s.linkTypeRepo.Create(s.ctx, &wilt)
		// then
		require.NotNil(t, err)
	})
	s.T().Run("missing topology", func(t *testing.T) {
		// given
		wilt := testWilt
		wilt.Topology = ""
		// when
		_, err := s.linkTypeRepo.Create(s.ctx, &wilt)
		// then
		require.NotNil(t, err)
	})
	s.T().Run("missing space", func(t *testing.T) {
		// given
		wilt := testWilt
		wilt.SpaceID = uuid.Nil
		// when
		_, err := s.linkTypeRepo.Create(s.ctx, &wilt)
		// then
		require.NotNil(t, err)
	})
}

func (s *linkTypeRepoBlackBoxTest) Test_WorkItemLinkType_Load() {
	// create a link type first and test loading in subtests
	desc := "Bar"
	testWilt := link.WorkItemLinkType{
		Name:           testsupport.CreateRandomValidTestName("work item link type"),
		Description:    &desc,
		ForwardName:    "forward name",
		ReverseName:    "reverse name",
		Topology:       link.TopologyNetwork,
		LinkCategoryID: s.testLinkCat,
		SpaceID:        s.testSpace,
	}
	wilt := testWilt
	linkType, err := s.linkTypeRepo.Create(s.ctx, &wilt)
	require.Nil(s.T(), err)

	s.T().Run("loading okay", func(t *testing.T) {
		// given
		existingLinkTypeID := linkType.ID
		// when
		loadedLinkType, err := s.linkTypeRepo.Load(s.ctx, existingLinkTypeID)
		// then
		require.Nil(t, err)
		require.Equal(t, testWilt.Name, loadedLinkType.Name)
		require.Equal(t, testWilt.Description, loadedLinkType.Description)
		require.Equal(t, testWilt.ForwardName, loadedLinkType.ForwardName)
		require.Equal(t, testWilt.ReverseName, loadedLinkType.ReverseName)
		require.Equal(t, testWilt.Topology, loadedLinkType.Topology)
		require.Equal(t, testWilt.LinkCategoryID, loadedLinkType.LinkCategoryID)
		require.Equal(t, testWilt.SpaceID, loadedLinkType.SpaceID)
	})
	s.T().Run("not existing link type", func(t *testing.T) {
		// given
		notExistingLinkTypeID := uuid.NewV4()
		// when
		_, err := s.linkTypeRepo.Load(s.ctx, notExistingLinkTypeID)
		// then
		require.NotNil(t, err)
	})
}

func (s *linkTypeRepoBlackBoxTest) Test_WorkItemLinkType_Exists() {
	s.T().Run("existing work item link type", func(t *testing.T) {
		// given
		desc := "Bar"
		linkType, err := s.linkTypeRepo.Create(s.ctx, &link.WorkItemLinkType{
			Name:           testsupport.CreateRandomValidTestName("work item link type"),
			Description:    &desc,
			ForwardName:    "forward name",
			ReverseName:    "reverse name",
			Topology:       link.TopologyNetwork,
			LinkCategoryID: s.testLinkCat,
			SpaceID:        s.testSpace,
		})
		require.Nil(t, err)
		// when
		exists, err := s.linkTypeRepo.Exists(s.ctx, linkType.ID)
		// then
		require.Nil(t, err)
		require.True(t, exists)
	})
	s.T().Run("not existing work item link type", func(t *testing.T) {
		// given
		nonExistingWilt := uuid.NewV4()
		// when
		exists, err := s.linkTypeRepo.Exists(s.ctx, nonExistingWilt)
		// then
		require.Nil(t, err)
		require.False(t, exists)
	})
}

// TODO(kwk): Add more tests for List, Delete, Save, ListSourceLinkTypes, and ListTargetLinkTypes in separate PR
