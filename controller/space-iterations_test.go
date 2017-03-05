package controller_test

import (
	"strconv"
	"testing"
	"time"

	"golang.org/x/net/context"

	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/iteration"
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
	gormsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunSpaceIterationREST(t *testing.T) {
	suite.Run(t, &TestSpaceIterationREST{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestSpaceIterationREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestSpaceIterationREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSpaceIterationREST) SecuredController() (*goa.Service, *SpaceIterationsController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Iteration-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewSpaceIterationsController(svc, rest.db)
}

func (rest *TestSpaceIterationREST) UnSecuredController() (*goa.Service, *SpaceIterationsController) {
	svc := goa.New("Iteration-Service")
	return svc, NewSpaceIterationsController(svc, rest.db)
}

func (rest *TestSpaceIterationREST) TestSuccessCreateIteration() {
	t := rest.T()
	resource.Require(t, resource.Database)

	var p *space.Space
	ci := createSpaceIteration("Sprint #21", nil)

	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Spaces()
		newSpace := space.Space{
			Name: "Test 1",
		}
		p, _ = repo.Create(context.Background(), &newSpace)
		return nil
	})
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateSpaceIterationsCreated(t, svc.Context, svc, ctrl, p.ID.String(), ci)
	require.NotNil(t, c.Data.ID)
	require.NotNil(t, c.Data.Relationships.Space)
	assert.Equal(t, p.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(t, iteration.IterationStateNew, *c.Data.Attributes.State)
	assert.Equal(t, "/", *c.Data.Attributes.ParentPath)
	require.NotNil(t, c.Data.Relationships.Workitems.Meta)
	assert.Equal(t, 0, c.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(t, 0, c.Data.Relationships.Workitems.Meta["closed"])
}

func (rest *TestSpaceIterationREST) TestSuccessCreateIterationWithOptionalValues() {
	t := rest.T()
	resource.Require(t, resource.Database)

	var p *space.Space
	iterationName := "Sprint #22"
	iterationDesc := "testing description"
	ci := createSpaceIteration(iterationName, &iterationDesc)

	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Spaces()
		testSpace := space.Space{
			Name: "Test 1",
		}
		p, _ = repo.Create(context.Background(), &testSpace)
		return nil
	})
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateSpaceIterationsCreated(t, svc.Context, svc, ctrl, p.ID.String(), ci)
	assert.NotNil(t, c.Data.ID)
	assert.NotNil(t, c.Data.Relationships.Space)
	assert.Equal(t, p.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(t, *c.Data.Attributes.Name, iterationName)
	assert.Equal(t, *c.Data.Attributes.Description, iterationDesc)

	// create another Iteration with nil description
	iterationName2 := "Sprint #23"
	ci = createSpaceIteration(iterationName2, nil)
	_, c = test.CreateSpaceIterationsCreated(t, svc.Context, svc, ctrl, p.ID.String(), ci)
	assert.Equal(t, *c.Data.Attributes.Name, iterationName2)
	assert.Nil(t, c.Data.Attributes.Description)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpace() {
	t := rest.T()
	resource.Require(t, resource.Database)

	var spaceID uuid.UUID
	var fatherIteration, childIteration, grandChildIteration *iteration.Iteration
	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Iterations()

		newSpace := space.Space{
			Name: "Test 1",
		}
		p, err := app.Spaces().Create(context.Background(), &newSpace)
		if err != nil {
			t.Error(err)
		}
		spaceID = p.ID

		for i := 0; i < 3; i++ {
			start := time.Now()
			end := start.Add(time.Hour * (24 * 8 * 3))
			name := "Sprint #2" + strconv.Itoa(i)

			i := iteration.Iteration{
				Name:    name,
				SpaceID: spaceID,
				StartAt: &start,
				EndAt:   &end,
			}
			repo.Create(context.Background(), &i)
		}

		// create one child iteration and test for relationships.Parent
		fatherIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: spaceID,
		}
		repo.Create(context.Background(), fatherIteration)

		childIteration = &iteration.Iteration{
			Name:    "Child Iteration",
			SpaceID: spaceID,
			Path:    iteration.ConvertToLtreeFormat(fatherIteration.ID.String()),
		}
		repo.Create(context.Background(), childIteration)

		grandChildIteration = &iteration.Iteration{
			Name:    "Grand Child Iteration",
			SpaceID: spaceID,
			Path:    iteration.ConvertToLtreeFormat(fatherIteration.ID.String() + iteration.PathSepInDatabase + childIteration.ID.String()),
		}
		repo.Create(context.Background(), grandChildIteration)

		return nil
	})

	svc, ctrl := rest.UnSecuredController()
	_, cs := test.ListSpaceIterationsOK(t, svc.Context, svc, ctrl, spaceID.String())
	assert.Len(t, cs.Data, 6)
	for _, iterationItem := range cs.Data {
		subString := fmt.Sprintf("?filter[iteration]=%s", iterationItem.ID.String())
		require.Contains(t, *iterationItem.Relationships.Workitems.Links.Related, subString)
		assert.Equal(t, 0, iterationItem.Relationships.Workitems.Meta["total"])
		assert.Equal(t, 0, iterationItem.Relationships.Workitems.Meta["closed"])
		if *iterationItem.ID == childIteration.ID {
			expectedParentPath := iteration.PathSepInService + fatherIteration.ID.String()
			expectedResolvedParentPath := iteration.PathSepInService + fatherIteration.Name
			require.NotNil(t, iterationItem.Relationships.Parent)
			assert.Equal(t, fatherIteration.ID.String(), *iterationItem.Relationships.Parent.Data.ID)
			assert.Equal(t, expectedParentPath, *iterationItem.Attributes.ParentPath)
			assert.Equal(t, expectedResolvedParentPath, *iterationItem.Attributes.ResolvedParentPath)
		}
		if *iterationItem.ID == grandChildIteration.ID {
			expectedParentPath := iteration.PathSepInService + fatherIteration.ID.String() + iteration.PathSepInService + childIteration.ID.String()
			expectedResolvedParentPath := iteration.PathSepInService + fatherIteration.Name + iteration.PathSepInService + childIteration.Name
			require.NotNil(t, iterationItem.Relationships.Parent)
			assert.Equal(t, childIteration.ID.String(), *iterationItem.Relationships.Parent.Data.ID)
			assert.Equal(t, expectedParentPath, *iterationItem.Attributes.ParentPath)
			assert.Equal(t, expectedResolvedParentPath, *iterationItem.Attributes.ResolvedParentPath)

		}
	}
}

func (rest *TestSpaceIterationREST) TestCreateIterationMissingSpace() {
	t := rest.T()
	resource.Require(t, resource.Database)

	ci := createSpaceIteration("Sprint #21", nil)

	svc, ctrl := rest.SecuredController()
	test.CreateSpaceIterationsNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), ci)
}

func (rest *TestSpaceIterationREST) TestFailCreateIterationNotAuthorized() {
	t := rest.T()
	resource.Require(t, resource.Database)

	ci := createSpaceIteration("Sprint #21", nil)

	svc, ctrl := rest.UnSecuredController()
	test.CreateSpaceIterationsUnauthorized(t, svc.Context, svc, ctrl, uuid.NewV4().String(), ci)
}

func (rest *TestSpaceIterationREST) TestFailListIterationsByMissingSpace() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.UnSecuredController()
	test.ListSpaceIterationsNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String())
}

func (rest *TestSpaceIterationREST) TestWICountsWithIterationListBySpace() {
	t := rest.T()
	resource.Require(t, resource.Database)
	// create seed data
	spaceRepo := space.NewRepository(rest.DB)
	spaceInstance := space.Space{
		Name: "Testing space",
	}
	_, e := spaceRepo.Create(context.Background(), &spaceInstance)
	require.Nil(rest.T(), e)
	fmt.Println("space id = ", spaceInstance.ID)
	require.NotEqual(rest.T(), uuid.UUID{}, spaceInstance.ID)

	iterationRepo := iteration.NewIterationRepository(rest.DB)
	iteration1 := iteration.Iteration{
		Name:    "Sprint 1",
		SpaceID: spaceInstance.ID,
	}
	iterationRepo.Create(context.Background(), &iteration1)
	fmt.Println("iteration1 id = ", iteration1.ID)
	assert.NotEqual(rest.T(), uuid.UUID{}, iteration1.ID)

	iteration2 := iteration.Iteration{
		Name:    "Sprint 2",
		SpaceID: spaceInstance.ID,
	}
	iterationRepo.Create(context.Background(), &iteration2)
	fmt.Println("iteration2 id = ", iteration2.ID)
	assert.NotEqual(rest.T(), uuid.UUID{}, iteration2.ID)

	wirepo := workitem.NewWorkItemRepository(rest.DB)

	for i := 0; i < 3; i++ {
		wirepo.Create(
			context.Background(), workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: iteration1.ID.String(),
			}, uuid.NewV4())
	}
	for i := 0; i < 2; i++ {
		wirepo.Create(
			context.Background(), workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("Closed issue #%d", i),
				workitem.SystemState:     workitem.SystemStateClosed,
				workitem.SystemIteration: iteration1.ID.String(),
			}, uuid.NewV4())
	}
	svc, ctrl := rest.UnSecuredController()
	_, cs := test.ListSpaceIterationsOK(t, svc.Context, svc, ctrl, spaceInstance.ID.String())
	assert.Len(t, cs.Data, 2)
	for _, iterationItem := range cs.Data {
		if uuid.Equal(*iterationItem.ID, iteration1.ID) {
			assert.Equal(t, 5, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(t, 2, iterationItem.Relationships.Workitems.Meta["closed"])
		} else if uuid.Equal(*iterationItem.ID, iteration2.ID) {
			assert.Equal(t, 0, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(t, 0, iterationItem.Relationships.Workitems.Meta["closed"])
		}
	}
	// seed 5 WI to iteration2
	for i := 0; i < 5; i++ {
		wirepo.Create(
			context.Background(), workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: iteration2.ID.String(),
			}, uuid.NewV4())
	}
	_, cs = test.ListSpaceIterationsOK(t, svc.Context, svc, ctrl, spaceInstance.ID.String())
	assert.Len(t, cs.Data, 2)
	for _, iterationItem := range cs.Data {
		if uuid.Equal(*iterationItem.ID, iteration1.ID) {
			assert.Equal(t, 5, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(t, 2, iterationItem.Relationships.Workitems.Meta["closed"])
		} else if uuid.Equal(*iterationItem.ID, iteration2.ID) {
			assert.Equal(t, 5, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(t, 0, iterationItem.Relationships.Workitems.Meta["closed"])
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
