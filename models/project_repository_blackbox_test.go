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
var testProject2 string = satoriuuid.NewV4().String()

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
	test.DB.Unscoped().Delete(&project.Project{}, "Name=?", testProject2)
}

func (test *projectRepoBBTest) TearDownTest() {
	test.undoScript.Run(test.DB)
}

func (test *projectRepoBBTest) TestCreate() {
	res, _ := expectProject(test.create(testProject), test.requireOk)

	require.Equal(test.T(), res.Name, testProject)

	expectProject(test.create(""), test.assertBadParameter())
	expectProject(test.create(testProject), test.assertBadParameter())
}

func (test *projectRepoBBTest) TestLoad() {
	expectProject(test.load(satoriuuid.NewV4()), test.assertNotFound())
	res, _ := expectProject(test.create(testProject), test.requireOk)

	res2, _ := expectProject(test.load(res.ID), test.requireOk)
	assert.True(test.T(), (*res).Equal(*res2))
}

func (test *projectRepoBBTest) TestSaveOk() {
	res, _ := expectProject(test.create(testProject), test.requireOk)

	newName := satoriuuid.NewV4().String()
	res.Name = newName
	res2, _ := expectProject(test.save(*res), test.requireOk)
	assert.Equal(test.T(), newName, res2.Name)
}

func (test *projectRepoBBTest) TestSaveFail() {
	p1, _ := expectProject(test.create(testProject), test.requireOk)
	p2, _ := expectProject(test.create(testProject2), test.requireOk)

	p1.Name = ""
	expectProject(test.save(*p1), test.assertBadParameter())

	p1.Name = p2.Name
	expectProject(test.save(*p1), test.assertBadParameter())
}

func (test *projectRepoBBTest) TestSaveNew() {
	p := project.Project{
		ID:      satoriuuid.NewV4(),
		Version: 0,
		Name:    testProject,
	}

	expectProject(test.save(p), test.requireErrorType(models.NotFoundError{}))
}

func (test *projectRepoBBTest) TestDelete() {
	res, _ := expectProject(test.create(testProject), test.requireOk)
	expectProject(test.load(res.ID), test.requireOk)
	expectProject(test.delete(res.ID), func(p *project.Project, err error) { require.Nil(test.T(), err) })
	expectProject(test.load(res.ID), test.assertNotFound())
	expectProject(test.delete(satoriuuid.NewV4()), test.assertNotFound())
}

func (test *projectRepoBBTest) TestList() {
	_, orgCount := test.findProjectNamed(testProject)
	p1, _ := expectProject(test.create(testProject), test.requireOk)
	p2, newCount := test.findProjectNamed(testProject)
	assert.Equal(test.T(), orgCount+1, newCount)
	assert.True(test.T(), p1.Equal(*p2))
}

func (test *projectRepoBBTest) findProjectNamed(name string) (*project.Project, uint64) {
	res, count, err := test.list(nil, nil)
	if err != nil {
		return nil, 0
	}
	start := 0
	for start < int(count) {
		for _, value := range res {
			if value.Name == name {
				return &value, count
			}
		}
		start += len(res)
		res, count, err = test.list(&start, nil)
	}
	return nil, count
}

type projectExpectation func(p *project.Project, err error)

func expectProject(f func() (*project.Project, error), e projectExpectation) (*project.Project, error) {
	p, err := f()
	e(p, err)
	return p, err
}

func (test *projectRepoBBTest) requireOk(p *project.Project, err error) {
	assert.NotNil(test.T(), p)
	require.Nil(test.T(), err)
}

func (test *projectRepoBBTest) assertNotFound() func(p *project.Project, err error) {
	return test.assertErrorType(models.NotFoundError{})
}
func (test *projectRepoBBTest) assertBadParameter() func(p *project.Project, err error) {
	return test.assertErrorType(models.BadParameterError{})
}

func (test *projectRepoBBTest) assertErrorType(e error) func(p *project.Project, e2 error) {
	return func(p *project.Project, err error) {
		assert.Nil(test.T(), p)
		assert.IsType(test.T(), e, err)
	}
}

func (test *projectRepoBBTest) requireErrorType(e error) func(p *project.Project, err error) {
	return func(p *project.Project, err error) {
		assert.Nil(test.T(), p)
		require.IsType(test.T(), e, err)
	}
}

func (test *projectRepoBBTest) create(name string) func() (*project.Project, error) {
	return func() (*project.Project, error) { return test.repo.Create(context.Background(), name) }
}

func (test *projectRepoBBTest) save(p project.Project) func() (*project.Project, error) {
	return func() (*project.Project, error) { return test.repo.Save(context.Background(), p) }
}

func (test *projectRepoBBTest) load(id satoriuuid.UUID) func() (*project.Project, error) {
	return func() (*project.Project, error) { return test.repo.Load(context.Background(), id) }
}

func (test *projectRepoBBTest) delete(id satoriuuid.UUID) func() (*project.Project, error) {
	return func() (*project.Project, error) { return nil, test.repo.Delete(context.Background(), id) }
}

func (test *projectRepoBBTest) list(start *int, length *int) ([]project.Project, uint64, error) {
	return test.repo.List(context.Background(), start, length)
}
