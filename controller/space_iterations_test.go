package controller_test

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"context"

	"fmt"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestSpaceIterationREST struct {
	gormtestsupport.DBTestSuite
	db           *gormapplication.GormDB
	testIdentity account.Identity
}

func TestRunSpaceIterationREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestSpaceIterationREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestSpaceIterationREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
	rest.db = gormapplication.NewGormDB(rest.DB)
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestSpaceIterationREST user", "test provider")
	require.Nil(rest.T(), err)
	rest.testIdentity = *testIdentity
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	rest.Ctx = goa.NewContext(context.Background(), nil, req, params)
}

func (rest *TestSpaceIterationREST) SecuredController() (*goa.Service, *SpaceIterationsController) {
	svc := testsupport.ServiceAsUser("Iteration-Service", testsupport.TestIdentity)
	return svc, NewSpaceIterationsController(svc, rest.db, rest.Configuration)
}

func (rest *TestSpaceIterationREST) SecuredControllerWithIdentity(idn *account.Identity) (*goa.Service, *SpaceIterationsController) {
	svc := testsupport.ServiceAsUser("Iteration-Service", *idn)
	return svc, NewSpaceIterationsController(svc, rest.db, rest.Configuration)
}

func (rest *TestSpaceIterationREST) UnSecuredController() (*goa.Service, *SpaceIterationsController) {
	svc := goa.New("Iteration-Service")
	return svc, NewSpaceIterationsController(svc, rest.db, rest.Configuration)
}

func (rest *TestSpaceIterationREST) TestSuccessCreateIteration() {
	// given
	var p *space.Space
	var rootItr *iteration.Iteration
	ci := createSpaceIteration("Sprint #21", nil)
	err := application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Spaces()
		newSpace := space.Space{
			Name:    "TestSuccessCreateIteration" + uuid.NewV4().String(),
			OwnerID: testsupport.TestIdentity.ID,
		}
		createdSpace, err := repo.Create(rest.Ctx, &newSpace)
		p = createdSpace
		if err != nil {
			return err
		}
		// create Root iteration for above space
		rootItr = &iteration.Iteration{
			SpaceID: newSpace.ID,
			Name:    newSpace.Name,
		}
		iterationRepo := app.Iterations()
		err = iterationRepo.Create(rest.Ctx, rootItr)
		return err
	})
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredController()
	// when
	_, c := test.CreateSpaceIterationsCreated(rest.T(), svc.Context, svc, ctrl, p.ID, ci)
	// then
	require.NotNil(rest.T(), c.Data.ID)
	require.NotNil(rest.T(), c.Data.Relationships.Space)
	assert.Equal(rest.T(), p.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(rest.T(), iteration.StateNew.String(), *c.Data.Attributes.State)
	assert.Equal(rest.T(), "/"+rootItr.ID.String(), *c.Data.Attributes.ParentPath)
	require.NotNil(rest.T(), c.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, c.Data.Relationships.Workitems.Meta[KeyTotalWorkItems])
	assert.Equal(rest.T(), 0, c.Data.Relationships.Workitems.Meta[KeyClosedWorkItems])
}

func (rest *TestSpaceIterationREST) TestSuccessCreateIterationWithOptionalValues() {
	// given
	var p *space.Space
	var rootItr *iteration.Iteration
	iterationName := "Sprint #22"
	iterationDesc := "testing description"
	ci := createSpaceIteration(iterationName, &iterationDesc)
	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Spaces()
		testSpace := space.Space{
			Name:    "TestSuccessCreateIterationWithOptionalValues-" + uuid.NewV4().String(),
			OwnerID: testsupport.TestIdentity.ID,
		}
		p, _ = repo.Create(rest.Ctx, &testSpace)
		// create Root iteration for above space
		rootItr = &iteration.Iteration{
			SpaceID: testSpace.ID,
			Name:    testSpace.Name,
		}
		iterationRepo := app.Iterations()
		err := iterationRepo.Create(rest.Ctx, rootItr)
		require.Nil(rest.T(), err)
		return nil
	})
	svc, ctrl := rest.SecuredController()
	// when
	_, c := test.CreateSpaceIterationsCreated(rest.T(), svc.Context, svc, ctrl, p.ID, ci)
	// then
	assert.NotNil(rest.T(), c.Data.ID)
	assert.NotNil(rest.T(), c.Data.Relationships.Space)
	assert.Equal(rest.T(), p.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(rest.T(), *c.Data.Attributes.Name, iterationName)
	assert.Equal(rest.T(), *c.Data.Attributes.Description, iterationDesc)

	// create another Iteration with nil description
	iterationName2 := "Sprint #23"
	ci = createSpaceIteration(iterationName2, nil)
	_, c = test.CreateSpaceIterationsCreated(rest.T(), svc.Context, svc, ctrl, p.ID, ci)
	assert.Equal(rest.T(), *c.Data.Attributes.Name, iterationName2)
	assert.Nil(rest.T(), c.Data.Attributes.Description)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceOK() {
	// given
	spaceID, fatherIteration, childIteration, grandChildIteration := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// when
	_, cs := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceID, nil, nil)
	// then
	assertIterations(rest.T(), cs.Data, fatherIteration, childIteration, grandChildIteration)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceOKUsingExpiredIfModifiedSinceHeader() {
	// given
	spaceID, fatherIteration, childIteration, grandChildIteration := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// when
	idModifiedSince := app.ToHTTPTime(fatherIteration.UpdatedAt.Add(-1 * time.Hour))
	_, cs := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceID, &idModifiedSince, nil)
	// then
	assertIterations(rest.T(), cs.Data, fatherIteration, childIteration, grandChildIteration)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceOKUsingExpiredIfNoneMatchSinceHeader() {
	// given
	spaceID, fatherIteration, childIteration, grandChildIteration := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// when
	idNoneMatch := "foo"
	_, cs := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceID, nil, &idNoneMatch)
	// then
	assertIterations(rest.T(), cs.Data, fatherIteration, childIteration, grandChildIteration)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceNotModifiedUsingIfModifiedSinceHeader() {
	// given
	spaceID, _, _, grandChildIteration := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// when/then
	idModifiedSince := app.ToHTTPTime(grandChildIteration.UpdatedAt)
	test.ListSpaceIterationsNotModified(rest.T(), svc.Context, svc, ctrl, spaceID, &idModifiedSince, nil)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceNotModifiedUsingIfNoneMatchSinceHeader() {
	// given
	spaceID, _, _, _ := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// here we need to get all iterations for the spaceId
	_, iterations := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceID, nil, nil)
	// when/then
	idNoneMatch := generateIterationsTag(*iterations)
	test.ListSpaceIterationsNotModified(rest.T(), svc.Context, svc, ctrl, spaceID, nil, &idNoneMatch)
}

func (rest *TestSpaceIterationREST) TestCreateIterationMissingSpace() {
	// given
	ci := createSpaceIteration("Sprint #21", nil)
	svc, ctrl := rest.SecuredController()
	// when/then
	test.CreateSpaceIterationsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4(), ci)
}

func (rest *TestSpaceIterationREST) TestFailCreateIterationNotAuthorized() {
	// given
	ci := createSpaceIteration("Sprint #21", nil)
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.CreateSpaceIterationsUnauthorized(rest.T(), svc.Context, svc, ctrl, uuid.NewV4(), ci)
}

func (rest *TestSpaceIterationREST) TestFailListIterationsByMissingSpace() {
	// given
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.ListSpaceIterationsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4(), nil, nil)
}

// Following is behaviour of the test that verifies the WI Count in an iteration
// Consider this iteration structure with workitems assigend to them:
//
//     + Root
//     |  + Iteration: "Sprint 1" (10 direct children)
//     |  |  * New Feature
//     |  |  * New Feature
//     |  |  * New Feature
//     |  |  * Closed Feature
//     |  |  * Closed Feature
//     |  |  * New Task
//     |  |  * New Task
//     |  |  * New Task
//     |  |  * Closed Task
//     |  |  * Closed Task
//     |  + Iteration: "Sprint 2" (no direct children)
//     |  |  + Iteration: "Sprint 2.1" (8 direct children)
//     |  |  |  * New Feature
//     |  |  |  * New Feature
//     |  |  |  * New Feature
//     |  |  |  * New Feature
//     |  |  |  * New Task
//     |  |  |  * New Task
//     |  |  |  * New Task
//     |  |  |  * New Task
//     |  |  + Iteration: "Sprint 2.1.1"(10 direct children)
//     |  |  |  * Closed Feature
//     |  |  |  * Closed Feature
//     |  |  |  * Closed Feature
//     |  |  |  * Closed Feature
//     |  |  |  * Closed Feature
//     |  |  |  * Closed Task
//     |  |  |  * Closed Task
//     |  |  |  * Closed Task
//     |  |  |  * Closed Task
//     |  |  |  * Closed Task
//
// Call List-Iterations API, should return Total & Closed WI count for every iteration
// Verify counts for all 4 iterations retrieved.
// Add few "new" & "closed" work items to i2
// Call List-Iterations API, should return Total & Closed WI count for every iteration
// Verify updated count values for all 4 iterations retrieved.
func (rest *TestSpaceIterationREST) TestWICountsWithIterationListBySpace() {
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB,
		tf.CreateWorkItemEnvironment(),
		tf.Iterations(5,
			tf.SetIterationNames("Root", "Sprint 1", "Sprint 2", "Sprint 2.1", "Sprint 2.1.1"),
			func(fxt *tf.TestFixture, idx int) error {
				if idx == 3 || idx == 4 {
					fxt.Iterations[idx].MakeChildOf(*fxt.Iterations[idx-1])
				}
				return nil
			}),
		tf.WorkItems(10+0+8+10, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			switch idx {
			case 0, 1, 2:
				wi.Fields[workitem.SystemState] = workitem.SystemStateNew
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("Sprint 1").ID.String()
				wi.Type = workitem.SystemFeature
			case 3, 4, 5:
				wi.Fields[workitem.SystemState] = workitem.SystemStateNew
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("Sprint 1").ID.String()
				wi.Type = workitem.SystemTask
			case 6, 7:
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("Sprint 1").ID.String()
				wi.Type = workitem.SystemFeature
			case 8, 9:
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("Sprint 1").ID.String()
				wi.Type = workitem.SystemTask
			case 10, 11, 12, 13:
				wi.Fields[workitem.SystemState] = workitem.SystemStateNew
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("Sprint 2.1").ID.String()
				wi.Type = workitem.SystemFeature
			case 14, 15, 16, 17:
				wi.Fields[workitem.SystemState] = workitem.SystemStateNew
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("Sprint 2.1").ID.String()
				wi.Type = workitem.SystemTask
			case 18, 19, 20, 21, 22:
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("Sprint 2.1.1").ID.String()
				wi.Type = workitem.SystemFeature
			case 23, 24, 25, 26, 27:
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("Sprint 2.1.1").ID.String()
				wi.Type = workitem.SystemTask
			}
			return nil
		}),
	)

	svc, ctrl := rest.UnSecuredController()
	// when
	_, cs := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, fxt.Spaces[0].ID, nil, nil)
	// then
	require.Len(rest.T(), cs.Data, len(fxt.Iterations))
	expectedTotalCounts := map[string]int{
		"Root":         0 + 5 + 0 + 4 + 5,
		"Sprint 1":     5,
		"Sprint 2":     0 + 4 + 5,
		"Sprint 2.1":   4 + 5,
		"Sprint 2.1.1": 5,
	}
	expectedClosedCounts := map[string]int{
		"Root":         0 + 2 + 0 + 0 + 5,
		"Sprint 1":     2,
		"Sprint 2":     0 + 0 + 5,
		"Sprint 2.1":   0 + 5,
		"Sprint 2.1.1": 5,
	}
	rest.T().Run("iteration", func(t *testing.T) {
		for _, iterationItem := range cs.Data {
			iterName := *iterationItem.Attributes.Name
			t.Run(iterName, func(t *testing.T) {
				t.Run("total count", func(t *testing.T) {
					expectedTotalCount, ok := expectedTotalCounts[iterName]
					require.True(t, ok, "failed to find iteration %s in %+v", iterName, expectedTotalCounts)
					require.Equal(t, expectedTotalCount, iterationItem.Relationships.Workitems.Meta[KeyTotalWorkItems])
				})
				t.Run("closed count", func(t *testing.T) {
					expectedClosedCount, ok := expectedClosedCounts[iterName]
					require.True(t, ok, "failed to find iteration %s in %+v", iterName, expectedClosedCounts)
					require.Equal(t, expectedClosedCount, iterationItem.Relationships.Workitems.Meta[KeyClosedWorkItems])
				})
			})
		}
	})

	tf.NewTestFixture(rest.T(), rest.DB,
		tf.CreateWorkItemEnvironment(),
		tf.WorkItems(16, func(fxt2 *tf.TestFixture, idx int) error {
			wi := fxt2.WorkItems[idx]
			switch idx {
			case 0, 1, 2, 3, 4:
				wi.Fields[workitem.SystemState] = workitem.SystemStateNew
				wi.Fields[workitem.SystemIteration] = fxt.Iterations[2].ID.String()
				wi.Type = workitem.SystemFeature
			case 5, 6, 7, 8, 9:
				wi.Fields[workitem.SystemState] = workitem.SystemStateNew
				wi.Fields[workitem.SystemIteration] = fxt.Iterations[2].ID.String()
				wi.Type = workitem.SystemTask
			case 10, 11, 12:
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
				wi.Fields[workitem.SystemIteration] = fxt.Iterations[2].ID.String()
				wi.Type = workitem.SystemFeature
			case 13, 14, 15:
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
				wi.Fields[workitem.SystemIteration] = fxt.Iterations[2].ID.String()
				wi.Type = workitem.SystemTask
			}
			return nil
		}),
	)

	// when
	_, cs = test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, fxt.Spaces[0].ID, nil, nil)
	// then
	require.Len(rest.T(), cs.Data, len(fxt.Iterations))
	expectedTotalCounts = map[string]int{
		"Root":         0 + 5 + 8 + 4 + 5,
		"Sprint 1":     5,
		"Sprint 2":     8 + 4 + 5,
		"Sprint 2.1":   4 + 5,
		"Sprint 2.1.1": 5,
	}
	expectedClosedCounts = map[string]int{
		"Root":         0 + 2 + 3 + 0 + 5,
		"Sprint 1":     2,
		"Sprint 2":     3 + 0 + 5,
		"Sprint 2.1":   0 + 5,
		"Sprint 2.1.1": 5,
	}
	rest.T().Run("iteration", func(t *testing.T) {
		for _, iterationItem := range cs.Data {
			iterName := *iterationItem.Attributes.Name
			t.Run(iterName, func(t *testing.T) {
				t.Run("total count", func(t *testing.T) {
					expectedTotalCount, ok := expectedTotalCounts[iterName]
					require.True(t, ok, "failed to find iteration %s in %+v", iterName, expectedTotalCounts)
					require.Equal(t, expectedTotalCount, iterationItem.Relationships.Workitems.Meta[KeyTotalWorkItems])
				})
				t.Run("closed count", func(t *testing.T) {
					expectedClosedCount, ok := expectedClosedCounts[iterName]
					require.True(t, ok, "failed to find iteration %s in %+v", iterName, expectedClosedCounts)
					require.Equal(t, expectedClosedCount, iterationItem.Relationships.Workitems.Meta[KeyClosedWorkItems])
				})
			})
		}
	})
}

func (rest *TestSpaceIterationREST) TestOnlySpaceOwnerCreateIteration() {
	var p *space.Space
	var rootItr *iteration.Iteration
	identityRepo := account.NewIdentityRepository(rest.DB)
	spaceOwner := &account.Identity{
		ID:           uuid.NewV4(),
		Username:     "space-owner-identity",
		ProviderType: account.KeycloakIDP}
	errInCreateOwner := identityRepo.Create(rest.Ctx, spaceOwner)
	require.Nil(rest.T(), errInCreateOwner)

	ci := createSpaceIteration("Sprint #21", nil)
	err := application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Spaces()
		newSpace := space.Space{
			Name:    "TestSuccessCreateIteration" + uuid.NewV4().String(),
			OwnerID: spaceOwner.ID,
		}
		createdSpace, err := repo.Create(rest.Ctx, &newSpace)
		p = createdSpace
		if err != nil {
			return err
		}
		// create Root iteration for above space
		rootItr = &iteration.Iteration{
			SpaceID: newSpace.ID,
			Name:    newSpace.Name,
		}
		iterationRepo := app.Iterations()
		err = iterationRepo.Create(rest.Ctx, rootItr)
		return err
	})
	require.Nil(rest.T(), err)

	spaceOwner, errInLoad := identityRepo.Load(rest.Ctx, p.OwnerID)
	require.Nil(rest.T(), errInLoad)

	svc, ctrl := rest.SecuredControllerWithIdentity(spaceOwner)

	// try creating iteration with space-owner. should pass
	_, c := test.CreateSpaceIterationsCreated(rest.T(), svc.Context, svc, ctrl, p.ID, ci)
	require.NotNil(rest.T(), c.Data.ID)
	require.NotNil(rest.T(), c.Data.Relationships.Space)
	assert.Equal(rest.T(), p.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(rest.T(), iteration.StateNew.String(), *c.Data.Attributes.State)
	assert.Equal(rest.T(), "/"+rootItr.ID.String(), *c.Data.Attributes.ParentPath)
	require.NotNil(rest.T(), c.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, c.Data.Relationships.Workitems.Meta[KeyTotalWorkItems])
	assert.Equal(rest.T(), 0, c.Data.Relationships.Workitems.Meta[KeyClosedWorkItems])

	otherIdentity := &account.Identity{
		ID:           uuid.NewV4(),
		Username:     "non-space-owner-identity",
		ProviderType: account.KeycloakIDP}
	errInCreateOther := identityRepo.Create(rest.Ctx, otherIdentity)
	require.Nil(rest.T(), errInCreateOther)

	svc, ctrl = rest.SecuredControllerWithIdentity(otherIdentity)
	test.CreateSpaceIterationsForbidden(rest.T(), svc.Context, svc, ctrl, p.ID, ci)
}

func createSpaceIteration(name string, desc *string) *app.CreateSpaceIterationsPayload {
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

func (rest *TestSpaceIterationREST) createIterations() (spaceID uuid.UUID, fatherIteration, childIteration, grandChildIteration *iteration.Iteration) {
	err := application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Iterations()
		newSpace := space.Space{
			Name: "TestListIterationsBySpace-" + uuid.NewV4().String(),
		}
		p, err := app.Spaces().Create(rest.Ctx, &newSpace)
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
			repo.Create(rest.Ctx, &i)
		}
		// create one child iteration and test for relationships.Parent
		fatherIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: spaceID,
		}
		repo.Create(rest.Ctx, fatherIteration)
		rest.T().Log("fatherIteration:", fatherIteration.ID, fatherIteration.Name, fatherIteration.Path)
		childIteration = &iteration.Iteration{
			Name:    "Child Iteration",
			SpaceID: spaceID,
			Path:    append(fatherIteration.Path, fatherIteration.ID),
		}
		repo.Create(rest.Ctx, childIteration)
		rest.T().Log("childIteration:", childIteration.ID, childIteration.Name, childIteration.Path)
		grandChildIteration = &iteration.Iteration{
			Name:    "Grand Child Iteration",
			SpaceID: spaceID,
			Path:    append(childIteration.Path, childIteration.ID),
		}
		repo.Create(rest.Ctx, grandChildIteration)
		rest.T().Log("grandChildIteration:", grandChildIteration.ID, grandChildIteration.Name, grandChildIteration.Path)

		return nil
	})
	require.Nil(rest.T(), err)
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
