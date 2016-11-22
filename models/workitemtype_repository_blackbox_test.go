package models_test

import (
	"golang.org/x/net/context"

	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type WorkItemTypeRepoBlackBoxTest struct {
	gormsupport.DBTestSuite
	undoScript *gormsupport.DBScript
	repo       application.WorkItemTypeRepository
}

func RunWorkItemTypeRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, new(WorkItemTypeRepoBlackBoxTest))
}

func (s *WorkItemTypeRepoBlackBoxTest) SetupTest() {

	resource.Require(s.T(), resource.Database)

	s.undoScript = &gormsupport.DBScript{}
	s.repo = models.NewUndoableWorkItemTypeRepository(models.NewWorkItemTypeRepository(s.DB), s.undoScript)
	db2 := s.DB.Unscoped().Delete(models.WorkItemType{Name: "foo.bar"})

	if db2.Error != nil {
		s.T().Fatalf("Could not setup test %s", db2.Error.Error())
		return
	}
}

func (s *WorkItemTypeRepoBlackBoxTest) TearDownTest() {
	s.undoScript.Run(s.DB)
}

func (s *WorkItemTypeRepoBlackBoxTest) TestCreateLoadWIT(t *testing.T) {

	wit, err := s.repo.Create(context.Background(), nil, "foo.bar", map[string]app.FieldDefinition{
		"foo": app.FieldDefinition{
			Required: true,
			Type:     &app.FieldType{Kind: string(models.KindFloat)},
		},
	})
	assert.Nil(t, err)
	assert.NotNil(t, wit)

	wit3, err := s.repo.Create(context.Background(), nil, "foo.bar", map[string]app.FieldDefinition{})
	assert.IsType(t, models.BadParameterError{}, err)
	assert.Nil(t, wit3)

	wit2, err := s.repo.Load(context.Background(), "foo.bar")
	assert.Nil(t, err)
	require.NotNil(t, wit2)
	field := wit2.Fields["foo"]
	require.NotNil(t, field)
	assert.Equal(t, string(models.KindFloat), field.Type.Kind)
	assert.Equal(t, true, field.Required)
	assert.Nil(t, field.Type.ComponentType)
	assert.Nil(t, field.Type.BaseType)
	assert.Nil(t, field.Type.Values)
}
