package workitem_test

import (
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRunRevisionRepositoryBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemRevisionRepositoryBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type workItemRevisionRepositoryBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repository         workitem.WorkItemRepository
	revisionRepository workitem.RevisionRepository
}

func (s *workItemRevisionRepositoryBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repository = workitem.NewWorkItemRepository(s.DB)
	s.revisionRepository = workitem.NewRevisionRepository(s.DB)
}

func (s *workItemRevisionRepositoryBlackBoxTest) TestStoreRevisions() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItems(1, tf.SetWorkItemTitles("Title")), tf.Identities(3))
		wi := fxt.WorkItems[0]
		// modify the workitem
		wi.Fields[workitem.SystemTitle] = "Updated Title"
		wi.Fields[workitem.SystemState] = workitem.SystemStateOpen
		wi, err := s.repository.Save(s.Ctx, wi.SpaceID, *wi, fxt.Identities[1].ID)
		require.NoError(t, err)
		// modify again the workitem
		wi.Fields[workitem.SystemTitle] = "Updated Title2"
		wi.Fields[workitem.SystemState] = workitem.SystemStateInProgress
		wi, err = s.repository.Save(s.Ctx, wi.SpaceID, *wi, fxt.Identities[1].ID)
		require.NoError(t, err)
		// delete the workitem
		err = s.repository.Delete(s.Ctx, wi.ID, fxt.Identities[2].ID)
		require.NoError(t, err)
		// when
		revisions, err := s.revisionRepository.List(s.Ctx, wi.ID)
		// then
		require.NoError(t, err)
		require.Len(t, revisions, 4)
		// revision 1
		revision1 := revisions[0]
		s.T().Log(fmt.Sprintf("Work item revision 1: modifier:%s type: %v version:%v fields:%v", revision1.ModifierIdentity, revision1.Type, revision1.WorkItemVersion, revision1.WorkItemFields))
		assert.Equal(t, wi.ID, revision1.WorkItemID)
		assert.Equal(t, workitem.RevisionTypeCreate, revision1.Type)
		assert.Equal(t, wi.Type, revision1.WorkItemTypeID)
		assert.Equal(t, fxt.Identities[0].ID, revision1.ModifierIdentity)
		require.NotNil(t, revision1.WorkItemFields)
		assert.Equal(t, "Title", revision1.WorkItemFields[workitem.SystemTitle])
		assert.Equal(t, workitem.SystemStateNew, revision1.WorkItemFields[workitem.SystemState])
		// revision 2
		revision2 := revisions[1]
		t.Log(fmt.Sprintf("Work item revision 2: modifier:%s type: %v version:%v fields:%v", revision2.ModifierIdentity, revision2.Type, revision2.WorkItemVersion, revision2.WorkItemFields))
		assert.Equal(t, wi.ID, revision2.WorkItemID)
		assert.Equal(t, workitem.RevisionTypeUpdate, revision2.Type)
		assert.Equal(t, wi.Type, revision2.WorkItemTypeID)
		assert.Equal(t, fxt.Identities[1].ID, revision2.ModifierIdentity)
		require.NotNil(t, revision2.WorkItemFields)
		assert.Equal(t, "Updated Title", revision2.WorkItemFields[workitem.SystemTitle])
		assert.Equal(t, workitem.SystemStateOpen, revision2.WorkItemFields[workitem.SystemState])
		// revision 3
		revision3 := revisions[2]
		t.Log(fmt.Sprintf("Work item revision 3: modifier:%s type: %v version:%v fields:%v", revision3.ModifierIdentity, revision3.Type, revision3.WorkItemVersion, revision3.WorkItemFields))
		assert.Equal(t, wi.ID, revision3.WorkItemID)
		assert.Equal(t, workitem.RevisionTypeUpdate, revision3.Type)
		assert.Equal(t, wi.Type, revision3.WorkItemTypeID)
		require.NotNil(t, revision3.WorkItemFields)
		assert.Equal(t, "Updated Title2", revision3.WorkItemFields[workitem.SystemTitle])
		assert.Equal(t, workitem.SystemStateInProgress, revision3.WorkItemFields[workitem.SystemState])
		assert.Equal(t, fxt.Identities[1].ID, revision3.ModifierIdentity)
		// revision 4
		revision4 := revisions[3]
		t.Log(fmt.Sprintf("Work item revision 4: modifier:%s type: %v version:%v fields:%v", revision4.ModifierIdentity, revision4.Type, revision4.WorkItemVersion, revision4.WorkItemFields))
		assert.Equal(t, wi.ID, revision4.WorkItemID)
		assert.Equal(t, workitem.RevisionTypeDelete, revision4.Type)
		assert.Equal(t, wi.Type, revision4.WorkItemTypeID)
		assert.Equal(t, fxt.Identities[2].ID, revision4.ModifierIdentity)
		require.Empty(t, revision4.WorkItemFields)
	})
}
