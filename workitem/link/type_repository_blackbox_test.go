package link_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	_ "github.com/lib/pq" // need to import postgres driver
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type typeRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	typeRepo *link.GormWorkItemLinkTypeRepository
}

func TestRunTypeRepoBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &typeRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite()})

}

func (s *typeRepoBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.typeRepo = link.NewWorkItemLinkTypeRepository(s.DB)
	// migration.BootstrapWorkItemLinking(s.Ctx, link.NewWorkItemLinkCategoryRepository(s.DB), space.NewRepository(s.DB), s.typeRepo)
}

func (s *typeRepoBlackBoxTest) TestList() {
	s.T().Run("link types from base space template", func(t *testing.T) {
		baseLinkTypes, err := s.typeRepo.List(s.Ctx, spacetemplate.SystemBaseTemplateID)
		require.NoError(t, err)
		require.NotEmpty(t, baseLinkTypes)
		toBeFound := id.MapFromSlice(id.Slice{
			link.SystemWorkItemLinkTypeBugBlockerID,
			link.SystemWorkItemLinkPlannerItemRelatedID,
			link.SystemWorkItemLinkTypeParentChildID,
		})
		for _, typ := range baseLinkTypes {
			_, ok := toBeFound[typ.ID]
			assert.True(t, ok, "found unexpected work item link type: %s", typ.Name)
			delete(toBeFound, typ.ID)
		}
		require.Empty(t, toBeFound, "failed to find these work item link types: %s", toBeFound)
	})

	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.SpaceTemplates(2),
			tf.WorkItemLinkTypes(6, func(fxt *tf.TestFixture, idx int) error {
				switch idx {
				case 0, 1, 2, 3:
					fxt.WorkItemLinkTypes[idx].SpaceTemplateID = fxt.SpaceTemplates[0].ID
				case 4, 5:
					fxt.WorkItemLinkTypes[idx].SpaceTemplateID = fxt.SpaceTemplates[1].ID
				}
				return nil
			}),
		)
		t.Run("list by 1st space template", func(t *testing.T) {
			// when
			types, err := s.typeRepo.List(s.Ctx, fxt.SpaceTemplates[0].ID)
			// then
			require.NoError(t, err)
			require.NotEmpty(t, types)
			toBeFound := id.MapFromSlice(id.Slice{
				fxt.WorkItemLinkTypes[0].ID,
				fxt.WorkItemLinkTypes[1].ID,
				fxt.WorkItemLinkTypes[2].ID,
				fxt.WorkItemLinkTypes[3].ID,
				// link types from base space template
				link.SystemWorkItemLinkTypeBugBlockerID,
				link.SystemWorkItemLinkPlannerItemRelatedID,
				link.SystemWorkItemLinkTypeParentChildID,
			})
			for _, typ := range types {
				_, ok := toBeFound[typ.ID]
				assert.True(t, ok, "found unexpected work item link type: %s", typ.Name)
				delete(toBeFound, typ.ID)
			}
			require.Empty(t, toBeFound, "failed to find these work item link types: %s", toBeFound)
		})
		t.Run("list by 2nd space", func(t *testing.T) {
			// when
			types, err := s.typeRepo.List(s.Ctx, fxt.SpaceTemplates[1].ID)
			// then
			require.NoError(t, err)
			require.NotEmpty(t, types)
			toBeFound := id.MapFromSlice(id.Slice{
				fxt.WorkItemLinkTypes[4].ID,
				fxt.WorkItemLinkTypes[5].ID,
				// link types from base space template
				link.SystemWorkItemLinkTypeBugBlockerID,
				link.SystemWorkItemLinkPlannerItemRelatedID,
				link.SystemWorkItemLinkTypeParentChildID,
			})
			for _, typ := range types {
				_, ok := toBeFound[typ.ID]
				assert.True(t, ok, "found unexpected work item link type: %s", typ.Name)
				delete(toBeFound, typ.ID)
			}
			require.Empty(t, toBeFound, "failed to find these work item link types: %s", toBeFound)
		})
	})

	s.T().Run("not found", func(t *testing.T) {
		// given
		spaceTemplateID := uuid.NewV4()
		// when
		types, err := s.typeRepo.List(s.Ctx, spaceTemplateID)
		// then
		require.Error(t, err)
		require.IsType(t, errors.NotFoundError{}, err)
		require.Empty(t, types)
	})
}

func (s *typeRepoBlackBoxTest) TestLoad() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinkTypes(1))
		// when
		typ, err := s.typeRepo.Load(s.Ctx, fxt.WorkItemLinkTypes[0].ID)
		// then
		require.NoError(t, err)
		require.NotNil(t, typ)
		require.True(t, fxt.WorkItemLinkTypes[0].Equal(*typ))
	})
	s.T().Run("not found", func(t *testing.T) {
		// given
		linkTypeID := uuid.NewV4()
		// when
		typ, err := s.typeRepo.Load(s.Ctx, linkTypeID)
		// then
		require.Error(t, err)
		require.Nil(t, typ)
	})
}

func (s *typeRepoBlackBoxTest) TestCreate() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1), tf.WorkItemLinkCategories(1))
		id := uuid.NewV4()
		typ := link.WorkItemLinkType{
			ID:              id,
			Name:            id.String(),
			Description:     ptr.String("description for WILT " + id.String()),
			ReverseName:     "reverse name",
			ForwardName:     "forward name",
			Topology:        link.TopologyTree,
			LinkCategoryID:  fxt.WorkItemLinkCategories[0].ID,
			SpaceTemplateID: fxt.SpaceTemplates[0].ID,
		}
		// when
		createdType, err := s.typeRepo.Create(s.Ctx, &typ)
		// then
		require.NoError(t, err)
		require.NotNil(t, createdType)
		require.Equal(t, id, createdType.ID)
		require.True(t, typ.Equal(*createdType))
		// check that loaded type is equal as well
		loadedType, err := s.typeRepo.Load(s.Ctx, createdType.ID)
		require.NoError(t, err)
		require.True(t, typ.Equal(*loadedType))
	})
	s.T().Run("unknown topology (bad parameter error)", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1), tf.WorkItemLinkCategories(1))
		id := uuid.NewV4()
		typ := link.WorkItemLinkType{
			ID:              id,
			Name:            id.String(),
			Description:     ptr.String("description for WILT " + id.String()),
			ReverseName:     "reverse name",
			ForwardName:     "forward name",
			Topology:        link.Topology("foobar"),
			LinkCategoryID:  fxt.WorkItemLinkCategories[0].ID,
			SpaceTemplateID: fxt.SpaceTemplates[0].ID,
		}
		// when
		createdType, err := s.typeRepo.Create(s.Ctx, &typ)
		// then
		require.Error(t, err)
		require.IsType(t, errors.BadParameterError{}, errs.Cause(err))
		require.Nil(t, createdType)
	})
	s.T().Run("empty name (bad parameter error)", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1), tf.WorkItemLinkCategories(1))
		id := uuid.NewV4()
		typ := link.WorkItemLinkType{
			ID:              id,
			Name:            "",
			Description:     ptr.String("description for WILT " + id.String()),
			ReverseName:     "reverse name",
			ForwardName:     "forward name",
			Topology:        link.Topology("foobar"),
			LinkCategoryID:  fxt.WorkItemLinkCategories[0].ID,
			SpaceTemplateID: fxt.SpaceTemplates[0].ID,
		}
		// when
		createdType, err := s.typeRepo.Create(s.Ctx, &typ)
		// then
		require.Error(t, err)
		require.IsType(t, errors.BadParameterError{}, errs.Cause(err))
		require.Nil(t, createdType)
	})
	s.T().Run("unique name violation (data conflict error)", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinkTypes(1))
		typ := *fxt.WorkItemLinkTypes[0]
		typ.ID = uuid.NewV4()
		// when
		createdType, err := s.typeRepo.Create(s.Ctx, &typ)
		// then
		require.Error(t, err)
		require.IsType(t, errors.DataConflictError{}, errs.Cause(err))
		require.Nil(t, createdType)
	})
}

func (s *typeRepoBlackBoxTest) TestCheckExists() {
	s.T().Run("existing", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinkTypes(1))
		// when
		err := s.typeRepo.CheckExists(s.Ctx, fxt.WorkItemLinkTypes[0].ID)
		// then
		require.NoError(t, err)
	})
	s.T().Run("nonexisting", func(t *testing.T) {
		// given
		linkTypeID := uuid.NewV4()
		// when
		err := s.typeRepo.CheckExists(s.Ctx, linkTypeID)
		// then
		require.Error(t, err)
	})
}

func (s *typeRepoBlackBoxTest) TestSave() {
	s.T().Run("version conflict", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinkTypes(1))
		modelToSave := *fxt.WorkItemLinkTypes[0]
		modelToSave.Version = modelToSave.Version + 10
		// when
		savedModel, err := s.typeRepo.Save(s.Ctx, modelToSave)
		// then
		require.Error(t, err)
		require.IsType(t, errors.VersionConflictError{}, errs.Cause(err))
		require.Nil(t, savedModel)
	})
	s.T().Run("space template reference changed (forbidden)", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinkTypes(1), tf.SpaceTemplates(2))
		modelToSave := *fxt.WorkItemLinkTypes[0]
		modelToSave.SpaceTemplateID = fxt.SpaceTemplates[1].ID
		// when
		savedModel, err := s.typeRepo.Save(s.Ctx, modelToSave)
		// then
		require.Error(t, err)
		require.IsType(t, errors.ForbiddenError{}, errs.Cause(err))
		require.Nil(t, savedModel)
	})
	s.T().Run("link type not found", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1), tf.WorkItemLinkCategories(1))
		id := uuid.NewV4()
		modelToSave := link.WorkItemLinkType{
			ID:              id,
			Name:            id.String(),
			Description:     ptr.String("description for WILT " + id.String()),
			ReverseName:     "reverse name",
			ForwardName:     "forward name",
			Topology:        link.TopologyTree,
			LinkCategoryID:  fxt.WorkItemLinkCategories[0].ID,
			SpaceTemplateID: fxt.SpaceTemplates[0].ID,
		}
		// when
		savedModel, err := s.typeRepo.Save(s.Ctx, modelToSave)
		// then
		require.Error(t, err)
		require.IsType(t, errors.NotFoundError{}, errs.Cause(err))
		require.Nil(t, savedModel)
	})
	s.T().Run("unknown topology (bad parameter error)", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinkTypes(1))
		modelToSave := *fxt.WorkItemLinkTypes[0]
		modelToSave.Topology = link.Topology("foobar")
		// when
		savedModel, err := s.typeRepo.Save(s.Ctx, modelToSave)
		// then
		require.Error(t, err)
		require.IsType(t, errors.BadParameterError{}, errs.Cause(err))
		require.Nil(t, savedModel)
	})
	s.T().Run("unique name violation (data conflict error)", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinkTypes(2))
		modelToSave := *fxt.WorkItemLinkTypes[1]
		// take name from other link type to provoke error
		modelToSave.Name = fxt.WorkItemLinkTypes[0].Name
		// when
		createdType, err := s.typeRepo.Save(s.Ctx, modelToSave)
		// then
		require.Error(t, err)
		require.IsType(t, errors.DataConflictError{}, errs.Cause(err))
		require.Nil(t, createdType)
	})
}
