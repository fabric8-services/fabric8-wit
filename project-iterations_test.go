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
	"github.com/almighty/almighty-core/project"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestProjectIterationREST struct {
	gormsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunProjectIterationREST(t *testing.T) {
	suite.Run(t, &TestProjectIterationREST{DBTestSuite: gormsupport.NewDBTestSuite("config.yaml")})
}

func (rest *TestProjectIterationREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = gormsupport.DeleteCreatedEntities(rest.DB)
}

func (rest *TestProjectIterationREST) TearDownTest() {
	rest.clean()
}

func (rest *TestProjectIterationREST) SecuredController() (*goa.Service, *ProjectIterationsController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Iteration-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
	return svc, NewProjectIterationsController(svc, rest.db)
}

func (rest *TestProjectIterationREST) UnSecuredController() (*goa.Service, *ProjectIterationsController) {
	svc := goa.New("Iteration-Service")
	return svc, NewProjectIterationsController(svc, rest.db)
}

func (rest *TestProjectIterationREST) TestSuccessCreateIteration() {
	t := rest.T()
	resource.Require(t, resource.Database)

	var p *project.Project
	ci := createProjectIteration("Sprint #21")

	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Projects()
		p, _ = repo.Create(context.Background(), "Test 1")
		return nil
	})
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateProjectIterationsCreated(t, svc.Context, svc, ctrl, p.ID.String(), ci)
	assert.NotNil(t, c.Data.ID)
	assert.NotNil(t, c.Data.Relationships.Project)
	assert.Equal(t, p.ID.String(), *c.Data.Relationships.Project.Data.ID)
}

func (rest *TestProjectIterationREST) TestListIterationsByProject() {
	t := rest.T()
	resource.Require(t, resource.Database)

	var projectID uuid.UUID
	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Iterations()

		p, err := app.Projects().Create(context.Background(), "Test 1")
		if err != nil {
			t.Error(err)
		}
		projectID = p.ID

		for i := 0; i < 3; i++ {
			start := time.Now()
			end := start.Add(time.Hour * (24 * 8 * 3))
			name := "Sprint #2" + strconv.Itoa(i)

			i := iteration.Iteration{
				Name:      name,
				ProjectID: projectID,
				StartAt:   &start,
				EndAt:     &end,
			}
			repo.Create(context.Background(), &i)
		}
		return nil
	})

	svc, ctrl := rest.UnSecuredController()
	_, cs := test.ListProjectIterationsOK(t, svc.Context, svc, ctrl, projectID.String(), nil, nil, nil)
	assert.Len(t, cs.Data, 3)
}

func (rest *TestProjectIterationREST) TestCreateIterationMissingProject() {
	t := rest.T()
	resource.Require(t, resource.Database)

	ci := createProjectIteration("Sprint #21")

	svc, ctrl := rest.SecuredController()
	test.CreateProjectIterationsNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), ci)
}

func (rest *TestProjectIterationREST) TestFailCreateIterationNotAuthorized() {
	t := rest.T()
	resource.Require(t, resource.Database)

	ci := createProjectIteration("Sprint #21")

	svc, ctrl := rest.UnSecuredController()
	test.CreateProjectIterationsUnauthorized(t, svc.Context, svc, ctrl, uuid.NewV4().String(), ci)
}

func (rest *TestProjectIterationREST) TestFailListIterationsByMissingProject() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.UnSecuredController()
	test.ListProjectIterationsNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String(), nil, nil, nil)
}

func createProjectIteration(name string) *app.CreateProjectIterationsPayload {
	start := time.Now()
	end := start.Add(time.Hour * (24 * 8 * 3))

	return &app.CreateProjectIterationsPayload{
		Data: &app.Iteration{
			Type: "iterations",
			Attributes: &app.IterationAttributes{
				Name:    &name,
				StartAt: &start,
				EndAt:   &end,
			},
		},
	}
}
