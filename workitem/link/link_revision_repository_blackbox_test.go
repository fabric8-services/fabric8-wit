package link_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

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
}

func (s *revisionRepositoryBlackBoxTest) TestList() {
	revRepo := link.NewRevisionRepository(s.DB)

	s.T().Run("ok - when deleting work item link", func(t *testing.T) {
		// given a work item link
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinks(1), tf.Identities(3))
		linkRepository := link.NewWorkItemLinkRepository(s.DB)
		// delete the source work item
		err := linkRepository.DeleteRelatedLinks(s.Ctx, fxt.WorkItems[0].ID, fxt.Identities[2].ID)
		require.NoError(t, err)
		// when
		workitemLinkRevisions, err := revRepo.List(s.Ctx, fxt.WorkItemLinks[0].ID)
		// then
		require.NoError(t, err)
		require.Len(t, workitemLinkRevisions, 2)
		// revision 1
		revision1 := workitemLinkRevisions[0]
		assert.Equal(t, fxt.WorkItemLinks[0].ID, revision1.WorkItemLinkID)
		assert.Equal(t, link.RevisionTypeCreate, revision1.Type)
		assert.Equal(t, fxt.Identities[0].ID, revision1.ModifierIdentity)
		assert.Equal(t, fxt.WorkItemLinks[0].SourceID, revision1.WorkItemLinkSourceID)
		assert.Equal(t, fxt.WorkItemLinks[0].TargetID, revision1.WorkItemLinkTargetID)
		assert.Equal(t, fxt.WorkItemLinkTypes[0].ID, revision1.WorkItemLinkTypeID)
		// revision 2
		revision2 := workitemLinkRevisions[1]
		assert.Equal(t, fxt.WorkItemLinks[0].ID, revision2.WorkItemLinkID)
		assert.Equal(t, link.RevisionTypeDelete, revision2.Type)
		assert.Equal(t, fxt.Identities[2].ID, revision2.ModifierIdentity)
		assert.Equal(t, fxt.WorkItemLinks[0].SourceID, revision2.WorkItemLinkSourceID)
		assert.Equal(t, fxt.WorkItemLinks[0].TargetID, revision2.WorkItemLinkTargetID)
		assert.Equal(t, fxt.WorkItemLinkTypes[0].ID, revision2.WorkItemLinkTypeID)
	})
}
