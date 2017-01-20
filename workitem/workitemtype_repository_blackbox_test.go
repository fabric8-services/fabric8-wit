package workitem_test

import (
	"golang.org/x/net/context"

	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/workitem"
	errs "github.com/pkg/errors"
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
	gWitRepo.ClearCache()
}

func (s *workItemTypeRepoBlackBoxTest) TearDownTest() {
	s.undoScript.Run(s.DB)
}

func (s *workItemTypeRepoBlackBoxTest) TestCreateLoadWIT() {

	wit, err := s.repo.Create(context.Background(), nil, "foo_bar", map[string]app.FieldDefinition{
		"foo": app.FieldDefinition{
			Required: true,
			Type:     &app.FieldType{Kind: string(workitem.KindFloat)},
		},
	})
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), wit)

	wit3, err := s.repo.Create(context.Background(), nil, "foo_bar", map[string]app.FieldDefinition{})
	assert.IsType(s.T(), errors.BadParameterError{}, errs.Cause(err))
	assert.Nil(s.T(), wit3)

	wit2, err := s.repo.Load(context.Background(), "foo_bar")
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), wit2)
	field := wit2.Fields["foo"]
	require.NotNil(s.T(), field)
	assert.Equal(s.T(), string(workitem.KindFloat), field.Type.Kind)
	assert.Equal(s.T(), true, field.Required)
	assert.Nil(s.T(), field.Type.ComponentType)
	assert.Nil(s.T(), field.Type.BaseType)
	assert.Nil(s.T(), field.Type.Values)
}

func (s *workItemTypeRepoBlackBoxTest) TestCreateLoadWITWithList() {
	bt := "string"
	wit, err := s.repo.Create(context.Background(), nil, "foo_bar", map[string]app.FieldDefinition{
		"foo": app.FieldDefinition{
			Required: true,
			Type: &app.FieldType{
				ComponentType: &bt,
				Kind:          string(workitem.KindList),
			},
		},
	})
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), wit)

	wit3, err := s.repo.Create(context.Background(), nil, "foo_bar", map[string]app.FieldDefinition{})
	assert.IsType(s.T(), errors.BadParameterError{}, errs.Cause(err))
	assert.Nil(s.T(), wit3)

	wit2, err := s.repo.Load(context.Background(), "foo_bar")
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), wit2)
	field := wit2.Fields["foo"]
	require.NotNil(s.T(), field)
	assert.Equal(s.T(), string(workitem.KindList), field.Type.Kind)
	assert.Equal(s.T(), true, field.Required)
	assert.Nil(s.T(), field.Type.BaseType)
	assert.Nil(s.T(), field.Type.Values)
}

func (s *workItemTypeRepoBlackBoxTest) TestCreateWITWithBaseType() {
	bt := "string"
	basetype := "foo.bar"
	baseWit, err := s.repo.Create(context.Background(), nil, basetype, map[string]app.FieldDefinition{
		"foo": app.FieldDefinition{
			Required: true,
			Type: &app.FieldType{
				ComponentType: &bt,
				Kind:          string(workitem.KindList),
			},
		},
	})
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), baseWit)
	extendedWit, err := s.repo.Create(context.Background(), &basetype, "foo.baz", map[string]app.FieldDefinition{})
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), extendedWit)
	// the Field 'foo' must exist since it is inherited from the base work item type
	assert.NotNil(s.T(), extendedWit.Fields["foo"])
}

func (s *workItemTypeRepoBlackBoxTest) TestDoNotCreateWITWithMissingBaseType() {
	basetype := "unknown"
	extendedWit, err := s.repo.Create(context.Background(), &basetype, "foo.baz", map[string]app.FieldDefinition{})
	// expect an error as the given base type does not exist
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), extendedWit)
}
