package controller_test

import (
	"context"
	"fmt"
	"html"
	"strings"
	"testing"
	"time"

	token "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/comment"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	notificationsupport "github.com/fabric8-services/fabric8-wit/test/notification"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// a normal test function that will kick off TestSuiteComments
func TestSuiteComments(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &CommentsSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

// ========== TestSuiteComments struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
type CommentsSuite struct {
	gormtestsupport.DBTestSuite
	testIdentity  account.Identity
	testIdentity2 account.Identity
	notification  notificationsupport.FakeNotificationChannel
}

func (s *CommentsSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "CommentsSuite user", "test provider")
	require.NoError(s.T(), err)
	s.testIdentity = *testIdentity
	testIdentity2, err := testsupport.CreateTestIdentity(s.DB, "CommentsSuite user2", "test provider")
	require.NoError(s.T(), err)
	s.testIdentity2 = *testIdentity2
	s.notification = notificationsupport.FakeNotificationChannel{}
}

var (
	markdownMarkup  = rendering.SystemMarkupMarkdown
	plaintextMarkup = rendering.SystemMarkupPlainText
	defaultMarkup   = rendering.SystemMarkupDefault
)

func (s *CommentsSuite) unsecuredController() (*goa.Service, *CommentsController) {
	svc := goa.New("Comments-service-test")
	commentsCtrl := NewNotifyingCommentsController(svc, s.GormDB, &s.notification, s.Configuration)
	return svc, commentsCtrl
}

func (s *CommentsSuite) securedControllers(identity account.Identity) (*goa.Service, *WorkitemController, *WorkitemsController, *WorkItemCommentsController, *CommentsController) {
	svc := testsupport.ServiceAsUser("Comment-Service", identity)
	workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
	workitemsCtrl := NewWorkitemsController(svc, s.GormDB, s.Configuration)
	workitemCommentsCtrl := NewWorkItemCommentsController(svc, s.GormDB, s.Configuration)

	commentsCtrl := NewNotifyingCommentsController(svc, s.GormDB, &s.notification, s.Configuration)
	return svc, workitemCtrl, workitemsCtrl, workitemCommentsCtrl, commentsCtrl
}

func newCreateWorkItemCommentsPayload(body string, markup *string, parentCommentID *uuid.UUID) *app.CreateWorkItemCommentsPayload {
	tempPayload := app.CreateWorkItemCommentsPayload{
		Data: &app.CreateComment{
			Type: "comments",
			Attributes: &app.CreateCommentAttributes{
				Body:   body,
				Markup: markup,
			},
		},
	}
	if parentCommentID != nil {
		tempPayload.Data.Relationships = &app.CreateCommentRelations{
			ParentComment: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: ptr.String("comments"),
					ID:   ptr.String(parentCommentID.String()),
				},
			},
		}
	}
	return &tempPayload
}

// createWorkItemComment creates a workitem comment that will be used to perform the comment operations during the tests.
func (s *CommentsSuite) createWorkItemComment(identity account.Identity, wiID uuid.UUID, body string, markup *string, parentCommentID *uuid.UUID) app.CommentSingle {
	createWorkItemCommentPayload := newCreateWorkItemCommentsPayload(body, markup, parentCommentID)
	userSvc, _, _, workitemCommentsCtrl, _ := s.securedControllers(identity)
	_, c := test.CreateWorkItemCommentsOK(s.T(), userSvc.Context, userSvc, workitemCommentsCtrl, wiID, createWorkItemCommentPayload)
	require.NotNil(s.T(), c)
	return *c
}

func newUpdateCommentsPayload(body string, markup *string) *app.UpdateCommentsPayload {
	return &app.UpdateCommentsPayload{
		Data: &app.Comment{
			Type: "comments",
			Attributes: &app.CommentAttributes{
				Body:   &body,
				Markup: markup,
			},
		},
	}
}

// updateComment updates the comment with the given commentId
func (s *CommentsSuite) updateComment(identity account.Identity, commentID uuid.UUID, body string, markup *string) app.CommentSingle {
	updateCommentsPayload := newUpdateCommentsPayload(body, markup)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(identity)
	_, c := test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentID, updateCommentsPayload)
	require.NotNil(s.T(), c)
	s.T().Log(fmt.Sprintf("Updated comment with id %v", *c.Data.ID))
	return *c
}

// deleteComment deletes the comment with the given commentId
func (s *CommentsSuite) deleteComment(identity account.Identity, commentID uuid.UUID) {
	userSvc, _, _, _, commentsCtrl := s.securedControllers(identity)
	test.DeleteCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentID)
	s.T().Log(fmt.Sprintf("Deleted comment with id %v", commentID))
}

func assertComment(t *testing.T, resultData *app.Comment, expectedIdentity account.Identity, expectedBody string, expectedMarkup string) {
	require.NotNil(t, resultData)
	assert.NotNil(t, resultData.ID)
	assert.NotNil(t, resultData.Type)
	require.NotNil(t, resultData.Attributes)
	require.NotNil(t, resultData.Attributes.CreatedAt)
	require.NotNil(t, resultData.Attributes.UpdatedAt)
	require.NotNil(t, resultData.Attributes.Body)
	require.NotNil(t, resultData.Attributes.Body)
	assert.Equal(t, expectedBody, *resultData.Attributes.Body)
	require.NotNil(t, resultData.Attributes.Markup)
	assert.Equal(t, expectedMarkup, *resultData.Attributes.Markup)
	assert.Equal(t, rendering.RenderMarkupToHTML(html.EscapeString(expectedBody), expectedMarkup), *resultData.Attributes.BodyRendered)
	require.NotNil(t, resultData.Relationships)
	require.NotNil(t, resultData.Relationships.Creator)
	require.NotNil(t, resultData.Relationships.Creator.Data)
	require.NotNil(t, resultData.Relationships.Creator.Data.ID)
	assert.Equal(t, expectedIdentity.ID, uuid.FromStringOrNil(*resultData.Relationships.Creator.Data.ID))
	assert.True(t, strings.Contains(*resultData.Relationships.Creator.Data.ID, *resultData.Relationships.Creator.Data.ID), "Link not found")
}

func ConvertCommentToModel(c app.CommentSingle) comment.Comment {
	return comment.Comment{
		ID: *c.Data.ID,
		Lifecycle: gormsupport.Lifecycle{
			UpdatedAt: *c.Data.Attributes.UpdatedAt,
		},
	}
}

func (s *CommentsSuite) TestShowCommentWithParentComment() {
	// create parent comment
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	p := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup, nil)
	// create child comment
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup, p.Data.ID)
	// when
	userSvc, commentsCtrl := s.unsecuredController()
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assert.Equal(s.T(), p.Data.ID.String(), *result.Data.Relationships.ParentComment.Data.ID)
}

func (s *CommentsSuite) TestShowCommentWithBackwardSupport() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup, nil)
	// when
	userSvc, commentsCtrl := s.unsecuredController()
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assert.Equal(s.T(), *result.Data.Relationships.Creator.Data.ID, result.Data.Relationships.CreatedBy.Data.ID.String())
}

func (s *CommentsSuite) TestShowCommentWithoutAuthOK() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup, nil)
	// when
	userSvc, commentsCtrl := s.unsecuredController()
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupMarkdown)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthOKUsingExpiredIfModifiedSinceHeader() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup, nil)
	// when
	ifModifiedSince := app.ToHTTPTime(c.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
	userSvc, commentsCtrl := s.unsecuredController()
	res, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, &ifModifiedSince, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupMarkdown)
	assertResponseHeaders(s.T(), res)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthOKUsingExpiredIfNoneMatchHeader() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup, nil)
	// when
	ifNoneMatch := "foo"
	userSvc, commentsCtrl := s.unsecuredController()
	res, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, &ifNoneMatch)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupMarkdown)
	assertResponseHeaders(s.T(), res)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthNotModifiedUsingIfModifiedSinceHeader() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup, nil)
	// when
	ifModifiedSince := app.ToHTTPTime(*c.Data.Attributes.UpdatedAt)
	userSvc, commentsCtrl := s.unsecuredController()
	res := test.ShowCommentsNotModified(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(s.T(), res)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthNotModifiedUsingIfNoneMatchHeader() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup, nil)
	// when
	commentModel := ConvertCommentToModel(c)
	ifNoneMatch := app.GenerateEntityTag(commentModel)
	userSvc, commentsCtrl := s.unsecuredController()
	res := test.ShowCommentsNotModified(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(s.T(), res)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthWithMarkup() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", nil, nil)
	// when
	userSvc, commentsCtrl := s.unsecuredController()
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupPlainText)
}

func (s *CommentsSuite) TestShowCommentWithAuth() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup, nil)
	// when
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupPlainText)
}

func (s *CommentsSuite) TestShowCommentWithEscapedScriptInjection() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "<img src=x onerror=alert('body') />", &plaintextMarkup, nil)
	// when
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "<img src=x onerror=alert('body') />", rendering.SystemMarkupPlainText)
}

func (s *CommentsSuite) TestUpdateCommentWithoutAuth() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup, nil)
	// when
	updateCommentPayload := newUpdateCommentsPayload("updated body", &markdownMarkup)
	userSvc, commentsCtrl := s.unsecuredController()
	test.UpdateCommentsUnauthorized(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
}

func (s *CommentsSuite) TestUpdateCommentWithSameUserWithOtherMarkup() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup, nil)
	// when
	updateCommentPayload := newUpdateCommentsPayload("updated body", &markdownMarkup)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	_, result := test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, s.testIdentity, "updated body", rendering.SystemMarkupMarkdown)
}

func (s *CommentsSuite) TestUpdateCommentWithSameUserWithNilMarkup() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup, nil)
	// when
	updateCommentPayload := newUpdateCommentsPayload("updated body", nil)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	_, result := test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, s.testIdentity, "updated body", rendering.SystemMarkupDefault)
}

func (s *CommentsSuite) TestDeleteCommentWithSameAuthenticatedUser() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup, nil)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	// when/then
	test.DeleteCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID)
}

func (s *CommentsSuite) TestDeleteCommentWithoutAuth() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup, nil)
	userSvc, commentsCtrl := s.unsecuredController()
	// when/then
	test.DeleteCommentsUnauthorized(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID)
}

// Following test creates a space and space_owner creates a WI in that space
// Space owner adds a comment on the created WI
// Create another user, which is not space collaborator.
// Test if another user can delete the comment
func (s *CommentsSuite) TestNonCollaboratorCanNotDelete() {
	// create space
	// create user
	// add user to the space collaborator list
	// create workitem in created space
	// create another user - do not add this user into collaborator list
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestNonCollaboraterCanNotDelete-"), "TestWIComments")
	require.NoError(s.T(), err)
	space := CreateSecuredSpace(s.T(), s.GormDB, s.Configuration, *testIdentity, "")

	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemFeature, *space.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", *testIdentity, &TestSpaceAuthzService{*testIdentity, ""})
	workitemsCtrl := NewWorkitemsController(svc, s.GormDB, s.Configuration)

	_, wi := test.CreateWorkitemsCreated(s.T(), svc.Context, svc, workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	c := s.createWorkItemComment(*testIdentity, *wi.Data.ID, "body", &plaintextMarkup, nil)

	testIdentity2, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestNonCollaboraterCanNotDelete-"), "TestWI")
	svcNotAuthorized := testsupport.ServiceAsSpaceUser("Collaborators-Service", *testIdentity2, &TestSpaceAuthzService{*testIdentity, ""})
	commentsCtrlNotAuthorized := NewCommentsController(svcNotAuthorized, s.GormDB, s.Configuration)

	test.DeleteCommentsForbidden(s.T(), svcNotAuthorized.Context, svcNotAuthorized, commentsCtrlNotAuthorized, *c.Data.ID)
}

func (s *CommentsSuite) TestCollaboratorCanDelete() {
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestCollaboratorCanDelete-"), "TestWIComments")
	require.NoError(s.T(), err)
	space := CreateSecuredSpace(s.T(), s.GormDB, s.Configuration, *testIdentity, "")

	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemFeature, *space.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	svc, _, workitemsCtrl, _, _ := s.securedControllers(*testIdentity)
	// svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", testIdentity, &TestSpaceAuthzService{testIdentity})
	// ctrl := NewWorkitemsController(svc, s.GormDB, s.Configuration)

	_, wi := test.CreateWorkitemsCreated(s.T(), svc.Context, svc, workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	c := s.createWorkItemComment(*testIdentity, *wi.Data.ID, "body", &plaintextMarkup, nil)
	commentCtrl := NewCommentsController(svc, s.GormDB, s.Configuration)
	test.DeleteCommentsOK(s.T(), svc.Context, svc, commentCtrl, *c.Data.ID)
}

func (s *CommentsSuite) TestCreatorCanDelete() {
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wID, "body", &plaintextMarkup, nil)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	test.DeleteCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID)
}

func (s *CommentsSuite) TestOtherCollaboratorCanDelete() {
	// create space owner identity
	spaceOwner, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestOtherCollaboratorCanDelete-"), "TestWIComments")
	require.NoError(s.T(), err)

	// create 2 space collaborators' identity
	collaborator1, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestOtherCollaboratorCanDelete-"), "TestWIComments")
	require.NoError(s.T(), err)

	collaborator2, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestOtherCollaboratorCanDelete-"), "TestWIComments")
	require.NoError(s.T(), err)

	// Add 2 identities as Collaborators
	space := CreateSecuredSpace(s.T(), s.GormDB, s.Configuration, *spaceOwner, fmt.Sprintf("%s,%s", collaborator1.ID.String(), collaborator2.ID.String()))
	svcWithSpaceOwner := testsupport.ServiceAsSpaceUser("Comments-Service", *spaceOwner, &TestSpaceAuthzService{*spaceOwner, fmt.Sprintf("%s,%s", collaborator1.ID.String(), collaborator2.ID.String())})

	// Build WI payload and create 1 WI (created_by = space owner)
	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemFeature, *space.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	workitemsCtrl := NewWorkitemsController(svcWithSpaceOwner, s.GormDB, s.Configuration)

	_, wi := test.CreateWorkitemsCreated(s.T(), svcWithSpaceOwner.Context, svcWithSpaceOwner, workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)

	// collaborator1 adds a comment on newly created work item
	c := s.createWorkItemComment(*collaborator1, *wi.Data.ID, "Hello woody", &plaintextMarkup, nil)

	// Collaborator2 deletes the comment
	svcWithCollaborator2 := testsupport.ServiceAsSpaceUser("Comments-Service", *collaborator2, &TestSpaceAuthzService{*collaborator2, ""})
	commentCtrl := NewCommentsController(svcWithCollaborator2, s.GormDB, s.Configuration)
	test.DeleteCommentsOK(s.T(), svcWithCollaborator2.Context, svcWithCollaborator2, commentCtrl, *c.Data.ID)
}

// Following test creates a space and space_owner creates a WI in that space
// Space owner adds a comment on the created WI
// Create another user, which is not space collaborator.
// Test if another user can edit/update the comment
func (s *CommentsSuite) TestNonCollaboratorCanNotUpdate() {
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestNonCollaboraterCanNotUpdate-"), "TestWIComments")
	require.NoError(s.T(), err)
	space := CreateSecuredSpace(s.T(), s.GormDB, s.Configuration, *testIdentity, "")

	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemFeature, *space.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI 2"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", *testIdentity, &TestSpaceAuthzService{*testIdentity, ""})
	workitemsCtrl := NewWorkitemsController(svc, s.GormDB, s.Configuration)

	_, wi := test.CreateWorkitemsCreated(s.T(), svc.Context, svc, workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	c := s.createWorkItemComment(*testIdentity, *wi.Data.ID, "body", &plaintextMarkup, nil)

	testIdentity2, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestNonCollaboraterCanNotUpdate-"), "TestWI")
	svcNotAuthorized := testsupport.ServiceAsSpaceUser("Collaborators-Service", *testIdentity2, &TestSpaceAuthzService{*testIdentity, ""})
	commentCtrlNotAuthorized := NewCommentsController(svcNotAuthorized, s.GormDB, s.Configuration)

	updateCommentPayload := newUpdateCommentsPayload("updated body", &markdownMarkup)
	test.UpdateCommentsForbidden(s.T(), svcNotAuthorized.Context, svcNotAuthorized, commentCtrlNotAuthorized, *c.Data.ID, updateCommentPayload)
}

func (s *CommentsSuite) TestCollaboratorCanUpdate() {
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestCollaboratorCanUpdate-"), "TestWIComments")
	require.NoError(s.T(), err)
	space := CreateSecuredSpace(s.T(), s.GormDB, s.Configuration, *testIdentity, "")

	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemFeature, *space.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	// svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", testIdentity, &TestSpaceAuthzService{testIdentity})
	// ctrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
	svc, _, workitemsCtrl, _, _ := s.securedControllers(*testIdentity)

	_, wi := test.CreateWorkitemsCreated(s.T(), svc.Context, svc, workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	c := s.createWorkItemComment(*testIdentity, *wi.Data.ID, "body", &plaintextMarkup, nil)
	commentCtrl := NewCommentsController(svc, s.GormDB, s.Configuration)

	updatedBody := "I am updated comment"
	updateCommentPayload := newUpdateCommentsPayload(updatedBody, &markdownMarkup)
	_, result := test.UpdateCommentsOK(s.T(), svc.Context, svc, commentCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, *testIdentity, updatedBody, markdownMarkup)
}

func (s *CommentsSuite) TestCreatorCanUpdate() {
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wID, "Hello world", &plaintextMarkup, nil)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)

	updatedBody := "Hello world in golang"
	updateCommentPayload := newUpdateCommentsPayload(updatedBody, &markdownMarkup)
	_, result := test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, s.testIdentity, updatedBody, markdownMarkup)
}

func (s *CommentsSuite) TestOtherCollaboratorCanUpdate() {
	// create space owner identity
	spaceOwner, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestOtherCollaboratorCanUpdate-"), "TestWIComments")
	require.NoError(s.T(), err)

	// create 2 space collaborators' identity
	collaborator1, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestOtherCollaboratorCanUpdate-"), "TestWIComments")
	require.NoError(s.T(), err)

	collaborator2, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestOtherCollaboratorCanUpdate-"), "TestWIComments")
	require.NoError(s.T(), err)

	// Add 2 Collaborators in space
	space := CreateSecuredSpace(s.T(), s.GormDB, s.Configuration, *spaceOwner, "")
	svcWithSpaceOwner := testsupport.ServiceAsSpaceUser("Comments-Service", *spaceOwner, &TestSpaceAuthzService{*spaceOwner, fmt.Sprintf("%s,%s", collaborator1.ID.String(), collaborator2.ID.String())})

	// Build WI payload and create 1 WI (created_by = space owner)
	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemFeature, *space.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	workitemsCtrl := NewWorkitemsController(svcWithSpaceOwner, s.GormDB, s.Configuration)

	_, wi := test.CreateWorkitemsCreated(s.T(), svcWithSpaceOwner.Context, svcWithSpaceOwner, workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)

	// collaborator1 adds a comment on newly created work item
	c := s.createWorkItemComment(*collaborator1, *wi.Data.ID, "Hello woody", &plaintextMarkup, nil)

	// update comment by collaborator 1
	updatedBody := "Another update on same comment"
	updateCommentPayload := newUpdateCommentsPayload(updatedBody, &markdownMarkup)
	svcWithCollaborator1 := testsupport.ServiceAsSpaceUser("Comments-Service", *collaborator1, &TestSpaceAuthzService{*collaborator1, ""})
	commentCtrl := NewCommentsController(svcWithCollaborator1, s.GormDB, s.Configuration)
	_, result := test.UpdateCommentsOK(s.T(), svcWithCollaborator1.Context, svcWithCollaborator1, commentCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, *collaborator1, updatedBody, markdownMarkup)

	// update comment by collaborator2
	updatedBody = "Modified body of comment"
	updateCommentPayload = newUpdateCommentsPayload(updatedBody, &markdownMarkup)
	svcWithCollaborator2 := testsupport.ServiceAsSpaceUser("Comments-Service", *collaborator2, &TestSpaceAuthzService{*collaborator2, ""})
	commentCtrl = NewCommentsController(svcWithCollaborator2, s.GormDB, s.Configuration)
	_, result = test.UpdateCommentsOK(s.T(), svcWithCollaborator2.Context, svcWithCollaborator2, commentCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, *collaborator1, updatedBody, markdownMarkup)
}

func (s *CommentsSuite) TestNotificationSendOnUpdate() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	wiID := fxt.WorkItems[0].ID
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup, nil)
	// when
	updateCommentPayload := newUpdateCommentsPayload("updated body", &markdownMarkup)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
	assert.True(s.T(), len(s.notification.Messages) > 0)
	assert.Equal(s.T(), "comment.update", s.notification.Messages[0].MessageType)
	assert.Equal(s.T(), c.Data.ID.String(), s.notification.Messages[0].TargetID)
}

func CreateSecuredSpace(t *testing.T, db application.DB, config SpaceConfiguration, owner account.Identity, userIDs string) app.Space {
	svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", owner, &TestSpaceAuthzService{owner: owner, userIDs: userIDs})
	spaceCtrl := NewSpaceController(svc, db, config, &DummyResourceManager{})
	require.NotNil(t, spaceCtrl)
	spacePayload := &app.CreateSpacePayload{
		Data: &app.Space{
			Type: "spaces",
			Attributes: &app.SpaceAttributes{
				Name:        ptr.String("TestCollaborators-space-" + uuid.NewV4().String()),
				Description: ptr.String("description"),
			},
		},
	}
	_, sp := test.CreateSpaceCreated(t, svc.Context, svc, spaceCtrl, spacePayload)
	require.NotNil(t, sp)
	require.NotNil(t, sp.Data)
	return *sp.Data
}

type TestSpaceAuthzService struct {
	owner   account.Identity
	userIDs string
}

func (s *TestSpaceAuthzService) Authorize(ctx context.Context, spaceID string) (bool, error) {
	jwtToken := goajwt.ContextJWT(ctx)
	if jwtToken == nil {
		return false, errors.NewUnauthorizedError("Missing token")
	}
	id := jwtToken.Claims.(token.MapClaims)["sub"].(string)
	return s.owner.ID.String() == id || strings.Contains(s.userIDs, id), nil
}

func (s *TestSpaceAuthzService) Configuration() auth.ServiceConfiguration {
	return nil
}
