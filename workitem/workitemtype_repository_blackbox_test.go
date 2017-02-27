package workitem_test

import (
	"golang.org/x/net/context"

	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/workitem"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type workItemTypeRepoBlackBoxTest struct {
	gormsupport.DBTestSuite
	undoScript *gormsupport.DBScript
	repo       workitem.WorkItemTypeRepository
}

func TestRunWorkItemTypeRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &workItemTypeRepoBlackBoxTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

func (s *workItemTypeRepoBlackBoxTest) SetupTest() {
	s.undoScript = &gormsupport.DBScript{}

	gWitRepo := workitem.NewWorkItemTypeRepository(s.DB)
	s.repo = workitem.NewUndoableWorkItemTypeRepository(gWitRepo, s.undoScript)

	db2 := s.DB.Unscoped().Delete(workitem.WorkItemType{Name: "foo_bar"})

	if db2.Error != nil {
		s.T().Fatalf("Could not setup test %s", db2.Error.Error())
		return
	}
	workitem.ClearGlobalWorkItemTypeCache()
}

func (s *workItemTypeRepoBlackBoxTest) TearDownTest() {
	s.undoScript.Run(s.DB)
}

func (s *workItemTypeRepoBlackBoxTest) TestCreateLoadWIT() {

	wit, err := s.repo.Create(context.Background(), nil, nil, "foo_bar", nil, map[string]app.FieldDefinition{
		"foo": {
			Required: true,
			Type:     &app.FieldType{Kind: string(workitem.KindFloat)},
		},
	})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit)
	require.NotNil(s.T(), wit.Data)
	require.NotNil(s.T(), wit.Data.ID)

	// Test that we can create a WIT with the same name as before.
	wit3, err := s.repo.Create(context.Background(), nil, nil, "foo_bar", nil, map[string]app.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit)
	require.NotNil(s.T(), wit.Data)
	require.NotNil(s.T(), wit.Data.ID)

	wit2, err := s.repo.Load(context.Background(), *wit3.Data.ID)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit2)
	require.NotNil(s.T(), wit2.Data)
	require.NotNil(s.T(), wit2.Data.Attributes)
	field := wit2.Data.Attributes.Fields["foo"]
	require.NotNil(s.T(), field)
	assert.Equal(s.T(), string(workitem.KindFloat), field.Type.Kind)
	assert.Equal(s.T(), true, field.Required)
	assert.Nil(s.T(), field.Type.ComponentType)
	assert.Nil(s.T(), field.Type.BaseType)
	assert.Nil(s.T(), field.Type.Values)
}

func (s *workItemTypeRepoBlackBoxTest) TestCreateLoadWITWithList() {
	bt := "string"
	wit, err := s.repo.Create(context.Background(), nil, nil, "foo_bar", nil, map[string]app.FieldDefinition{
		"foo": {
			Required: true,
			Type: &app.FieldType{
				ComponentType: &bt,
				Kind:          string(workitem.KindList),
			},
		},
	})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit)
	require.NotNil(s.T(), wit.Data)
	require.NotNil(s.T(), wit.Data.ID)

	wit3, err := s.repo.Create(context.Background(), nil, nil, "foo_bar", nil, map[string]app.FieldDefinition{})
	require.Nil(s.T(), err)
	require.Nil(s.T(), wit3)
	require.NotNil(s.T(), wit3.Data)
	require.NotNil(s.T(), wit3.Data.ID)

	wit2, err := s.repo.Load(context.Background(), *wit.Data.ID)
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), wit2)
	require.NotNil(s.T(), wit2.Data)
	require.NotNil(s.T(), wit2.Data.Attributes)
	field := wit2.Data.Attributes.Fields["foo"]
	require.NotNil(s.T(), field)
	assert.Equal(s.T(), string(workitem.KindList), field.Type.Kind)
	assert.Equal(s.T(), true, field.Required)
	assert.Nil(s.T(), field.Type.BaseType)
	assert.Nil(s.T(), field.Type.Values)
}

func (s *workItemTypeRepoBlackBoxTest) TestCreateWITWithBaseType() {
	bt := "string"
	basetype := "foo.bar"
	baseWit, err := s.repo.Create(context.Background(), nil, nil, basetype, nil, map[string]app.FieldDefinition{
		"foo": {
			Required: true,
			Type: &app.FieldType{
				ComponentType: &bt,
				Kind:          string(workitem.KindList),
			},
		},
	})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), baseWit)
	require.NotNil(s.T(), baseWit.Data)
	require.NotNil(s.T(), baseWit.Data.ID)
	extendedWit, err := s.repo.Create(context.Background(), nil, baseWit.Data.ID, "foo.baz", nil, map[string]app.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), extendedWit)
	require.NotNil(s.T(), extendedWit.Data)
	require.NotNil(s.T(), extendedWit.Data.Attributes)
	// the Field 'foo' must exist since it is inherited from the base work item type
	assert.NotNil(s.T(), extendedWit.Data.Attributes.Fields["foo"])
}

func (s *workItemTypeRepoBlackBoxTest) TestDoNotCreateWITWithMissingBaseType() {
	baseTypeID := uuid.Nil
	extendedWit, err := s.repo.Create(context.Background(), nil, &baseTypeID, "foo.baz", nil, map[string]app.FieldDefinition{})
	// expect an error as the given base type does not exist
	require.NotNil(s.T(), err)
	require.Nil(s.T(), extendedWit)
}
