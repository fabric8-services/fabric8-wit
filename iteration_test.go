package main_test

import (
	"strings"
	"testing"
	"time"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type TestIterationREST struct {
	gormsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunIterationREST(t *testing.T) {
	suite.Run(t, &TestIterationREST{DBTestSuite: gormsupport.NewDBTestSuite("config.yaml")})
}

func (rest *TestIterationREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestIterationREST) TearDownTest() {
	rest.clean()
}

func (rest *TestIterationREST) SecuredController() (*goa.Service, *IterationController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Iteration-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewIterationController(svc, rest.db)
}

func (rest *TestIterationREST) UnSecuredController() (*goa.Service, *IterationController) {
	svc := goa.New("Iteration-Service")
	return svc, NewIterationController(svc, rest.db)
}

func (rest *TestIterationREST) TestSuccessCreateChildIteration() {
	t := rest.T()
	resource.Require(t, resource.Database)

	parentID := createSpaceAndIteration(t, rest.db).ID
	name := "Sprint #21"
	ci := createChildIteration(&name)

	svc, ctrl := rest.SecuredController()
	_, created := test.CreateChildIterationCreated(t, svc.Context, svc, ctrl, parentID.String(), ci)
	assertChildIterationLinking(t, created.Data)
	assert.Equal(t, *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
}

func (rest *TestIterationREST) TestFailCreateChildIterationMissingName() {
	t := rest.T()
	resource.Require(t, resource.Database)

	parentID := createSpaceAndIteration(t, rest.db).ID
	ci := createChildIteration(nil)

	svc, ctrl := rest.SecuredController()
	test.CreateChildIterationBadRequest(t, svc.Context, svc, ctrl, parentID.String(), ci)
}

func (rest *TestIterationREST) TestFailCreateChildIterationMissingParent() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Sprint #21"
	ci := createChildIteration(&name)

	svc, ctrl := rest.SecuredController()
	test.CreateChildIterationNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), ci)
}

func (rest *TestIterationREST) TestFailCreateChildIterationNotAuthorized() {
	t := rest.T()
	resource.Require(t, resource.Database)

	parentID := createSpaceAndIteration(t, rest.db).ID
	name := "Sprint #21"
	ci := createChildIteration(&name)

	svc, ctrl := rest.UnSecuredController()
	test.CreateChildIterationUnauthorized(t, svc.Context, svc, ctrl, parentID.String(), ci)
}

func (rest *TestIterationREST) TestSuccessShowIteration() {
	t := rest.T()
	resource.Require(t, resource.Database)

	itrID := createSpaceAndIteration(t, rest.db)

	svc, ctrl := rest.SecuredController()
	_, created := test.ShowIterationOK(t, svc.Context, svc, ctrl, itrID.ID.String())
	assertIterationLinking(t, created.Data)
}

func (rest *TestIterationREST) TestFailShowIterationMissing() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.SecuredController()
	test.ShowIterationNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String())
}

func (rest *TestIterationREST) TestSuccessUpdateIteration() {
	t := rest.T()
	resource.Require(t, resource.Database)

	itr := createSpaceAndIteration(t, rest.db)

	newName := "Sprint 1001"
	newDesc := "New Description"
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{
				Name:        &newName,
				Description: &newDesc,
			},
			ID:   &itr.ID,
			Type: iteration.APIStringTypeIteration,
		},
	}
	svc, ctrl := rest.SecuredController()
	_, updated := test.UpdateIterationOK(t, svc.Context, svc, ctrl, itr.ID.String(), &payload)
	assert.Equal(t, newName, *updated.Data.Attributes.Name)
	assert.Equal(t, newDesc, *updated.Data.Attributes.Description)
}

func (rest *TestIterationREST) TestFailUpdateIterationNotFound() {
	t := rest.T()
	resource.Require(t, resource.Database)
	itr := createSpaceAndIteration(t, rest.db)
	itr.ID = uuid.NewV4()
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{},
			ID:         &itr.ID,
			Type:       iteration.APIStringTypeIteration,
		},
	}
	svc, ctrl := rest.SecuredController()
	test.UpdateIterationNotFound(t, svc.Context, svc, ctrl, itr.ID.String(), &payload)
}

func (rest *TestIterationREST) TestFailUpdateIterationUnauthorized() {
	t := rest.T()
	resource.Require(t, resource.Database)
	itr := createSpaceAndIteration(t, rest.db)
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{},
			ID:         &itr.ID,
			Type:       iteration.APIStringTypeIteration,
		},
	}
	svc, ctrl := rest.UnSecuredController()
	test.UpdateIterationUnauthorized(t, svc.Context, svc, ctrl, itr.ID.String(), &payload)
}

func (rest *TestIterationREST) TestIterationStateTransitions() {
	t := rest.T()
	resource.Require(t, resource.Database)

	itr1 := createSpaceAndIteration(t, rest.db)
	assert.Equal(t, iteration.IterationStateNew, itr1.State)

	startState := iteration.IterationStateStart
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{
				State: &startState,
			},
			ID:   &itr1.ID,
			Type: iteration.APIStringTypeIteration,
		},
	}
	svc, ctrl := rest.SecuredController()
	_, updated := test.UpdateIterationOK(t, svc.Context, svc, ctrl, itr1.ID.String(), &payload)
	assert.Equal(t, startState, *updated.Data.Attributes.State)

	// create another iteration in same space and then change State to start
	itr2 := iteration.Iteration{
		Name:    "Spring 123",
		SpaceID: itr1.SpaceID,
	}
	err := rest.db.Iterations().Create(context.Background(), &itr2)
	require.Nil(t, err)
	payload2 := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{
				State: &startState,
			},
			ID:   &itr2.ID,
			Type: iteration.APIStringTypeIteration,
		},
	}
	test.UpdateIterationBadRequest(t, svc.Context, svc, ctrl, itr2.ID.String(), &payload2)

	// now close first iteration
	closeState := iteration.IterationStateClose
	payload.Data.Attributes.State = &closeState
	_, updated = test.UpdateIterationOK(t, svc.Context, svc, ctrl, itr1.ID.String(), &payload)
	assert.Equal(t, closeState, *updated.Data.Attributes.State)

	// try to start iteration 2 now
	_, updated2 := test.UpdateIterationOK(t, svc.Context, svc, ctrl, itr2.ID.String(), &payload2)
	assert.Equal(t, startState, *updated2.Data.Attributes.State)
}

func createChildIteration(name *string) *app.CreateChildIterationPayload {
	start := time.Now()
	end := start.Add(time.Hour * (24 * 8 * 3))

	itType := iteration.APIStringTypeIteration

	return &app.CreateChildIterationPayload{
		Data: &app.Iteration{
			Type: itType,
			Attributes: &app.IterationAttributes{
				Name:    name,
				StartAt: &start,
				EndAt:   &end,
			},
		},
	}
}

func createSpaceAndIteration(t *testing.T, db *gormapplication.GormDB) iteration.Iteration {
	var itr iteration.Iteration
	application.Transactional(db, func(app application.Application) error {
		repo := app.Iterations()

		newSpace := space.Space{
			Name: "Test 1" + uuid.NewV4().String(),
		}
		p, err := app.Spaces().Create(context.Background(), &newSpace)
		if err != nil {
			t.Error(err)
		}
		start := time.Now()
		end := start.Add(time.Hour * (24 * 8 * 3))
		name := "Sprint #2"

		i := iteration.Iteration{
			Name:    name,
			SpaceID: p.ID,
			StartAt: &start,
			EndAt:   &end,
		}
		repo.Create(context.Background(), &i)
		itr = i
		return nil
	})
	return itr
}

func assertIterationLinking(t *testing.T, target *app.Iteration) {
	assert.NotNil(t, target.ID)
	assert.Equal(t, iteration.APIStringTypeIteration, target.Type)
	assert.NotNil(t, target.Links.Self)
	require.NotNil(t, target.Relationships)
	require.NotNil(t, target.Relationships.Space)
	require.NotNil(t, target.Relationships.Space.Links)
	require.NotNil(t, target.Relationships.Space.Links.Self)
	assert.True(t, strings.Contains(*target.Relationships.Space.Links.Self, "/api/spaces/"))
}

func assertChildIterationLinking(t *testing.T, target *app.Iteration) {
	assertIterationLinking(t, target)
	require.NotNil(t, target.Relationships)
	require.NotNil(t, target.Relationships.Parent)
	require.NotNil(t, target.Relationships.Parent.Links)
	require.NotNil(t, target.Relationships.Parent.Links.Self)
}
