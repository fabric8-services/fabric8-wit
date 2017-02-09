package space_test

import (
	"testing"

	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/space"
	errs "github.com/pkg/errors"
	satoriuuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

var testSpace string = satoriuuid.NewV4().String()
var testSpace2 string = satoriuuid.NewV4().String()

func TestRunRepoBBTest(t *testing.T) {
	suite.Run(t, &repoBBTest{DBTestSuite: gormsupport.NewDBTestSuite("../" + config.GetDefaultConfigurationFile())})
}

type repoBBTest struct {
	gormsupport.DBTestSuite
	repo  space.Repository
	clean func()
}

func (test *repoBBTest) SetupTest() {
	test.repo = space.NewRepository(test.DB)
	test.clean = cleaner.DeleteCreatedEntities(test.DB)
}

func (test *repoBBTest) TearDownTest() {
	test.clean()
}

func (test *repoBBTest) TestCreate() {
	res, _ := expectSpace(test.create(testSpace), test.requireOk)

	require.Equal(test.T(), res.Name, testSpace)

	expectSpace(test.create(""), test.assertBadParameter())
	expectSpace(test.create(testSpace), test.assertBadParameter())
}

func (test *repoBBTest) TestLoad() {
	expectSpace(test.load(satoriuuid.NewV4()), test.assertNotFound())
	res, _ := expectSpace(test.create(testSpace), test.requireOk)

	res2, _ := expectSpace(test.load(res.ID), test.requireOk)
	assert.True(test.T(), (*res).Equal(*res2))
}

func (test *repoBBTest) TestSaveOk() {
	res, _ := expectSpace(test.create(testSpace), test.requireOk)

	newName := satoriuuid.NewV4().String()
	res.Name = newName
	res2, _ := expectSpace(test.save(*res), test.requireOk)
	assert.Equal(test.T(), newName, res2.Name)
}

func (test *repoBBTest) TestSaveFail() {
	p1, _ := expectSpace(test.create(testSpace), test.requireOk)
	p2, _ := expectSpace(test.create(testSpace2), test.requireOk)

	p1.Name = ""
	expectSpace(test.save(*p1), test.assertBadParameter())

	p1.Name = p2.Name
	expectSpace(test.save(*p1), test.assertBadParameter())
}

func (test *repoBBTest) TestSaveNew() {
	p := space.Space{
		ID:      satoriuuid.NewV4(),
		Version: 0,
		Name:    testSpace,
	}

	expectSpace(test.save(p), test.requireErrorType(errors.NotFoundError{}))
}

func (test *repoBBTest) TestDelete() {
	res, _ := expectSpace(test.create(testSpace), test.requireOk)
	expectSpace(test.load(res.ID), test.requireOk)
	expectSpace(test.delete(res.ID), func(p *space.Space, err error) { require.Nil(test.T(), err) })
	expectSpace(test.load(res.ID), test.assertNotFound())
	expectSpace(test.delete(satoriuuid.NewV4()), test.assertNotFound())
	expectSpace(test.delete(satoriuuid.Nil), test.assertNotFound())
}

func (test *repoBBTest) TestList() {
	_, orgCount, _ := test.list(nil, nil)
	expectSpace(test.create(testSpace), test.requireOk)
	_, newCount, _ := test.list(nil, nil)
	assert.Equal(test.T(), orgCount+1, newCount)
}

func (test *repoBBTest) TestListDoNotReturnPointerToSameObject() {
	expectSpace(test.create(testSpace), test.requireOk)
	expectSpace(test.create(testSpace2), test.requireOk)
	spaces, newCount, _ := test.list(nil, nil)
	assert.True(test.T(), newCount >= 2)
	assert.True(test.T(), spaces[0].Name != spaces[1].Name)
}

type spaceExpectation func(p *space.Space, err error)

func expectSpace(f func() (*space.Space, error), e spaceExpectation) (*space.Space, error) {
	p, err := f()
	e(p, err)
	return p, errs.WithStack(err)
}

func (test *repoBBTest) requireOk(p *space.Space, err error) {
	assert.NotNil(test.T(), p)
	require.Nil(test.T(), err)
}

func (test *repoBBTest) assertNotFound() func(p *space.Space, err error) {
	return test.assertErrorType(errors.NotFoundError{})
}
func (test *repoBBTest) assertBadParameter() func(p *space.Space, err error) {
	return test.assertErrorType(errors.BadParameterError{})
}

func (test *repoBBTest) assertErrorType(e error) func(p *space.Space, e2 error) {
	return func(p *space.Space, err error) {
		assert.Nil(test.T(), p)
		assert.IsType(test.T(), e, err, "error was %v", err)
	}
}

func (test *repoBBTest) requireErrorType(e error) func(p *space.Space, err error) {
	return func(p *space.Space, err error) {
		assert.Nil(test.T(), p)
		require.IsType(test.T(), e, err)
	}
}

func (test *repoBBTest) create(name string) func() (*space.Space, error) {
	newSpace := space.Space{
		Name: name,
	}
	return func() (*space.Space, error) { return test.repo.Create(context.Background(), &newSpace) }
}

func (test *repoBBTest) save(p space.Space) func() (*space.Space, error) {
	return func() (*space.Space, error) { return test.repo.Save(context.Background(), &p) }
}

func (test *repoBBTest) load(id satoriuuid.UUID) func() (*space.Space, error) {
	return func() (*space.Space, error) { return test.repo.Load(context.Background(), id) }
}

func (test *repoBBTest) delete(id satoriuuid.UUID) func() (*space.Space, error) {
	return func() (*space.Space, error) { return nil, test.repo.Delete(context.Background(), id) }
}

func (test *repoBBTest) list(start *int, length *int) ([]*space.Space, uint64, error) {
	return test.repo.List(context.Background(), start, length)
}
