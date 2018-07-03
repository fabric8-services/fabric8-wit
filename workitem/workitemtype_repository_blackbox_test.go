package workitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/id"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type workItemTypeRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo workitem.WorkItemTypeRepository
}

func TestRunWorkItemTypeRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &workItemTypeRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *workItemTypeRepoBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = workitem.NewWorkItemTypeRepository(s.DB)
	workitem.ClearGlobalWorkItemTypeCache()
}

func (s *workItemTypeRepoBlackBoxTest) TestExists() {
	s.T().Run("wit exists", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypes(1))
		// when
		err := s.repo.CheckExists(s.Ctx, fxt.WorkItemTypes[0].ID)
		// then
		require.NoError(s.T(), err)
	})

	s.T().Run("wit doesn't exist", func(t *testing.T) {
		// given
		nonExistingWorkItemTypeID := uuid.NewV4()
		// when
		err := s.repo.CheckExists(s.Ctx, nonExistingWorkItemTypeID)
		// then
		require.IsType(t, errors.NotFoundError{}, err)
	})
}

func (s *workItemTypeRepoBlackBoxTest) TestList() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypes(4,
			func(fxt *tf.TestFixture, idx int) error {
				// Have one non-planner item based work item type
				if idx == 3 {
					fxt.WorkItemTypes[idx].Path = workitem.LtreeSafeID(fxt.WorkItemTypes[idx].ID)
				}
				return nil
			}),
		)
		// when
		wits, err := s.repo.List(s.Ctx, fxt.SpaceTemplates[0].ID)
		// then
		require.NoError(t, err)
		toBeFound := id.Slice{
			fxt.WorkItemTypes[0].ID,
			fxt.WorkItemTypes[1].ID,
			fxt.WorkItemTypes[2].ID,
			// NOTE: We ARE listing the non-planner item based type.
			fxt.WorkItemTypes[3].ID,
		}.ToMap()
		for _, wit := range wits {
			_, ok := toBeFound[wit.ID]
			assert.True(t, ok, "found unexpected work item type %s", wit.ID)
			delete(toBeFound, wit.ID)
		}
		require.Empty(t, toBeFound, "failed to find work item types: %s", toBeFound)
	})

	s.T().Run("not found for non-existing space", func(t *testing.T) {
		// given
		id := uuid.NewV4()
		// when
		wits, err := s.repo.List(s.Ctx, id)
		// then
		require.Error(t, err)
		require.IsType(t, errors.NotFoundError{}, err)
		require.Nil(t, wits)
	})
}

func (s *workItemTypeRepoBlackBoxTest) TestListPlannerItemTypes() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypes(4,
			func(fxt *tf.TestFixture, idx int) error {
				// Have one non-planner item based work item type
				if idx == 3 {
					fxt.WorkItemTypes[idx].Extends = uuid.Nil
				}
				return nil
			}),
		)
		// when
		wits, err := s.repo.ListPlannerItemTypes(s.Ctx, fxt.SpaceTemplates[0].ID)
		// then
		require.NoError(t, err)
		toBeFound := id.Slice{
			fxt.WorkItemTypes[0].ID,
			fxt.WorkItemTypes[1].ID,
			fxt.WorkItemTypes[2].ID,
			// NOTE: We're NOT listing the non-planner item based type.
		}.ToMap()
		for _, wit := range wits {
			_, ok := toBeFound[wit.ID]
			assert.True(t, ok, "found unexpected work item type %s", wit.ID)
			delete(toBeFound, wit.ID)
		}
		require.Empty(t, toBeFound, "failed to find work item types: %s", toBeFound)
	})

	s.T().Run("not found for non-existing space", func(t *testing.T) {
		// given
		id := uuid.NewV4()
		// when
		wits, err := s.repo.ListPlannerItemTypes(s.Ctx, id)
		// then
		require.Error(t, err)
		require.IsType(t, errors.NotFoundError{}, err)
		require.Nil(t, wits)
	})
}

func (s *workItemTypeRepoBlackBoxTest) TestCreate() {
	s.T().Run("create and load", func(t *testing.T) {
		// Test that we can create two WITs with the same name, the second has a
		// different fields definition
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItemTypes(2,
				tf.SetWorkItemTypeNames("foo_bar", "foo_bar"),
				func(fxt *tf.TestFixture, idx int) error {
					if idx == 1 {
						fxt.WorkItemTypes[1].Fields = map[string]workitem.FieldDefinition{
							"foo": {
								Required:    true,
								Label:       "foo",
								Description: "foo description",
								Type:        &workitem.SimpleType{Kind: workitem.KindFloat},
							},
						}
					}
					return nil
				},
			),
		)
		require.Equal(t, "foo_bar", fxt.WorkItemTypes[0].Name)
		require.Equal(t, "foo_bar", fxt.WorkItemTypes[1].Name)

		require.NotNil(t, fxt.WorkItemTypes[1])
		require.NotNil(t, fxt.WorkItemTypes[1].Fields)
		field := fxt.WorkItemTypes[1].Fields["foo"]
		require.NotNil(t, field)
		assert.Equal(t, workitem.KindFloat, field.Type.GetKind())
		assert.Equal(t, true, field.Required)
	})

	s.T().Run("ok - WIT with base type", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1))
		basetype := "foo.bar"
		baseWit, err := s.repo.Create(s.Ctx, fxt.SpaceTemplates[0].ID, nil, nil, basetype, nil, "fa-bomb", map[string]workitem.FieldDefinition{
			"foo": {
				Required:    true,
				Label:       "foo",
				Description: "foo description",
				Type: &workitem.ListType{
					SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
					ComponentType: workitem.SimpleType{Kind: workitem.KindString}},
			},
		}, true)

		require.NoError(t, err)
		require.NotNil(t, baseWit)
		require.NotNil(t, baseWit.ID)
		extendedWit, err := s.repo.Create(s.Ctx, fxt.SpaceTemplates[0].ID, nil, &baseWit.ID, "foo.baz", nil, "fa-bomb", map[string]workitem.FieldDefinition{}, true)
		require.NoError(t, err)
		require.NotNil(t, extendedWit)
		require.NotNil(t, extendedWit.Fields)
		// the Field 'foo' must exist since it is inherited from the base work item type
		assert.NotNil(t, extendedWit.Fields["foo"])
	})

	s.T().Run("fail - WIT with missing base type", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.SpaceTemplates(1))
		baseTypeID := uuid.NewV4()
		extendedWit, err := s.repo.Create(s.Ctx, fxt.SpaceTemplates[0].ID, nil, &baseTypeID, "foo.baz", nil, "fa-bomb", map[string]workitem.FieldDefinition{}, true)
		// expect an error as the given base type does not exist
		require.Error(t, err)
		require.Nil(t, extendedWit)
	})

	s.T().Run("ok - WIT with list field type", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB,
			tf.WorkItemTypes(2,
				tf.SetWorkItemTypeNames("foo_bar", "foo_bar"),
				func(fxt *tf.TestFixture, idx int) error {
					switch idx {
					case 0:
						fxt.WorkItemTypes[idx].Fields = map[string]workitem.FieldDefinition{
							"foo": {
								Required:    true,
								Label:       "foo",
								Description: "foo description",
								Type: &workitem.ListType{
									SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
									ComponentType: workitem.SimpleType{Kind: workitem.KindString}},
							},
						}
					case 1:
						fxt.WorkItemTypes[idx].Fields = map[string]workitem.FieldDefinition{}
					}
					return nil
				},
			),
		)

		wit, err := s.repo.Load(s.Ctx, fxt.WorkItemTypes[0].ID)
		require.NoError(t, err)
		require.NotNil(t, wit)
		require.NotNil(t, wit.Fields)

		field := wit.Fields["foo"]
		require.NotNil(t, field)
		assert.Equal(t, workitem.KindList, field.Type.GetKind())
		assert.Equal(t, true, field.Required)
	})
}

func (s *workItemTypeRepoBlackBoxTest) TestAddChildTypes() {
	s.T().Run("existing child types", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypes(3))
		// when
		err := s.repo.AddChildTypes(s.Ctx, fxt.WorkItemTypes[0].ID, []uuid.UUID{
			fxt.WorkItemTypes[1].ID,
			fxt.WorkItemTypes[2].ID,
		})
		// then
		require.NoError(t, err)
		wit, err := s.repo.Load(s.Ctx, fxt.WorkItemTypes[0].ID)
		require.NoError(t, err)
		require.Equal(t, wit.ChildTypeIDs, []uuid.UUID{
			fxt.WorkItemTypes[1].ID,
			fxt.WorkItemTypes[2].ID,
		})
	})
	s.T().Run("non existing child types", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemTypes(3))
		// when
		err := s.repo.AddChildTypes(s.Ctx, fxt.WorkItemTypes[0].ID, []uuid.UUID{
			fxt.WorkItemTypes[2].ID,
			uuid.NewV4(),
		})
		// then
		require.Error(t, err)
		wit, err := s.repo.Load(s.Ctx, fxt.WorkItemTypes[0].ID)
		require.NoError(t, err)
		require.Equal(t, []uuid.UUID{fxt.WorkItemTypes[2].ID}, wit.ChildTypeIDs)
	})
}
