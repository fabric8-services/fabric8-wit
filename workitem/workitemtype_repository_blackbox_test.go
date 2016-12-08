package workitem_test

import (
	"golang.org/x/net/context"

	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/workitem"
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
	s.repo = workitem.NewUndoableWorkItemTypeRepository(workitem.NewWorkItemTypeRepository(s.DB), s.undoScript)
	db2 := s.DB.Unscoped().Delete(workitem.WorkItemType{Name: "foo.bar"})

	if db2.Error != nil {
		s.T().Fatalf("Could not setup test %s", db2.Error.Error())
		return
	}
}

func (s *workItemTypeRepoBlackBoxTest) TearDownTest() {
	s.undoScript.Run(s.DB)
}

func (s *workItemTypeRepoBlackBoxTest) TestCreateLoadWIT() {

	wit, err := s.repo.Create(context.Background(), nil, "foo.bar", map[string]app.FieldDefinition{
		"foo": app.FieldDefinition{
			Required: true,
			Type:     &app.FieldType{Kind: string(workitem.KindFloat)},
		},
	})
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), wit)

	wit3, err := s.repo.Create(context.Background(), nil, "foo.bar", map[string]app.FieldDefinition{})
	assert.IsType(s.T(), errors.BadParameterError{}, err)
	assert.Nil(s.T(), wit3)

	wit2, err := s.repo.Load(context.Background(), "foo.bar")
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
