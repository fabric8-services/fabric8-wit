package main_test

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestCommentREST struct {
	gormsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunCommentREST(t *testing.T) {
	suite.Run(t, &TestCommentREST{DBTestSuite: gormsupport.NewDBTestSuite("config.yaml")})
}

func (rest *TestCommentREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = gormsupport.DeleteCreatedEntities(rest.DB)
}

func (rest *TestCommentREST) TearDownTest() {
	rest.clean()
}

func (rest *TestCommentREST) SecuredController() (*goa.Service, *WorkItemCommentsController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("WorkItemComment-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
	return svc, NewWorkItemCommentsController(svc, rest.db)
}

func (rest *TestCommentREST) UnSecuredController() (*goa.Service, *WorkItemCommentsController) {
	svc := goa.New("WorkItemComment-Service")
	return svc, NewWorkItemCommentsController(svc, rest.db)
}

func (rest *TestCommentREST) TestSuccessCreateSingleComment() {
	t := rest.T()
	resource.Require(t, resource.Database)

	wiid, err := createWorkItem(rest.db)
	if err != nil {
		t.Error(err)
	}

	p := createComment("Test")

	svc, ctrl := rest.SecuredController()
	_, c := test.CreateWorkItemCommentsOK(t, svc.Context, svc, ctrl, wiid, p)
	assertComment(t, c.Data)
}

func (rest *TestCommentREST) TestListCommentsByParentWorkItem() {
	t := rest.T()
	resource.Require(t, resource.Database)

	wiid, err := createWorkItem(rest.db)
	if err != nil {
		t.Error(err)
	}
	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Comments()
		repo.Create(context.Background(), &comment.Comment{ParentID: wiid, Body: "Test 1", CreatedBy: uuid.NewV4()})
		repo.Create(context.Background(), &comment.Comment{ParentID: wiid, Body: "Test 2", CreatedBy: uuid.NewV4()})
		repo.Create(context.Background(), &comment.Comment{ParentID: wiid, Body: "Test 3", CreatedBy: uuid.NewV4()})
		repo.Create(context.Background(), &comment.Comment{ParentID: wiid + "_other", Body: "Test 1", CreatedBy: uuid.NewV4()})
		return nil
	})

	svc, ctrl := rest.UnSecuredController()
	_, cs := test.ListWorkItemCommentsOK(t, svc.Context, svc, ctrl, wiid)
	if len(cs.Data) != 3 {
		t.Error("Listed comments of wrong length")
	}
	assertComment(t, cs.Data[0])
}

func (rest *TestCommentREST) TestEmptyListCommentsByParentWorkItem() {
	t := rest.T()
	resource.Require(t, resource.Database)

	wiid, err := createWorkItem(rest.db)
	if err != nil {
		t.Error(err)
	}

	svc, ctrl := rest.UnSecuredController()
	_, cs := test.ListWorkItemCommentsOK(t, svc.Context, svc, ctrl, wiid)
	if len(cs.Data) != 0 {
		t.Error("Listed comments of wrong length")
	}
}

func (rest *TestCommentREST) TestCreateSingleCommentMissingWorkItem() {
	t := rest.T()
	resource.Require(t, resource.Database)

	p := createComment("Test")

	svc, ctrl := rest.SecuredController()
	test.CreateWorkItemCommentsNotFound(t, svc.Context, svc, ctrl, "0000000", p)
}

func (rest *TestCommentREST) TestCreateSingleNoAuthorized() {
	t := rest.T()
	resource.Require(t, resource.Database)

	wiid, err := createWorkItem(rest.db)
	if err != nil {
		t.Error(err)
	}

	p := createComment("Test")

	svc, ctrl := rest.UnSecuredController()
	test.CreateWorkItemCommentsUnauthorized(t, svc.Context, svc, ctrl, wiid, p)
}

// Can not be tested via normal Goa testing framework as setting empty body on CreateCommentAttributes is
// validated before Request to service is made. Validate model and assume it will be validated before hitting
// service when mounted. Test to show intent.
func (rest *TestCommentREST) TestShouldNotCreateEmptyBody() {
	t := rest.T()

	p := createComment("")
	err := p.Validate()
	if err == nil {
		t.Error("Should not allow empty body", err)
	}
}

func (rest *TestCommentREST) TestListCommentsByMissingParentWorkItem() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.SecuredController()
	test.ListWorkItemCommentsNotFound(t, svc.Context, svc, ctrl, "0000000")
}

func assertComment(t *testing.T, c *app.Comment) {
	assert.NotNil(t, c)
	assert.Equal(t, "comments", c.Type)
	assert.NotNil(t, c.ID)
	assert.NotNil(t, c.Attributes.Body)
	assert.NotNil(t, c.Attributes.CreatedAt)
	assert.WithinDuration(t, time.Now(), *c.Attributes.CreatedAt, 2*time.Second)
	assert.NotNil(t, c.Relationships)
	assert.NotNil(t, c.Relationships.CreatedBy)
	assert.Equal(t, "identities", c.Relationships.CreatedBy.Data.Type)
	assert.NotNil(t, c.Relationships.CreatedBy.Data.ID)
}

func createComment(body string) *app.CreateWorkItemCommentsPayload {
	return &app.CreateWorkItemCommentsPayload{
		Data: &app.CreateComment{
			Type: "comments",
			Attributes: &app.CreateCommentAttributes{
				Body: body,
			},
		},
	}
}

func createWorkItem(db *gormapplication.GormDB) (string, error) {
	var wiid string
	err := application.Transactional(db, func(appl application.Application) error {
		repo := appl.WorkItems()
		wi, err := repo.Create(
			context.Background(),
			workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle: "A",
				workitem.SystemState: "new",
			},
			uuid.NewV4().String())
		if err != nil {
			return errors.WithStack(err)
		}
		wiid = wi.ID
		return nil
	})
	if err != nil {
		return "", errors.WithStack(err)
	}
	return wiid, nil
}
