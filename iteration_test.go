package main_test

import (
	"testing"
	"time"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
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
	rest.clean = gormsupport.DeleteCreatedEntities(rest.DB)
}

func (rest *TestIterationREST) TearDownTest() {
	rest.clean()
}

func (rest *TestIterationREST) SecuredController() (*goa.Service, *IterationController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Iteration-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
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
	_, err := test.UpdateIterationNotFound(t, svc.Context, svc, ctrl, itr.ID.String(), &payload)
	assert.NotEmpty(t, err.Errors)
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

		p, err := app.Spaces().Create(context.Background(), "Test 1"+uuid.NewV4().String())
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
	assert.NotNil(t, target.Relationships)
	assert.NotNil(t, target.Relationships.Space)
	assert.NotNil(t, target.Relationships.Space.Links.Self)
}

func assertChildIterationLinking(t *testing.T, target *app.Iteration) {
	assertIterationLinking(t, target)
	assert.NotNil(t, target.Relationships.Parent)
	assert.NotNil(t, target.Relationships.Parent.Links.Self)
}
