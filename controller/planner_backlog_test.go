package controller_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestPlannerBlacklogREST struct {
	gormtestsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
	ctx   context.Context
}

func TestRunPlannerBlacklogREST(t *testing.T) {
	suite.Run(t, &TestPlannerBlacklogREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestPlannerBlacklogREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)

	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	rest.ctx = goa.NewContext(context.Background(), nil, req, params)
}

func (rest *TestPlannerBlacklogREST) TearDownTest() {
	rest.clean()
}

func (rest *TestPlannerBlacklogREST) UnSecuredController() (*goa.Service, *PlannerBacklogController) {
	svc := goa.New("PlannerBlacklog-Service")
	return svc, NewPlannerBacklogController(svc, rest.db)
}

func (rest *TestPlannerBlacklogREST) TestSuccessCreateIteration() {
	t := rest.T()
	resource.Require(t, resource.Database)

	var spaceID uuid.UUID
	var fatherIteration, childIteration, grandChildIteration *iteration.Iteration
	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Iterations()

		newSpace := space.Space{
			Name: "Test 1",
		}
		p, err := app.Spaces().Create(rest.ctx, &newSpace)
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
			repo.Create(rest.ctx, &i)
		}

		// create one child iteration and test for relationships.Parent
		fatherIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: spaceID,
		}
		repo.Create(rest.ctx, fatherIteration)

		childIteration = &iteration.Iteration{
			Name:    "Child Iteration",
			SpaceID: spaceID,
			Path:    append(fatherIteration.Path, fatherIteration.ID),
		}
		repo.Create(rest.ctx, childIteration)

		grandChildIteration = &iteration.Iteration{
			Name:    "Grand Child Iteration",
			SpaceID: spaceID,
			Path:    append(childIteration.Path, childIteration.ID),
		}
		repo.Create(rest.ctx, grandChildIteration)

		return nil
	})

	svc, ctrl := rest.UnSecuredController()
	page := "0,-1"
	_, cs := test.ListPlannerBacklogOK(t, svc.Context, svc, ctrl, spaceID.String(), &page)
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

func (rest *TestPlannerBlacklogREST) TestFailListPlannerBacklogByMissingSpace() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.UnSecuredController()
	page := "0,-1"
	test.ListPlannerBacklogNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), &page)
}
