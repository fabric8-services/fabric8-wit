package controller_test

import (
	"html"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/ptr"

	"context"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/comment"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	notificationsupport "github.com/fabric8-services/fabric8-wit/test/notification"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestCommentREST struct {
	gormtestsupport.DBTestSuite
	testIdentity account.Identity
	ctx          context.Context
	notification notificationsupport.FakeNotificationChannel
}

func TestRunCommentREST(t *testing.T) {
	suite.Run(t, &TestCommentREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (rest *TestCommentREST) SetupTest() {
	resource.Require(rest.T(), resource.Database)
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestCommentREST setup user", "test provider")
	require.NoError(rest.T(), err)
	rest.testIdentity = *testIdentity
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	rest.ctx = goa.NewContext(context.Background(), nil, req, params)
	rest.notification = notificationsupport.FakeNotificationChannel{}
}

func (rest *TestCommentREST) SecuredController() (*goa.Service, *WorkItemCommentsController) {
	svc := testsupport.ServiceAsUser("WorkItemComment-Service", rest.testIdentity)
	return svc, NewNotifyingWorkItemCommentsController(svc, rest.GormDB, &rest.notification, rest.Configuration)
}

func (rest *TestCommentREST) UnSecuredController() (*goa.Service, *WorkItemCommentsController) {
	svc := goa.New("WorkItemComment-Service")
	return svc, NewWorkItemCommentsController(svc, rest.GormDB, rest.Configuration)
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

func assertWorkItemComment(t *testing.T, c *app.Comment, expectedBody string, expectedMarkup string) {
	assert.NotNil(t, c)
	assert.Equal(t, "comments", c.Type)
	assert.NotNil(t, c.ID)
	require.NotNil(t, c.Attributes)
	assert.Equal(t, expectedBody, *c.Attributes.Body)
	assert.Equal(t, expectedMarkup, *c.Attributes.Markup)
	assert.Equal(t, rendering.RenderMarkupToHTML(html.EscapeString(expectedBody), expectedMarkup), *c.Attributes.BodyRendered)
	require.NotNil(t, c.Attributes.CreatedAt)
	assert.WithinDuration(t, time.Now(), *c.Attributes.CreatedAt, 2*time.Second)
	require.NotNil(t, c.Relationships)
	require.NotNil(t, c.Relationships.Creator)
	require.NotNil(t, c.Relationships.Creator.Data)
	assert.Equal(t, "identities", c.Relationships.Creator.Data.Type)
	assert.NotNil(t, c.Relationships.Creator.Data.ID)
}

func (rest *TestCommentREST) TestSuccessCreateSingleCommentWithParentComment() {
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wi := fxt.WorkItems[0]

	// create parent
	markup := rendering.SystemMarkupMarkdown
	p := rest.newCreateWorkItemCommentsPayload("Test", &markup)
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, p)
	// then
	assertComment(rest.T(), c.Data, rest.testIdentity, "Test", markup)

	// create child
	parentID := c.Data.ID.String()
	child := rest.newCreateWorkItemCommentsPayload("Test Child", &markup)
	child.Data.Relationships = &app.CreateCommentRelations{
		ParentComment: &app.RelationGeneric{
			Data: &app.GenericData{
				Type: ptr.String("comments"),
				ID:   ptr.String(c.Data.ID.String()),
			},
		},
	}
	// make sure the above correctly sets the parentCommentID
	assert.Equal(rest.T(), *child.Data.Relationships.ParentComment.Data.ID, parentID)
	// exec create child
	_, d := test.CreateWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, child)
	// check if the result has the correct parentCommentID
	assert.Equal(rest.T(), c.Data.ID.String(), *d.Data.Relationships.ParentComment.Data.ID)
}

func (rest *TestCommentREST) TestSuccessCreateSingleCommentWithMarkup() {
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wi := fxt.WorkItems[0]

	// when
	markup := rendering.SystemMarkupMarkdown
	p := rest.newCreateWorkItemCommentsPayload("Test", &markup)
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, p)
	// then
	assertComment(rest.T(), c.Data, rest.testIdentity, "Test", markup)
}

func (rest *TestCommentREST) TestSuccessCreateSingleCommentWithDefaultMarkup() {
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wi := fxt.WorkItems[0]
	// when
	p := rest.newCreateWorkItemCommentsPayload("Test", nil)
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, p)
	// then
	assertComment(rest.T(), c.Data, rest.testIdentity, "Test", rendering.SystemMarkupDefault)
}

func (rest *TestCommentREST) TestNotificationSentOnCreate() {
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wi := fxt.WorkItems[0]
	// when
	p := rest.newCreateWorkItemCommentsPayload("Test", nil)
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, p)
	// then
	assert.True(rest.T(), len(rest.notification.Messages) > 0)
	assert.Equal(rest.T(), "comment.create", rest.notification.Messages[0].MessageType)
	assert.Equal(rest.T(), c.Data.ID.String(), rest.notification.Messages[0].TargetID)
}

func (rest *TestCommentREST) setupComments() (workitem.WorkItem, []*comment.Comment) {
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wi := fxt.WorkItems[0]
	comments := make([]*comment.Comment, 4)
	comments[0] = &comment.Comment{ParentID: wi.ID, Body: "Test 1", Creator: rest.testIdentity.ID}
	comments[1] = &comment.Comment{ParentID: wi.ID, Body: "Test 2", Creator: rest.testIdentity.ID}
	comments[2] = &comment.Comment{ParentID: wi.ID, Body: "Test 3", Creator: rest.testIdentity.ID}
	comments[3] = &comment.Comment{ParentID: uuid.NewV4(), Body: "Test 1", Creator: rest.testIdentity.ID}
	application.Transactional(rest.GormDB, func(app application.Application) error {
		repo := app.Comments()
		for _, c := range comments {
			repo.Create(rest.ctx, c, rest.testIdentity.ID)
		}
		return nil
	})
	return *wi, comments
}

func (rest *TestCommentREST) setupCommentsWithParentComments() (workitem.WorkItem, []*comment.Comment) {
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wi := fxt.WorkItems[0]
	comments := make([]*comment.Comment, 3)
	// first entry is the parent comment
	comments[0] = &comment.Comment{ParentID: wi.ID, Body: "Parent Comment", Creator: rest.testIdentity.ID}
	application.Transactional(rest.GormDB, func(app application.Application) error {
		repo := app.Comments()
		repo.Create(rest.ctx, comments[0], rest.testIdentity.ID)
		return nil
	})
	// create the childs
	parentCommentID := id.NullUUID{
		UUID:  comments[0].ID,
		Valid: true,
	}
	comments[1] = &comment.Comment{ParentID: wi.ID, Body: "Child Comment 1", Creator: rest.testIdentity.ID, ParentCommentID: parentCommentID}
	comments[2] = &comment.Comment{ParentID: wi.ID, Body: "Child Comment 2", Creator: rest.testIdentity.ID, ParentCommentID: parentCommentID}
	application.Transactional(rest.GormDB, func(app application.Application) error {
		repo := app.Comments()
		repo.Create(rest.ctx, comments[1], rest.testIdentity.ID)
		repo.Create(rest.ctx, comments[2], rest.testIdentity.ID)
		return nil
	})
	return *wi, comments
}

func assertComments(t *testing.T, expectedIdentity account.Identity, comments *app.CommentList) {
	require.Equal(t, 3, len(comments.Data))
	assertComment(t, comments.Data[0], expectedIdentity, "Test 3", rendering.SystemMarkupDefault) // items are returned in reverse order or creation
}

func (rest *TestCommentREST) TestListCommentsByParentWorkItemOK() {
	// given
	wi, _ := rest.setupComments()
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 3
	res, cs := test.ListWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, &limit, &offset, nil, nil)
	// then
	assertComments(rest.T(), rest.testIdentity, cs)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestCommentREST) TestListCommentsByParentWorkItemOKWithParentComments() {
	// given
	wi, _ := rest.setupCommentsWithParentComments()
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 3
	res, cs := test.ListWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, &limit, &offset, nil, nil)
	// note: the comments are returned in reverse order, [2] is the parent
	parentCommentID := cs.Data[2].ID.String()
	assert.Equal(rest.T(), parentCommentID, *cs.Data[1].Relationships.ParentComment.Data.ID)
	assert.Equal(rest.T(), parentCommentID, *cs.Data[0].Relationships.ParentComment.Data.ID)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestCommentREST) TestListCommentsByParentWorkItemOKUsingExpiredIfModifiedSinceHeader() {
	// given
	wi, comments := rest.setupComments()
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 3
	ifModifiedSince := app.ToHTTPTime(comments[3].UpdatedAt.Add(-1 * time.Hour))
	res, cs := test.ListWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, &limit, &offset, &ifModifiedSince, nil)
	// then
	assertComments(rest.T(), rest.testIdentity, cs)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestCommentREST) TestListCommentsByParentWorkItemOKUsingExpiredIfNoneMatchHeader() {
	// given
	wi, _ := rest.setupComments()
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 3
	ifNoneMatch := "foo"
	res, cs := test.ListWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, &limit, &offset, nil, &ifNoneMatch)
	// then
	assertComments(rest.T(), rest.testIdentity, cs)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestCommentREST) TestListCommentsByParentWorkItemNotModifiedUsingIfModifiedSinceHeader() {
	// given
	wi, comments := rest.setupComments()
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 3
	ifModifiedSince := app.ToHTTPTime(comments[3].UpdatedAt)
	res := test.ListWorkItemCommentsNotModified(rest.T(), svc.Context, svc, ctrl, wi.ID, &limit, &offset, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestCommentREST) TestListCommentsByParentWorkItemNotModifiedUsingIfNoneMatchHeader() {
	// given
	wi, comments := rest.setupComments()
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 3
	ifNoneMatch := app.GenerateEntitiesTag([]app.ConditionalRequestEntity{
		comments[2],
		comments[1],
		comments[0],
	})
	res := test.ListWorkItemCommentsNotModified(rest.T(), svc.Context, svc, ctrl, wi.ID, &limit, &offset, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestCommentREST) TestEmptyListCommentsByParentWorkItem() {
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wi := fxt.WorkItems[0]
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 1
	_, cs := test.ListWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, &limit, &offset, nil, nil)
	// then
	assert.Equal(rest.T(), 0, len(cs.Data))
}

func (rest *TestCommentREST) TestCreateSingleCommentMissingWorkItem() {
	// given
	p := rest.newCreateWorkItemCommentsPayload("Test", nil)
	// when/then
	svc, ctrl := rest.SecuredController()
	test.CreateWorkItemCommentsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4(), p)
}

func (rest *TestCommentREST) TestCreateSingleNoAuthorized() {
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wi := fxt.WorkItems[0]
	// when/then
	p := rest.newCreateWorkItemCommentsPayload("Test", nil)
	svc, ctrl := rest.UnSecuredController()
	test.CreateWorkItemCommentsUnauthorized(rest.T(), svc.Context, svc, ctrl, wi.ID, p)
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
	require.Error(rest.T(), err)
}

func (rest *TestCommentREST) TestListCommentsByMissingParentWorkItem() {
	// given
	svc, ctrl := rest.SecuredController()
	// when/then
	offset := "0"
	limit := 1
	test.ListWorkItemCommentsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4(), &limit, &offset, nil, nil)
}
