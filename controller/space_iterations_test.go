package controller_test

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"golang.org/x/net/context"

	"fmt"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestSpaceIterationREST struct {
	gormtestsupport.DBTestSuite
	db           *gormapplication.GormDB
	clean        func()
	ctx          context.Context
	testIdentity account.Identity
}

func TestRunSpaceIterationREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestSpaceIterationREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (rest *TestSpaceIterationREST) SetupSuite() {
	rest.DBTestSuite.SetupSuite()
	rest.ctx = migration.NewMigrationContext(context.Background())
	rest.DBTestSuite.PopulateDBTestSuite(rest.ctx)
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestSpaceIterationREST user", "test provider")
	require.Nil(rest.T(), err)
	rest.testIdentity = testIdentity
}

func (rest *TestSpaceIterationREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	rest.ctx = goa.NewContext(context.Background(), nil, req, params)
}

func (rest *TestSpaceIterationREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSpaceIterationREST) SecuredController() (*goa.Service, *SpaceIterationsController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Iteration-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
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
			Name: "TestSuccessCreateIteration" + uuid.NewV4().String(),
		}
		createdSpace, err := repo.Create(rest.ctx, &newSpace)
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
		err = iterationRepo.Create(rest.ctx, rootItr)
		return err
	})
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredController()
	// when
	_, c := test.CreateSpaceIterationsCreated(rest.T(), svc.Context, svc, ctrl, p.ID.String(), ci)
	// then
	require.NotNil(rest.T(), c.Data.ID)
	require.NotNil(rest.T(), c.Data.Relationships.Space)
	assert.Equal(rest.T(), p.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(rest.T(), iteration.IterationStateNew, *c.Data.Attributes.State)
	assert.Equal(rest.T(), "/"+rootItr.ID.String(), *c.Data.Attributes.ParentPath)
	require.NotNil(rest.T(), c.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, c.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, c.Data.Relationships.Workitems.Meta["closed"])
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
			Name: "TestSuccessCreateIterationWithOptionalValues-" + uuid.NewV4().String(),
		}
		p, _ = repo.Create(rest.ctx, &testSpace)
		// create Root iteration for above space
		rootItr = &iteration.Iteration{
			SpaceID: testSpace.ID,
			Name:    testSpace.Name,
		}
		iterationRepo := app.Iterations()
		err := iterationRepo.Create(rest.ctx, rootItr)
		require.Nil(rest.T(), err)
		return nil
	})
	svc, ctrl := rest.SecuredController()
	// when
	_, c := test.CreateSpaceIterationsCreated(rest.T(), svc.Context, svc, ctrl, p.ID.String(), ci)
	// then
	assert.NotNil(rest.T(), c.Data.ID)
	assert.NotNil(rest.T(), c.Data.Relationships.Space)
	assert.Equal(rest.T(), p.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(rest.T(), *c.Data.Attributes.Name, iterationName)
	assert.Equal(rest.T(), *c.Data.Attributes.Description, iterationDesc)

	// create another Iteration with nil description
	iterationName2 := "Sprint #23"
	ci = createSpaceIteration(iterationName2, nil)
	_, c = test.CreateSpaceIterationsCreated(rest.T(), svc.Context, svc, ctrl, p.ID.String(), ci)
	assert.Equal(rest.T(), *c.Data.Attributes.Name, iterationName2)
	assert.Nil(rest.T(), c.Data.Attributes.Description)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceOK() {
	// given
	spaceID, fatherIteration, childIteration, grandChildIteration := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// when
	_, cs := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceID.String(), nil, nil)
	// then
	assertIterations(rest.T(), cs.Data, fatherIteration, childIteration, grandChildIteration)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceOKUsingExpiredIfModifiedSinceHeader() {
	// given
	spaceID, fatherIteration, childIteration, grandChildIteration := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// when
	idModifiedSince := app.ToHTTPTime(fatherIteration.UpdatedAt.Add(-1 * time.Hour))
	_, cs := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceID.String(), &idModifiedSince, nil)
	// then
	assertIterations(rest.T(), cs.Data, fatherIteration, childIteration, grandChildIteration)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceOKUsingExpiredIfNoneMatchSinceHeader() {
	// given
	spaceID, fatherIteration, childIteration, grandChildIteration := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// when
	idNoneMatch := "foo"
	_, cs := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceID.String(), nil, &idNoneMatch)
	// then
	assertIterations(rest.T(), cs.Data, fatherIteration, childIteration, grandChildIteration)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceNotModifiedUsingIfModifiedSinceHeader() {
	// given
	spaceID, _, _, grandChildIteration := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// when/then
	idModifiedSince := app.ToHTTPTime(grandChildIteration.UpdatedAt)
	test.ListSpaceIterationsNotModified(rest.T(), svc.Context, svc, ctrl, spaceID.String(), &idModifiedSince, nil)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceNotModifiedUsingIfNoneMatchSinceHeader() {
	// given
	spaceID, _, _, _ := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// here we need to get all iterations for the spaceId
	_, iterations := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceID.String(), nil, nil)
	// when/then
	idNoneMatch := generateIterationsTag(*iterations)
	test.ListSpaceIterationsNotModified(rest.T(), svc.Context, svc, ctrl, spaceID.String(), nil, &idNoneMatch)
}

func (rest *TestSpaceIterationREST) TestCreateIterationMissingSpace() {
	// given
	ci := createSpaceIteration("Sprint #21", nil)
	svc, ctrl := rest.SecuredController()
	// when/then
	test.CreateSpaceIterationsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), ci)
}

func (rest *TestSpaceIterationREST) TestFailCreateIterationNotAuthorized() {
	// given
	ci := createSpaceIteration("Sprint #21", nil)
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.CreateSpaceIterationsUnauthorized(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), ci)
}

func (rest *TestSpaceIterationREST) TestFailListIterationsByMissingSpace() {
	// given
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.ListSpaceIterationsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), nil, nil)
}

func (rest *TestSpaceIterationREST) TestWICountsWithIterationListBySpace() {
	// given
	resource.Require(rest.T(), resource.Database)
	// create seed data
	spaceRepo := space.NewRepository(rest.DB)
	spaceInstance := space.Space{
		Name: "TestWICountsWithIterationListBySpace-" + uuid.NewV4().String(),
	}
	_, e := spaceRepo.Create(rest.ctx, &spaceInstance)
	require.Nil(rest.T(), e)
	fmt.Println("space id = ", spaceInstance.ID)
	require.NotEqual(rest.T(), uuid.UUID{}, spaceInstance.ID)

	iterationRepo := iteration.NewIterationRepository(rest.DB)
	iteration1 := iteration.Iteration{
		Name:    "Sprint 1",
		SpaceID: spaceInstance.ID,
	}
	iterationRepo.Create(rest.ctx, &iteration1)
	fmt.Println("iteration1 id = ", iteration1.ID)
	assert.NotEqual(rest.T(), uuid.UUID{}, iteration1.ID)

	iteration2 := iteration.Iteration{
		Name:    "Sprint 2",
		SpaceID: spaceInstance.ID,
	}
	iterationRepo.Create(rest.ctx, &iteration2)
	fmt.Println("iteration2 id = ", iteration2.ID)
	assert.NotEqual(rest.T(), uuid.UUID{}, iteration2.ID)

	wirepo := workitem.NewWorkItemRepository(rest.DB)

	for i := 0; i < 3; i++ {
		wirepo.Create(
			rest.ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: iteration1.ID.String(),
			}, rest.testIdentity.ID)
	}
	for i := 0; i < 2; i++ {
		_, err := wirepo.Create(
			rest.ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("Closed issue #%d", i),
				workitem.SystemState:     workitem.SystemStateClosed,
				workitem.SystemIteration: iteration1.ID.String(),
			}, rest.testIdentity.ID)
		require.Nil(rest.T(), err)
	}
	svc, ctrl := rest.UnSecuredController()
	// when
	_, cs := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceInstance.ID.String(), nil, nil)
	// then
	require.Len(rest.T(), cs.Data, 2)
	for _, iterationItem := range cs.Data {
		if uuid.Equal(*iterationItem.ID, iteration1.ID) {
			assert.Equal(rest.T(), 5, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(rest.T(), 2, iterationItem.Relationships.Workitems.Meta["closed"])
		} else if uuid.Equal(*iterationItem.ID, iteration2.ID) {
			assert.Equal(rest.T(), 0, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(rest.T(), 0, iterationItem.Relationships.Workitems.Meta["closed"])
		}
	}
	// seed 5 WI to iteration2
	for i := 0; i < 5; i++ {
		_, err := wirepo.Create(
			rest.ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: iteration2.ID.String(),
			}, rest.testIdentity.ID)
		require.Nil(rest.T(), err)
	}
	// when
	_, cs = test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceInstance.ID.String(), nil, nil)
	// then
	require.Len(rest.T(), cs.Data, 2)
	for _, iterationItem := range cs.Data {
		if uuid.Equal(*iterationItem.ID, iteration1.ID) {
			assert.Equal(rest.T(), 5, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(rest.T(), 2, iterationItem.Relationships.Workitems.Meta["closed"])
		} else if uuid.Equal(*iterationItem.ID, iteration2.ID) {
			assert.Equal(rest.T(), 5, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(rest.T(), 0, iterationItem.Relationships.Workitems.Meta["closed"])
		}
	}
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
		p, err := app.Spaces().Create(rest.ctx, &newSpace)
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
			repo.Create(rest.ctx, &i)
		}
		// create one child iteration and test for relationships.Parent
		fatherIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: spaceID,
		}
		repo.Create(rest.ctx, fatherIteration)
		rest.T().Log("fatherIteration:", fatherIteration.ID, fatherIteration.Name, fatherIteration.Path)
		childIteration = &iteration.Iteration{
			Name:    "Child Iteration",
			SpaceID: spaceID,
			Path:    append(fatherIteration.Path, fatherIteration.ID),
		}
		repo.Create(rest.ctx, childIteration)
		rest.T().Log("childIteration:", childIteration.ID, childIteration.Name, childIteration.Path)
		grandChildIteration = &iteration.Iteration{
			Name:    "Grand Child Iteration",
			SpaceID: spaceID,
			Path:    append(childIteration.Path, childIteration.ID),
		}
		repo.Create(rest.ctx, grandChildIteration)
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
		assert.Equal(t, 0, iterationItem.Relationships.Workitems.Meta["total"])
		assert.Equal(t, 0, iterationItem.Relationships.Workitems.Meta["closed"])
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
	modelEntities := make([]app.ConditionalResponseEntity, len(iterations.Data))
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
