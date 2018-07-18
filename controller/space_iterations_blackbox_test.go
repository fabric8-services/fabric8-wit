package controller_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SpaceIterationControllerTestSuite struct {
	gormtestsupport.DBTestSuite
	testIdentity account.Identity
	testDir      string
}

func TestSpaceIterationController(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &SpaceIterationControllerTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *SpaceIterationControllerTestSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "SpaceIterationControllerTestSuite user", "test provider")
	require.NoError(s.T(), err)
	s.testIdentity = *testIdentity
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	s.Ctx = goa.NewContext(context.Background(), nil, req, params)
	s.testDir = filepath.Join("test-files", "space_iterations")
}

func (s *SpaceIterationControllerTestSuite) SecuredController(identity ...account.Identity) (*goa.Service, *SpaceIterationsController) {
	i := testsupport.TestIdentity
	if identity != nil && len(identity) > 0 {
		i = identity[0]
	}
	svc := testsupport.ServiceAsUser("Iteration-Service", i)
	return svc, NewSpaceIterationsController(svc, s.GormDB, s.Configuration)
}

func (s *SpaceIterationControllerTestSuite) SecuredControllerWithIdentity(idn *account.Identity) (*goa.Service, *SpaceIterationsController) {
	svc := testsupport.ServiceAsUser("Iteration-Service", *idn)
	return svc, NewSpaceIterationsController(svc, s.GormDB, s.Configuration)
}

func (s *SpaceIterationControllerTestSuite) UnSecuredController() (*goa.Service, *SpaceIterationsController) {
	svc := goa.New("Iteration-Service")
	return svc, NewSpaceIterationsController(svc, s.GormDB, s.Configuration)
}

func (s *SpaceIterationControllerTestSuite) TestCreate() {
	s.T().Run("success", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			// given
			ci := newCreateSpaceIterationPayload("Sprint #42", nil)
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment())
			svc := testsupport.ServiceAsUser("Iteration-Service", *fxt.Identities[0])
			ctrl := NewSpaceIterationsController(svc, s.GormDB, s.Configuration)
			// when
			resp, iter := test.CreateSpaceIterationsCreated(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, ci)
			// then
			compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "create", "ok.payload.res.golden.json"), iter)
			compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "create", "ok.headers.res.golden.json"), resp)
		})
		t.Run("with force active", func(t *testing.T) {
			// given
			ci := newCreateSpaceIterationPayload("Sprint #43", nil)
			ci.Data.Attributes.UserActive = ptr.Bool(true)
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment())
			svc := testsupport.ServiceAsUser("Iteration-Service", *fxt.Identities[0])
			ctrl := NewSpaceIterationsController(svc, s.GormDB, s.Configuration)
			// when
			resp, iter := test.CreateSpaceIterationsCreated(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, ci)
			// then
			compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "create", "ok_with_force_active.payload.res.golden.json"), iter)
			compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "create", "ok_with_force_active.headers.res.golden.json"), resp)
		})

		t.Run("with optional values", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment())
			iterationName := "Sprint #22"
			iterationDesc := "testing description"
			ci := newCreateSpaceIterationPayload(iterationName, &iterationDesc)
			svc, ctrl := s.SecuredController(*fxt.Identities[0])
			// when
			_, c := test.CreateSpaceIterationsCreated(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, ci)
			// then
			assert.NotNil(t, c.Data.ID)
			assert.NotNil(t, c.Data.Relationships.Space)
			assert.Equal(t, fxt.Spaces[0].ID.String(), *c.Data.Relationships.Space.Data.ID)
			assert.Equal(t, *c.Data.Attributes.Name, iterationName)
			assert.Equal(t, *c.Data.Attributes.Description, iterationDesc)

			// create another Iteration with nil description
			iterationName2 := "Sprint #23"
			ci = newCreateSpaceIterationPayload(iterationName2, nil)
			_, c = test.CreateSpaceIterationsCreated(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, ci)
			assert.Equal(t, *c.Data.Attributes.Name, iterationName2)
			assert.Nil(t, c.Data.Attributes.Description)
		})
	})

	s.T().Run("failure", func(t *testing.T) {
		t.Run("missing space", func(t *testing.T) {
			// given
			ci := newCreateSpaceIterationPayload("Sprint #21", nil)
			svc, ctrl := s.SecuredController()
			// when/then
			test.CreateSpaceIterationsNotFound(s.T(), svc.Context, svc, ctrl, uuid.NewV4(), ci)
		})

		t.Run("duplicate iteration", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, s.DB,
				tf.Identities(1),
				tf.Iterations(2, func(fxt *tf.TestFixture, idx int) error {
					if idx == 1 {
						fxt.Iterations[idx].MakeChildOf(*fxt.Iterations[0])
					}
					return nil
				}))
			iterationName := fxt.Iterations[1].Name
			iterationDesc := "duplicate iteration description"
			ci := newCreateSpaceIterationPayload(iterationName, &iterationDesc)
			svc, ctrl := s.SecuredController(*fxt.Identities[0])
			// when
			test.CreateSpaceIterationsConflict(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, ci)
		})

		t.Run("not authorized", func(t *testing.T) {
			// given
			ci := newCreateSpaceIterationPayload("Sprint #21", nil)
			svc, ctrl := s.UnSecuredController()
			// when/then
			test.CreateSpaceIterationsUnauthorized(s.T(), svc.Context, svc, ctrl, uuid.NewV4(), ci)
		})

		t.Run("forbidden", func(t *testing.T) {
			ci := newCreateSpaceIterationPayload("Sprint #21", nil)
			fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.Identities(2))

			spaceOwner := fxt.Identities[0]
			otherIdentity := fxt.Identities[1]
			rootItr := fxt.Iterations[0]
			p := fxt.Spaces[0]

			svc, ctrl := s.SecuredControllerWithIdentity(spaceOwner)

			// try creating iteration with space-owner. should pass
			_, c := test.CreateSpaceIterationsCreated(s.T(), svc.Context, svc, ctrl, p.ID, ci)
			require.NotNil(s.T(), c.Data.ID)
			require.NotNil(s.T(), c.Data.Relationships.Space)
			assert.Equal(s.T(), p.ID.String(), *c.Data.Relationships.Space.Data.ID)
			assert.Equal(s.T(), iteration.StateNew.String(), *c.Data.Attributes.State)
			assert.Equal(s.T(), "/"+rootItr.ID.String(), *c.Data.Attributes.ParentPath)
			require.NotNil(s.T(), c.Data.Relationships.Workitems.Meta)
			assert.Equal(s.T(), 0, c.Data.Relationships.Workitems.Meta[KeyTotalWorkItems])
			assert.Equal(s.T(), 0, c.Data.Relationships.Workitems.Meta[KeyClosedWorkItems])

			svc, ctrl = s.SecuredControllerWithIdentity(otherIdentity)
			test.CreateSpaceIterationsForbidden(s.T(), svc.Context, svc, ctrl, p.ID, ci)
		})

	})

}

func (s *SpaceIterationControllerTestSuite) TestListIterationsBySpace() {
	s.T().Run("ok", func(t *testing.T) {
		t.Run("default", func(t *testing.T) {
			// given
			spaceID, fatherIteration, childIteration, grandChildIteration := s.createIterations()
			svc, ctrl := s.UnSecuredController()
			// when
			_, cs := test.ListSpaceIterationsOK(t, svc.Context, svc, ctrl, spaceID, nil, nil)
			// then
			assertIterations(t, cs.Data, fatherIteration, childIteration, grandChildIteration)
		})

		s.T().Run("using expired if-modified-since header", func(t *testing.T) {
			// given
			spaceID, fatherIteration, childIteration, grandChildIteration := s.createIterations()
			svc, ctrl := s.UnSecuredController()
			// when
			idModifiedSince := app.ToHTTPTime(fatherIteration.UpdatedAt.Add(-1 * time.Hour))
			_, cs := test.ListSpaceIterationsOK(t, svc.Context, svc, ctrl, spaceID, &idModifiedSince, nil)
			// then
			assertIterations(t, cs.Data, fatherIteration, childIteration, grandChildIteration)
		})

		s.T().Run("using expired if-none-match header", func(t *testing.T) {
			// given
			spaceID, fatherIteration, childIteration, grandChildIteration := s.createIterations()
			svc, ctrl := s.UnSecuredController()
			// when
			idNoneMatch := "foo"
			_, cs := test.ListSpaceIterationsOK(t, svc.Context, svc, ctrl, spaceID, nil, &idNoneMatch)
			// then
			assertIterations(t, cs.Data, fatherIteration, childIteration, grandChildIteration)
		})
	})

	s.T().Run("not modified", func(t *testing.T) {
		t.Run("not modified using expired if-modified-since header", func(t *testing.T) {
			// given
			spaceID, _, _, grandChildIteration := s.createIterations()
			svc, ctrl := s.UnSecuredController()
			// when/then
			idModifiedSince := app.ToHTTPTime(grandChildIteration.UpdatedAt)
			test.ListSpaceIterationsNotModified(t, svc.Context, svc, ctrl, spaceID, &idModifiedSince, nil)
		})

		t.Run("not modified using expired if-none-match header", func(t *testing.T) {
			// given
			spaceID, _, _, _ := s.createIterations()
			svc, ctrl := s.UnSecuredController()
			// here we need to get all iterations for the spaceId
			_, iterations := test.ListSpaceIterationsOK(t, svc.Context, svc, ctrl, spaceID, nil, nil)
			// when/then
			idNoneMatch := generateIterationsTag(*iterations)
			test.ListSpaceIterationsNotModified(t, svc.Context, svc, ctrl, spaceID, nil, &idNoneMatch)
		})
	})

	s.T().Run("fail", func(t *testing.T) {
		t.Run("missing space", func(t *testing.T) {
			// given
			svc, ctrl := s.UnSecuredController()
			// when/then
			test.ListSpaceIterationsNotFound(s.T(), svc.Context, svc, ctrl, uuid.NewV4(), nil, nil)
		})
	})

}

// Following is behaviour of the test that verifies the WI Count in an iteration
// Consider, iteration i1 has 2 children c1 & c2
// Total WI for i1 = WI assigned to i1 + WI assigned to c1 + WI assigned to c2
// Begin test with following setup :-
// Create a space s1
// create iteartion i1 & iteration i2 in s1
// Create child of i2 : name it child
// Create child of child : name it grandChild
// Add few "new" & "closed" work items to i1
// Add few "new" work items to child
// Add few "closed" work items to grandChild
// Call List-Iterations API, should return Total & Closed WI count for every itearion
// Verify counts for all 4 iterations retrieved.
// Add few "new" & "closed" work items to i2
// Call List-Iterations API, should return Total & Closed WI count for every itearion
// Verify updated count values for all 4 iterations retrieved.
func (s *SpaceIterationControllerTestSuite) TestWICountsWithIterationListBySpace() {
	// given
	resource.Require(s.T(), resource.Database)
	// create seed data
	spaceRepo := space.NewRepository(s.DB)
	spaceInstance := space.Space{
		Name:            testsupport.CreateRandomValidTestName("TestWICountsWithIterationListBySpace-"),
		SpaceTemplateID: spacetemplate.SystemLegacyTemplateID,
	}
	_, e := spaceRepo.Create(s.Ctx, &spaceInstance)
	require.Nil(s.T(), e)
	require.NotEqual(s.T(), uuid.UUID{}, spaceInstance.ID)

	iterationRepo := iteration.NewIterationRepository(s.DB)
	iteration1 := iteration.Iteration{
		Name:    "Sprint 1",
		SpaceID: spaceInstance.ID,
	}
	iterationRepo.Create(s.Ctx, &iteration1)
	assert.NotEqual(s.T(), uuid.UUID{}, iteration1.ID)

	iteration2 := iteration.Iteration{
		Name:    "Sprint 2",
		SpaceID: spaceInstance.ID,
	}
	iterationRepo.Create(s.Ctx, &iteration2)
	assert.NotEqual(s.T(), uuid.UUID{}, iteration2.ID)

	childOfIteration2 := iteration.Iteration{
		Name:    "Sprint 2.1",
		SpaceID: spaceInstance.ID,
		Path:    append(iteration2.Path, iteration2.ID),
	}
	iterationRepo.Create(s.Ctx, &childOfIteration2)
	require.NotEqual(s.T(), uuid.Nil, childOfIteration2.ID)

	grandChildOfIteration2 := iteration.Iteration{
		Name:    "Sprint 2.1.1",
		SpaceID: spaceInstance.ID,
		Path:    append(childOfIteration2.Path, childOfIteration2.ID),
	}
	iterationRepo.Create(s.Ctx, &grandChildOfIteration2)
	require.NotEqual(s.T(), uuid.UUID{}, grandChildOfIteration2.ID)

	wirepo := workitem.NewWorkItemRepository(s.DB)

	for i := 0; i < 3; i++ {
		wirepo.Create(
			s.Ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: iteration1.ID.String(),
			}, s.testIdentity.ID)
	}
	for i := 0; i < 2; i++ {
		_, err := wirepo.Create(
			s.Ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("Closed issue #%d", i),
				workitem.SystemState:     workitem.SystemStateClosed,
				workitem.SystemIteration: iteration1.ID.String(),
			}, s.testIdentity.ID)
		require.NoError(s.T(), err)
	}
	// add items to nested iteration level 1
	for i := 0; i < 4; i++ {
		_, err := wirepo.Create(
			s.Ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: childOfIteration2.ID.String(),
			}, s.testIdentity.ID)
		require.NoError(s.T(), err)
	}
	// add items to nested iteration level 2
	for i := 0; i < 5; i++ {
		_, err := wirepo.Create(
			s.Ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("Closed issue #%d", i),
				workitem.SystemState:     workitem.SystemStateClosed,
				workitem.SystemIteration: grandChildOfIteration2.ID.String(),
			}, s.testIdentity.ID)
		require.NoError(s.T(), err)
	}

	svc, ctrl := s.UnSecuredController()
	// when
	_, cs := test.ListSpaceIterationsOK(s.T(), svc.Context, svc, ctrl, spaceInstance.ID, nil, nil)
	// then
	require.Len(s.T(), cs.Data, 4)
	for _, iterationItem := range cs.Data {
		if uuid.Equal(*iterationItem.ID, iteration1.ID) {
			assert.Equal(s.T(), 5, iterationItem.Relationships.Workitems.Meta[KeyTotalWorkItems])
			assert.Equal(s.T(), 2, iterationItem.Relationships.Workitems.Meta[KeyClosedWorkItems])
		} else if uuid.Equal(*iterationItem.ID, iteration2.ID) {
			// we expect these counts should include that of child iterations too.
			expectedTotal := 0 + 4 + 5  // sum of all items of self + child + grand-child
			expectedClosed := 0 + 0 + 5 // sum of closed items self + child + grand-child
			assert.Equal(s.T(), expectedTotal, iterationItem.Relationships.Workitems.Meta[KeyTotalWorkItems])
			assert.Equal(s.T(), expectedClosed, iterationItem.Relationships.Workitems.Meta[KeyClosedWorkItems])
		} else if uuid.Equal(*iterationItem.ID, childOfIteration2.ID) {
			// we expect these counts should include that of child iterations too.
			expectedTotal := 4 + 5  // sum of all items of self and child
			expectedClosed := 0 + 5 // sum of closed items of self and child
			assert.Equal(s.T(), expectedTotal, iterationItem.Relationships.Workitems.Meta[KeyTotalWorkItems])
			assert.Equal(s.T(), expectedClosed, iterationItem.Relationships.Workitems.Meta[KeyClosedWorkItems])
		} else if uuid.Equal(*iterationItem.ID, grandChildOfIteration2.ID) {
			// we expect these counts should include that of child iterations too.
			expectedTotal := 5 + 0  // sum of all items of self and child
			expectedClosed := 5 + 0 // sum of closed items of self and child
			assert.Equal(s.T(), expectedTotal, iterationItem.Relationships.Workitems.Meta[KeyTotalWorkItems])
			assert.Equal(s.T(), expectedClosed, iterationItem.Relationships.Workitems.Meta[KeyClosedWorkItems])
		}
	}
	// seed 5 New WI to iteration2
	for i := 0; i < 5; i++ {
		_, err := wirepo.Create(
			s.Ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: iteration2.ID.String(),
			}, s.testIdentity.ID)
		require.NoError(s.T(), err)
	}
	// seed 2 Closed WI to iteration2
	for i := 0; i < 3; i++ {
		_, err := wirepo.Create(
			s.Ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("Closed issue #%d", i),
				workitem.SystemState:     workitem.SystemStateClosed,
				workitem.SystemIteration: iteration2.ID.String(),
			}, s.testIdentity.ID)
		require.NoError(s.T(), err)
	}
	// when
	_, cs = test.ListSpaceIterationsOK(s.T(), svc.Context, svc, ctrl, spaceInstance.ID, nil, nil)
	// then
	require.Len(s.T(), cs.Data, 4)
	for _, iterationItem := range cs.Data {
		if uuid.Equal(*iterationItem.ID, iteration1.ID) {
			assert.Equal(s.T(), 5, iterationItem.Relationships.Workitems.Meta[KeyTotalWorkItems])
			assert.Equal(s.T(), 2, iterationItem.Relationships.Workitems.Meta[KeyClosedWorkItems])
		} else if uuid.Equal(*iterationItem.ID, iteration2.ID) {
			// we expect these counts should include that of child iterations too.
			expectedTotal := 8 + 4 + 5  // sum of all items of self + child + grand-child
			expectedClosed := 3 + 0 + 5 // sum of closed items self + child + grand-child
			assert.Equal(s.T(), expectedTotal, iterationItem.Relationships.Workitems.Meta[KeyTotalWorkItems])
			assert.Equal(s.T(), expectedClosed, iterationItem.Relationships.Workitems.Meta[KeyClosedWorkItems])
		} else if uuid.Equal(*iterationItem.ID, childOfIteration2.ID) {
			// we expect these counts should include that of child iterations too.
			expectedTotal := 4 + 5  // sum of all items of self + child + grand-child
			expectedClosed := 0 + 5 // sum of closed items self + child + grand-child
			assert.Equal(s.T(), expectedTotal, iterationItem.Relationships.Workitems.Meta[KeyTotalWorkItems])
			assert.Equal(s.T(), expectedClosed, iterationItem.Relationships.Workitems.Meta[KeyClosedWorkItems])
		} else if uuid.Equal(*iterationItem.ID, grandChildOfIteration2.ID) {
			// we expect these counts should include that of child iterations too.
			expectedTotal := 5 + 0  // sum of all items of self + child + grand-child
			expectedClosed := 5 + 0 // sum of closed items self + child + grand-child
			assert.Equal(s.T(), expectedTotal, iterationItem.Relationships.Workitems.Meta[KeyTotalWorkItems])
			assert.Equal(s.T(), expectedClosed, iterationItem.Relationships.Workitems.Meta[KeyClosedWorkItems])
		}
	}
}

func newCreateSpaceIterationPayload(name string, desc *string) *app.CreateSpaceIterationsPayload {
	start := time.Now()
	end := start.Add(time.Hour * (24 * 8 * 3))

	return &app.CreateSpaceIterationsPayload{
		Data: &app.Iteration{
			Type: iteration.APIStringTypeIteration,
			Attributes: &app.IterationAttributes{
				Name:        &name,
				StartAt:     &start,
				EndAt:       &end,
				Description: desc,
			},
		},
	}
}

func (s *SpaceIterationControllerTestSuite) createIterations() (spaceID uuid.UUID, fatherIteration, childIteration, grandChildIteration *iteration.Iteration) {
	err := application.Transactional(s.GormDB, func(app application.Application) error {
		repo := app.Iterations()
		newSpace := space.Space{
			Name:            testsupport.CreateRandomValidTestName("TestListIterationsBySpace-"),
			SpaceTemplateID: spacetemplate.SystemLegacyTemplateID,
		}
		p, err := app.Spaces().Create(s.Ctx, &newSpace)
		if err != nil {
			return err
		}
		spaceID = p.ID
		for i := 0; i < 3; i++ {
			start := time.Now()
			end := start.Add(time.Hour * (24 * 8 * 3))
			name := "Sprint Test #" + strconv.Itoa(i)
			i := iteration.Iteration{
				Name:    name,
				SpaceID: spaceID,
				StartAt: &start,
				EndAt:   &end,
			}
			repo.Create(s.Ctx, &i)
		}
		// create one child iteration and test for relationships.Parent
		fatherIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: spaceID,
		}
		repo.Create(s.Ctx, fatherIteration)
		s.T().Log("fatherIteration:", fatherIteration.ID, fatherIteration.Name, fatherIteration.Path)
		childIteration = &iteration.Iteration{
			Name:    "Child Iteration",
			SpaceID: spaceID,
			Path:    append(fatherIteration.Path, fatherIteration.ID),
		}
		repo.Create(s.Ctx, childIteration)
		s.T().Log("childIteration:", childIteration.ID, childIteration.Name, childIteration.Path)
		grandChildIteration = &iteration.Iteration{
			Name:    "Grand Child Iteration",
			SpaceID: spaceID,
			Path:    append(childIteration.Path, childIteration.ID),
		}
		repo.Create(s.Ctx, grandChildIteration)
		s.T().Log("grandChildIteration:", grandChildIteration.ID, grandChildIteration.Name, grandChildIteration.Path)

		return nil
	})
	require.NoError(s.T(), err)
	return
}

func assertIterations(t *testing.T, data []*app.Iteration, fatherIteration, childIteration, grandChildIteration *iteration.Iteration) {
	assert.Len(t, data, 6)
	for _, iterationItem := range data {
		subString := fmt.Sprintf("?filter[iteration]=%s", iterationItem.ID.String())
		require.Contains(t, *iterationItem.Relationships.Workitems.Links.Related, subString)
		assert.Equal(t, 0, iterationItem.Relationships.Workitems.Meta[KeyTotalWorkItems])
		assert.Equal(t, 0, iterationItem.Relationships.Workitems.Meta[KeyClosedWorkItems])
		if *iterationItem.ID == childIteration.ID {
			t.Log("childIteration:", iterationItem.ID, *iterationItem.Attributes.Name, *iterationItem.Attributes.ParentPath, *iterationItem.Relationships.Parent.Data.ID)
			expectedParentPath := iteration.PathSepInService + fatherIteration.ID.String()
			expectedResolvedParentPath := iteration.PathSepInService + fatherIteration.Name
			require.NotNil(t, iterationItem.Relationships.Parent)
			assert.Equal(t, fatherIteration.ID.String(), *iterationItem.Relationships.Parent.Data.ID)
			assert.Equal(t, expectedParentPath, *iterationItem.Attributes.ParentPath)
			assert.Equal(t, expectedResolvedParentPath, *iterationItem.Attributes.ResolvedParentPath)
		}
		if *iterationItem.ID == grandChildIteration.ID {
			t.Log("grandChildIteration:", iterationItem.ID, *iterationItem.Attributes.Name, *iterationItem.Attributes.ParentPath, *iterationItem.Relationships.Parent.Data.ID)
			expectedParentPath := iteration.PathSepInService + fatherIteration.ID.String() + iteration.PathSepInService + childIteration.ID.String()
			expectedResolvedParentPath := iteration.PathSepInService + fatherIteration.Name + iteration.PathSepInService + childIteration.Name
			require.NotNil(t, iterationItem.Relationships.Parent)
			assert.Equal(t, childIteration.ID.String(), *iterationItem.Relationships.Parent.Data.ID)
			assert.Equal(t, expectedParentPath, *iterationItem.Attributes.ParentPath)
			assert.Equal(t, expectedResolvedParentPath, *iterationItem.Attributes.ResolvedParentPath)

		}
	}
}

func generateIterationsTag(iterations app.IterationList) string {
	modelEntities := make([]app.ConditionalRequestEntity, len(iterations.Data))
	for i, entity := range iterations.Data {
		modelEntities[i] = iteration.Iteration{
			ID: *entity.ID,
			Lifecycle: gormsupport.Lifecycle{
				UpdatedAt: *entity.Attributes.UpdatedAt,
			},
		}
	}
	return app.GenerateEntitiesTag(modelEntities)
}
