package spacetemplate_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type repoSuite struct {
	gormtestsupport.DBTestSuite
	spaceTemplateRepo spacetemplate.Repository
	witRepo           workitem.WorkItemTypeRepository
	wiltRepo          link.WorkItemLinkTypeRepository
	witgRepo          workitem.WorkItemTypeGroupRepository
}

func TestRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &repoSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *repoSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()

	//s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.spaceTemplateRepo = spacetemplate.NewRepository(s.DB)
	s.witRepo = workitem.NewWorkItemTypeRepository(s.DB)
	s.wiltRepo = link.NewWorkItemLinkTypeRepository(s.DB)
	s.witgRepo = workitem.NewWorkItemTypeGroupRepository(s.DB)
}

func diff(expectedStr, actualStr string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(expectedStr, actualStr, false)
	return dmp.DiffPrettyText(diffs)
}

func (s *repoSuite) TestExists() {
	resource.Require(s.T(), resource.Database)

	s.T().Run("existing template", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1))
		// when
		err := s.spaceTemplateRepo.CheckExists(s.Ctx, fxt.SpaceTemplates[0].ID)
		// then
		require.NoError(t, err)
	})

	s.T().Run("not existing template", func(t *testing.T) {
		// given
		notExistingID := uuid.NewV4()
		// when
		err := s.spaceTemplateRepo.CheckExists(s.Ctx, notExistingID)
		// then
		require.Error(t, err)
	})
}

func (s *repoSuite) TestLoad() {
	resource.Require(s.T(), resource.Database)

	s.T().Run("existing template", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1))
		// when
		actual, err := s.spaceTemplateRepo.Load(s.Ctx, fxt.SpaceTemplates[0].ID)
		// then
		require.NoError(t, err)
		require.NotNil(t, actual)
		require.Equal(t, fxt.SpaceTemplates[0].ID, actual.ID)
		require.Equal(t, fxt.SpaceTemplates[0].Name, actual.Name)
		require.Equal(t, fxt.SpaceTemplates[0].Description, actual.Description)
	})

	s.T().Run("not existing template)", func(t *testing.T) {
		// when
		s, err := s.spaceTemplateRepo.Load(s.Ctx, uuid.NewV4())
		// then
		require.Error(t, err)
		require.Nil(t, s)
	})
}

func (s *repoSuite) TestRepository_List() {
	resource.Require(s.T(), resource.Database)

	s.T().Run("empty or filled", func(t *testing.T) {
		// when
		spaceTemplates, err := s.spaceTemplateRepo.List(s.Ctx)
		// then expect zero or more space templates
		require.NoError(t, err)
		assert.True(t, len(spaceTemplates) >= 0)
	})

	s.T().Run("list 2 space templates", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(2))
		// when
		spaceTemplates, err := s.spaceTemplateRepo.List(s.Ctx)
		// then
		require.NoError(t, err)
		assert.True(t, len(spaceTemplates) >= 2)
		spaceTemplatesToBeFound := id.Map{
			fxt.SpaceTemplates[0].ID: {},
			fxt.SpaceTemplates[0].ID: {},
		}
		for _, st := range spaceTemplates {
			if _, ok := spaceTemplatesToBeFound[st.ID]; ok {
				delete(spaceTemplatesToBeFound, st.ID)
			}
		}
		require.Len(t, spaceTemplatesToBeFound, 0, "these space templates where not found", spaceTemplatesToBeFound)
	})
}
