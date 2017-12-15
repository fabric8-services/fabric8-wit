package workitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
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
	suite.Run(t, &workItemTypeRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
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
		err := s.repo.CheckExists(s.Ctx, fxt.WorkItemTypes[0].ID.String())
		// then
		require.NoError(s.T(), err)
	})

	s.T().Run("wit doesn't exist", func(t *testing.T) {
		// given
		nonExistingWorkItemTypeID := uuid.NewV4()
		// when
		err := s.repo.CheckExists(s.Ctx, nonExistingWorkItemTypeID.String())
		// then
		require.IsType(t, errors.NotFoundError{}, err)
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
								Required: true,
								Type:     &workitem.SimpleType{Kind: workitem.KindFloat},
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
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
		basetype := "foo.bar"
		baseWit, err := s.repo.Create(s.Ctx, fxt.Spaces[0].ID, nil, nil, basetype, nil, "fa-bomb", map[string]workitem.FieldDefinition{
			"foo": {
				Required: true,
				Type: &workitem.ListType{
					SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
					ComponentType: workitem.SimpleType{Kind: workitem.KindString}},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, baseWit)
		require.NotNil(t, baseWit.ID)
		extendedWit, err := s.repo.Create(s.Ctx, fxt.Spaces[0].ID, nil, &baseWit.ID, "foo.baz", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
		require.NoError(t, err)
		require.NotNil(t, extendedWit)
		require.NotNil(t, extendedWit.Fields)
		// the Field 'foo' must exist since it is inherited from the base work item type
		assert.NotNil(t, extendedWit.Fields["foo"])
	})

	s.T().Run("fail - WIT with missing base type", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.Spaces(1))
		baseTypeID := uuid.Nil
		extendedWit, err := s.repo.Create(s.Ctx, fxt.Spaces[0].ID, nil, &baseTypeID, "foo.baz", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
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
								Required: true,
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

		wit, err := s.repo.Load(s.Ctx, fxt.WorkItemTypes[0].SpaceID, fxt.WorkItemTypes[0].ID)
		require.NoError(t, err)
		require.NotNil(t, wit)
		require.NotNil(t, wit.Fields)

		field := wit.Fields["foo"]
		require.NotNil(t, field)
		assert.Equal(t, workitem.KindList, field.Type.GetKind())
		assert.Equal(t, true, field.Required)
	})
}
