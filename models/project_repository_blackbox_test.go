package models_test

import (
	"testing"

	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/project"
	satoriuuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

var testProject string = satoriuuid.NewV4().String()

func TestRunProjectRepoBBTest(t *testing.T) {
	suite.Run(t, &projectRepoBBTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

type projectRepoBBTest struct {
	gormsupport.DBTestSuite
	undoScript *gormsupport.DBScript
	repo       *models.UndoableProjectRepository
}

func (test *projectRepoBBTest) SetupTest() {
	test.undoScript = &gormsupport.DBScript{}
	test.repo = models.NewUndoableProjectRepository(models.NewProjectRepository(test.DB), test.undoScript)
	test.DB.Unscoped().Delete(&project.Project{}, "Name=?", testProject)
}

func (test *projectRepoBBTest) TearDownTest() {
	test.undoScript.Run(test.DB)
}

func (test *projectRepoBBTest) TestCreate() {
	res, err := test.repo.Create(context.Background(), testProject)
	if err != nil {
		test.T().Fatal(err)
	}
	require.NotNil(test.T(), res)
	require.Equal(test.T(), res.Name, testProject)

	test.failCreate("", models.BadParameterError{})
	test.failCreate(testProject, models.InternalError{})
}

func (test *projectRepoBBTest) failCreate(name string, expected error) {
	res, err := test.repo.Create(context.Background(), name)
	assert.Nil(test.T(), res)
	assert.IsType(test.T(), expected, err)
}

func (test *projectRepoBBTest) TestSaveNew() {
	p := project.Project{
		ID:      satoriuuid.NewV4(),
		Version: 0,
		Name:    testProject,
	}
	_, err := test.repo.Save(context.Background(), p)
	if err == nil {
		test.T().Fatal("Save succeded for new project")
	} else if assert.IsType(test.T(), models.NotFoundError{}, err) {
		test.T().Logf("got expected error: %v", err)
	} else {
		test.T().Errorf("unexpected error: %v", err)
	}
}
