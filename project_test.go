package main_test

import (
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestProjectREST struct {
	gormsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunProjectREST(t *testing.T) {
	suite.Run(t, &TestProjectREST{DBTestSuite: gormsupport.NewDBTestSuite("config.yaml")})
}

func (rest *TestProjectREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = gormsupport.DeleteCreatedEntities(rest.DB)
}

func (rest *TestProjectREST) TearDownTest() {
	rest.clean()
}

func (rest *TestProjectREST) SecuredController() (*goa.Service, *ProjectController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Project-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
	return svc, NewProjectController(svc, rest.db)
}

func (rest *TestProjectREST) UnSecuredController() (*goa.Service, *ProjectController) {
	svc := goa.New("Project-Service")
	return svc, NewProjectController(svc, rest.db)
}

func (rest *TestProjectREST) TestFailCreateProjectUnsecure() {
	t := rest.T()
	resource.Require(t, resource.Database)

	p := minimumRequiredCreateProject()

	svc, ctrl := rest.UnSecuredController()
	test.CreateProjectUnauthorized(t, svc.Context, svc, ctrl, p)
}

func (rest *TestProjectREST) TestFailCreateProjectMissingName() {
	t := rest.T()
	resource.Require(t, resource.Database)

	p := minimumRequiredCreateProject()

	svc, ctrl := rest.SecuredController()
	test.CreateProjectBadRequest(t, svc.Context, svc, ctrl, p)
}

func (rest *TestProjectREST) TestSuccessCreateProject() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 24"

	p := minimumRequiredCreateProject()
	p.Data.Attributes.Name = &name

	svc, ctrl := rest.SecuredController()
	_, created := test.CreateProjectCreated(t, svc.Context, svc, ctrl, p)
	assert.NotNil(t, created.Data)
	assert.NotNil(t, created.Data.Attributes)
	assert.NotNil(t, created.Data.Attributes.CreatedAt)
	assert.NotNil(t, created.Data.Attributes.UpdatedAt)
	assert.NotNil(t, created.Data.Attributes.Name)
	assert.Equal(t, name, *created.Data.Attributes.Name)
	assert.NotNil(t, created.Data.Links)
	assert.NotNil(t, created.Data.Links.Self)
}

func (rest *TestProjectREST) TestSuccessUpdateProject() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 25"
	newName := "Test 26"

	p := minimumRequiredCreateProject()
	p.Data.Attributes.Name = &name

	svc, ctrl := rest.SecuredController()
	_, created := test.CreateProjectCreated(t, svc.Context, svc, ctrl, p)

	u := minimumRequiredUpdateProject()
	u.Data.ID = created.Data.ID
	u.Data.Attributes.Version = created.Data.Attributes.Version
	u.Data.Attributes.Name = &newName

	_, updated := test.UpdateProjectOK(t, svc.Context, svc, ctrl, created.Data.ID.String(), u)
	assert.Equal(t, newName, *updated.Data.Attributes.Name)
}

func (rest *TestProjectREST) TestFailUpdateProjectUnSecure() {
	t := rest.T()
	resource.Require(t, resource.Database)

	u := minimumRequiredUpdateProject()

	svc, ctrl := rest.UnSecuredController()
	test.UpdateProjectUnauthorized(t, svc.Context, svc, ctrl, uuid.NewV4().String(), u)
}

func (rest *TestProjectREST) TestFailUpdateProjectNotFound() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "test"
	version := 0
	id := uuid.NewV4()

	u := minimumRequiredUpdateProject()
	u.Data.Attributes.Name = &name
	u.Data.Attributes.Version = &version
	u.Data.ID = id

	svc, ctrl := rest.SecuredController()
	test.UpdateProjectNotFound(t, svc.Context, svc, ctrl, id.String(), u)
}

func (rest *TestProjectREST) TestFailUpdateProjectMissingName() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 25"

	p := minimumRequiredCreateProject()
	p.Data.Attributes.Name = &name

	svc, ctrl := rest.SecuredController()
	_, created := test.CreateProjectCreated(t, svc.Context, svc, ctrl, p)

	u := minimumRequiredUpdateProject()
	u.Data.ID = created.Data.ID
	u.Data.Attributes.Version = created.Data.Attributes.Version

	test.UpdateProjectBadRequest(t, svc.Context, svc, ctrl, created.Data.ID.String(), u)
}

func (rest *TestProjectREST) TestFailUpdateProjectMissingVersion() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 25"
	newName := "Test 26"

	p := minimumRequiredCreateProject()
	p.Data.Attributes.Name = &name

	svc, ctrl := rest.SecuredController()
	_, created := test.CreateProjectCreated(t, svc.Context, svc, ctrl, p)

	u := minimumRequiredUpdateProject()
	u.Data.ID = created.Data.ID
	u.Data.Attributes.Name = &newName

	test.UpdateProjectBadRequest(t, svc.Context, svc, ctrl, created.Data.ID.String(), u)
}

func (rest *TestProjectREST) TestSuccessShowProject() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 27"
	p := minimumRequiredCreateProject()
	p.Data.Attributes.Name = &name

	svc, ctrl := rest.SecuredController()
	_, created := test.CreateProjectCreated(t, svc.Context, svc, ctrl, p)

	_, fetched := test.ShowProjectOK(t, svc.Context, svc, ctrl, created.Data.ID.String())
	assert.Equal(t, created.Data.ID, fetched.Data.ID)
	assert.Equal(t, *created.Data.Attributes.Name, *fetched.Data.Attributes.Name)
	assert.Equal(t, *created.Data.Attributes.Version, *fetched.Data.Attributes.Version)
}

func (rest *TestProjectREST) TestFailShowProjectNotFound() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.UnSecuredController()
	test.ShowProjectNotFound(t, svc.Context, svc, ctrl, uuid.NewV4().String())
}

func (rest *TestProjectREST) TestFailShowProjectNotFoundBadID() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.UnSecuredController()
	test.ShowProjectNotFound(t, svc.Context, svc, ctrl, "asfasfsaf")
}

func (rest *TestProjectREST) TestSuccessListProjects() {
	t := rest.T()
	resource.Require(t, resource.Database)

	name := "Test 24"

	p := minimumRequiredCreateProject()
	p.Data.Attributes.Name = &name

	svc, ctrl := rest.SecuredController()
	test.CreateProjectCreated(t, svc.Context, svc, ctrl, p)

	_, list := test.ListProjectOK(t, svc.Context, svc, ctrl, nil, nil)
	assert.True(t, len(list.Data) > 0)
}

func minimumRequiredCreateProject() *app.CreateProjectPayload {
	return &app.CreateProjectPayload{
		Data: &app.Project{
			Type:       "projects",
			Attributes: &app.ProjectAttributes{},
		},
	}
}

func minimumRequiredUpdateProject() *app.UpdateProjectPayload {
	return &app.UpdateProjectPayload{
		Data: &app.Project{
			Type:       "projects",
			Attributes: &app.ProjectAttributes{},
		},
	}
}
