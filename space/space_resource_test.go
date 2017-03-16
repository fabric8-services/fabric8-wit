package space_test

import (
	"testing"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/space"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

var testResourceID string = uuid.NewV4().String()
var testPolicyID string = uuid.NewV4().String()
var testPermissionID string = uuid.NewV4().String()
var testResource2ID string = uuid.NewV4().String()
var testPolicyID2 string = uuid.NewV4().String()
var testPermissionID2 string = uuid.NewV4().String()

func TestRunResourceRepoBBTest(t *testing.T) {
	suite.Run(t, &resourceRepoBBTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type resourceRepoBBTest struct {
	gormtestsupport.DBTestSuite
	repo  space.ResourceRepository
	sRepo space.Repository
	clean func()
}

func (test *resourceRepoBBTest) SetupTest() {
	test.repo = space.NewResourceRepository(test.DB)
	test.sRepo = space.NewRepository(test.DB)
	test.clean = cleaner.DeleteCreatedEntities(test.DB)
}

func (test *resourceRepoBBTest) TearDownTest() {
	test.clean()
}

func (test *resourceRepoBBTest) TestCreate() {
	res, _, _ := expectResource(test.create(testResourceID, testPolicyID, testPermissionID), test.requireOk)
	require.Equal(test.T(), res.PolicyID, testPolicyID)
	require.Equal(test.T(), res.PermissionID, testPermissionID)
	require.Equal(test.T(), res.ResourceID, testResourceID)

}

func (test *resourceRepoBBTest) TestLoad() {
	expectResource(test.load(uuid.NewV4()), test.assertNotFound())
	res, _, _ := expectResource(test.create(testResourceID, testPolicyID, testPermissionID), test.requireOk)

	res2, _, _ := expectResource(test.load(res.ID), test.requireOk)
	assert.True(test.T(), (*res).Equal(*res2))
}

func (test *resourceRepoBBTest) TestSaveOk() {
	res, _, _ := expectResource(test.create(testResourceID, testPolicyID, testPermissionID), test.requireOk)

	newResourceID := uuid.NewV4().String()
	newPermissionID := uuid.NewV4().String()
	newPolicyID := uuid.NewV4().String()
	res.PermissionID = newPermissionID
	res.PolicyID = newPolicyID
	res.ResourceID = newResourceID
	res2, _, _ := expectResource(test.save(*res), test.requireOk)
	assert.Equal(test.T(), newPermissionID, res2.PermissionID)
	assert.Equal(test.T(), newPolicyID, res2.PolicyID)
	assert.Equal(test.T(), newResourceID, res2.ResourceID)
}

func (test *resourceRepoBBTest) TestSaveNew() {
	p := space.Resource{
		ID:           uuid.NewV4(),
		ResourceID:   testResourceID,
		PolicyID:     testPolicyID,
		PermissionID: testPermissionID,
	}

	expectResource(test.save(p), test.requireErrorType(errors.NotFoundError{}))
}

func (test *resourceRepoBBTest) TestDelete() {
	res, _, _ := expectResource(test.create(testResourceID, testPolicyID, testPermissionID), test.requireOk)
	expectResource(test.load(res.ID), test.requireOk)
	expectResource(test.delete(res.ID), func(p *space.Resource, s *space.Space, err error) { require.Nil(test.T(), err) })
	expectResource(test.load(res.ID), test.assertNotFound())
	expectResource(test.delete(uuid.NewV4()), test.assertNotFound())
	expectResource(test.delete(uuid.Nil), test.assertNotFound())
}

func (test *resourceRepoBBTest) TestLoadBySpace() {
	expectResource(test.load(uuid.NewV4()), test.assertNotFound())
	res, s, _ := expectResource(test.create(testResourceID, testPolicyID, testPermissionID), test.requireOk)

	res2, _, _ := expectResource(test.loadBySpace(s.ID), test.requireOk)
	assert.True(test.T(), (*res).Equal(*res2))
}

func (test *resourceRepoBBTest) TestLoadByDifferentSpaceFails() {
	test.create(testResourceID, testPolicyID, testPermissionID)

	_, _, err := expectResource(test.loadBySpace(uuid.NewV4()), test.requireErrorType(errors.NotFoundError{}))
	assert.NotNil(test.T(), err)
}

type resourceExpectation func(p *space.Resource, s *space.Space, err error)

func expectResource(f func() (*space.Resource, *space.Space, error), e resourceExpectation) (*space.Resource, *space.Space, error) {
	p, s, err := f()
	e(p, s, err)
	return p, s, errs.WithStack(err)
}

func (test *resourceRepoBBTest) requireOk(p *space.Resource, s *space.Space, err error) {
	assert.NotNil(test.T(), p)
	require.Nil(test.T(), err)
}

func (test *resourceRepoBBTest) assertNotFound() func(p *space.Resource, s *space.Space, err error) {
	return test.assertErrorType(errors.NotFoundError{})
}

func (test *resourceRepoBBTest) assertErrorType(e error) func(p *space.Resource, s *space.Space, e2 error) {
	return func(p *space.Resource, s *space.Space, err error) {
		assert.Nil(test.T(), p)
		assert.IsType(test.T(), e, err, "error was %v", err)
	}
}

func (test *resourceRepoBBTest) requireErrorType(e error) func(p *space.Resource, s *space.Space, err error) {
	return func(p *space.Resource, s *space.Space, err error) {
		assert.Nil(test.T(), p)
		require.IsType(test.T(), e, err)
	}
}

func (test *resourceRepoBBTest) create(resourceID string, policyID string, permissionID string) func() (*space.Resource, *space.Space, error) {
	newSpace := space.Space{
		Name:    uuid.NewV4().String(),
		OwnerId: uuid.Nil,
	}

	newResource := space.Resource{
		ResourceID:   resourceID,
		PolicyID:     policyID,
		PermissionID: permissionID,
	}
	return func() (*space.Resource, *space.Space, error) {
		s, err := test.sRepo.Create(context.Background(), &newSpace)
		require.Nil(test.T(), err)
		newResource.SpaceID = s.ID
		r, err := test.repo.Create(context.Background(), &newResource)
		return r, s, err
	}
}

func (test *resourceRepoBBTest) save(p space.Resource) func() (*space.Resource, *space.Space, error) {
	return func() (*space.Resource, *space.Space, error) {
		r, err := test.repo.Save(context.Background(), &p)
		return r, nil, err
	}
}

func (test *resourceRepoBBTest) load(id uuid.UUID) func() (*space.Resource, *space.Space, error) {
	return func() (*space.Resource, *space.Space, error) {
		r, err := test.repo.Load(context.Background(), id)
		return r, nil, err
	}
}

func (test *resourceRepoBBTest) loadBySpace(spaceID uuid.UUID) func() (*space.Resource, *space.Space, error) {
	return func() (*space.Resource, *space.Space, error) {
		r, err := test.repo.LoadBySpace(context.Background(), &spaceID)
		return r, nil, err
	}
}

func (test *resourceRepoBBTest) delete(id uuid.UUID) func() (*space.Resource, *space.Space, error) {
	return func() (*space.Resource, *space.Space, error) {
		err := test.repo.Delete(context.Background(), id)
		return nil, nil, err
	}
}
