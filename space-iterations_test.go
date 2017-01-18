package main_test

import (
	"strconv"
	"testing"
	"time"

	"golang.org/x/net/context"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
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
)

type TestSpaceIterationREST struct {
	gormsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunSpaceIterationREST(t *testing.T) {
	suite.Run(t, &TestSpaceIterationREST{DBTestSuite: gormsupport.NewDBTestSuite("config.yaml")})
}

func (rest *TestSpaceIterationREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = gormsupport.DeleteCreatedEntities(rest.DB)
}

func (rest *TestSpaceIterationREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSpaceIterationREST) SecuredController() (*goa.Service, *SpaceIterationsController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Iteration-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
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
		p, _ = repo.Create(context.Background(), "Test 1")
		return nil
	})
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateSpaceIterationsCreated(t, svc.Context, svc, ctrl, p.ID.String(), ci)
	require.NotNil(t, c.Data.ID)
	require.NotNil(t, c.Data.Relationships.Space)
	assert.Equal(t, p.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(t, iteration.IterationStateNew, *c.Data.Attributes.State)
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
		p, _ = repo.Create(context.Background(), "Test 1")
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
	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Iterations()

		p, err := app.Spaces().Create(context.Background(), "Test 1")
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
		return nil
	})

	svc, ctrl := rest.UnSecuredController()
	_, cs := test.ListSpaceIterationsOK(t, svc.Context, svc, ctrl, spaceID.String())
	assert.Len(t, cs.Data, 3)
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
