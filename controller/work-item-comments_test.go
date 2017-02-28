package controller_test

import (
	"html"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/comment"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestCommentREST struct {
	gormsupport.DBTestSuite

	db    *gormapplication.GormDB
	clean func()
}

func TestRunCommentREST(t *testing.T) {
	suite.Run(t, &TestCommentREST{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestCommentREST) SetupTest() {
	resource.Require(rest.T(), resource.Database)
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestCommentREST) TearDownTest() {
	rest.clean()
}

func (rest *TestCommentREST) SecuredController() (*goa.Service, *WorkItemCommentsController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("WorkItemComment-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewWorkItemCommentsController(svc, rest.db)
}

func (rest *TestCommentREST) UnSecuredController() (*goa.Service, *WorkItemCommentsController) {
	svc := goa.New("WorkItemComment-Service")
	return svc, NewWorkItemCommentsController(svc, rest.db)
}

func (rest *TestCommentREST) newCreateWorkItemCommentsPayload(body string, markup *string) *app.CreateWorkItemCommentsPayload {
	return &app.CreateWorkItemCommentsPayload{
		Data: &app.CreateComment{
			Type: "comments",
			Attributes: &app.CreateCommentAttributes{
				Body:   body,
				Markup: markup,
			},
		},
	}
}

func (rest *TestCommentREST) createDefaultWorkItem() string {
	var wiid string
	err := application.Transactional(rest.db, func(appl application.Application) error {
		repo := appl.WorkItems()
		wi, err := repo.Create(
			context.Background(),
			workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle: "A",
				workitem.SystemState: "new",
			},
			uuid.NewV4(), space.SystemSpace)
		if err != nil {
			return errors.WithStack(err)
		}
		wiid = wi.ID
		return nil
	})
	require.Nil(rest.T(), err)
	return wiid
}

func (rest *TestCommentREST) assertComment(c *app.Comment, expectedBody string, expectedMarkup string) {
	assert.NotNil(rest.T(), c)
	assert.Equal(rest.T(), "comments", c.Type)
	assert.NotNil(rest.T(), c.ID)
	require.NotNil(rest.T(), c.Attributes)
	assert.Equal(rest.T(), expectedBody, *c.Attributes.Body)
	assert.Equal(rest.T(), expectedMarkup, *c.Attributes.Markup)
	assert.Equal(rest.T(), rendering.RenderMarkupToHTML(html.EscapeString(expectedBody), expectedMarkup), *c.Attributes.BodyRendered)
	require.NotNil(rest.T(), c.Attributes.CreatedAt)
	assert.WithinDuration(rest.T(), time.Now(), *c.Attributes.CreatedAt, 2*time.Second)
	require.NotNil(rest.T(), c.Relationships)
	require.NotNil(rest.T(), c.Relationships.CreatedBy)
	require.NotNil(rest.T(), c.Relationships.CreatedBy.Data)
	assert.Equal(rest.T(), "identities", c.Relationships.CreatedBy.Data.Type)
	assert.NotNil(rest.T(), c.Relationships.CreatedBy.Data.ID)
}

func (rest *TestCommentREST) TestSuccessCreateSingleCommentWithMarkup() {
	// given
	wiid := rest.createDefaultWorkItem()
	// when
	markup := rendering.SystemMarkupMarkdown
	p := rest.newCreateWorkItemCommentsPayload("Test", &markup)
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wiid, p)
	// then
	rest.assertComment(c.Data, "Test", markup)
}

func (rest *TestCommentREST) TestSuccessCreateSingleCommentWithDefaultMarkup() {
	// given
	wiid := rest.createDefaultWorkItem()
	// when
	p := rest.newCreateWorkItemCommentsPayload("Test", nil)
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wiid, p)
	// then
	rest.assertComment(c.Data, "Test", rendering.SystemMarkupDefault)
}

func (rest *TestCommentREST) TestListCommentsByParentWorkItem() {
	// given
	wiid := rest.createDefaultWorkItem()
	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Comments()
		repo.Create(context.Background(), &comment.Comment{ParentID: wiid, Body: "Test 1", CreatedBy: uuid.NewV4()})
		repo.Create(context.Background(), &comment.Comment{ParentID: wiid, Body: "Test 2", CreatedBy: uuid.NewV4()})
		repo.Create(context.Background(), &comment.Comment{ParentID: wiid, Body: "Test 3", CreatedBy: uuid.NewV4()})
		repo.Create(context.Background(), &comment.Comment{ParentID: wiid + "_other", Body: "Test 1", CreatedBy: uuid.NewV4()})
		return nil
	})
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 3
	_, cs := test.ListWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wiid, &limit, &offset)
	// then
	require.Equal(rest.T(), 3, len(cs.Data))
	rest.assertComment(cs.Data[0], "Test 3", rendering.SystemMarkupDefault) // items are returned in reverse order or creation
	// given
	wiid2 := rest.createDefaultWorkItem()
	// when
	_, cs2 := test.ListWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wiid2, &limit, &offset)
	// then
	assert.Equal(rest.T(), 0, len(cs2.Data))
}

func (rest *TestCommentREST) TestEmptyListCommentsByParentWorkItem() {
	// given
	wiid := rest.createDefaultWorkItem()
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 1
	_, cs := test.ListWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wiid, &limit, &offset)
	// then
	assert.Equal(rest.T(), 0, len(cs.Data))
}

func (rest *TestCommentREST) TestCreateSingleCommentMissingWorkItem() {
	// given
	p := rest.newCreateWorkItemCommentsPayload("Test", nil)
	// when/then
	svc, ctrl := rest.SecuredController()
	test.CreateWorkItemCommentsNotFound(rest.T(), svc.Context, svc, ctrl, "0000000", p)
}

func (rest *TestCommentREST) TestCreateSingleNoAuthorized() {
	// given
	wiid := rest.createDefaultWorkItem()
	// when/then
	p := rest.newCreateWorkItemCommentsPayload("Test", nil)
	svc, ctrl := rest.UnSecuredController()
	test.CreateWorkItemCommentsUnauthorized(rest.T(), svc.Context, svc, ctrl, wiid, p)
}

// Can not be tested via normal Goa testing framework as setting empty body on CreateCommentAttributes is
// validated before Request to service is made. Validate model and assume it will be validated before hitting
// service when mounted. Test to show intent.
func (rest *TestCommentREST) TestShouldNotCreateEmptyBody() {
	// given
	p := rest.newCreateWorkItemCommentsPayload("", nil)
	// when
	err := p.Validate()
	// then
	require.NotNil(rest.T(), err)
}

func (rest *TestCommentREST) TestListCommentsByMissingParentWorkItem() {
	// given
	svc, ctrl := rest.SecuredController()
	// when/then
	offset := "0"
	limit := 1
	test.ListWorkItemCommentsNotFound(rest.T(), svc.Context, svc, ctrl, "0000000", &limit, &offset)
}
