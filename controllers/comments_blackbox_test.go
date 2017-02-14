package controllers

import (
	"fmt"
	"html"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// a normal test function that will kick off TestSuiteComments
func TestSuiteComments(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &CommentsSuite{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

// ========== TestSuiteComments struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
type CommentsSuite struct {
	gormsupport.DBTestSuite
	db    *gormapplication.GormDB
	clean func()
}

func (s *CommentsSuite) SetupTest() {
	s.db = gormapplication.NewGormDB(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *CommentsSuite) TearDownTest() {
	s.clean()
}

var (
	markdownMarkup  = rendering.SystemMarkupMarkdown
	plaintextMarkup = rendering.SystemMarkupPlainText
	defaultMarkup   = rendering.SystemMarkupDefault
)

func (s *CommentsSuite) unsecuredController() (*goa.Service, *CommentsController) {
	svc := goa.New("Comments-service-test")
	commentsCtrl := NewCommentsController(svc, s.db)
	return svc, commentsCtrl
}

func (s *CommentsSuite) securedControllers(identity account.Identity) (*goa.Service, *WorkitemController, *WorkItemCommentsController, *CommentsController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("Comment-Service", almtoken.NewManagerWithPrivateKey(priv), identity)
	workitemCtrl := NewWorkitemController(svc, s.db)
	workitemCommentsCtrl := NewWorkItemCommentsController(svc, s.db)
	commentsCtrl := NewCommentsController(svc, s.db)
	return svc, workitemCtrl, workitemCommentsCtrl, commentsCtrl
}

// createWorkItem creates a workitem that will be used to perform the comment operations during the tests.
func (s *CommentsSuite) createWorkItem(identity account.Identity) string {
	createWorkitemPayload := app.CreateWorkitemPayload{
		Data: &app.WorkItem2{
			Type: APIStringTypeWorkItem,
			Attributes: map[string]interface{}{
				workitem.SystemTitle: "work item title",
				workitem.SystemState: workitem.SystemStateNew},
			Relationships: &app.WorkItemRelationships{
				BaseType: &app.RelationBaseType{
					Data: &app.BaseTypeData{
						Type: "workitemtypes",
						ID:   workitem.SystemBug,
					},
				},
			},
		},
	}
	userSvc, workitemCtrl, _, _ := s.securedControllers(identity)
	_, wi := test.CreateWorkitemCreated(s.T(), userSvc.Context, userSvc, workitemCtrl, &createWorkitemPayload)
	workitemId := *wi.Data.ID
	s.T().Log(fmt.Sprintf("Created workitem with id %v", workitemId))
	return workitemId
}

func (s *CommentsSuite) newCreateWorkItemCommentsPayload(body string, markup *string) *app.CreateWorkItemCommentsPayload {
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

func (s *CommentsSuite) newUpdateCommentsPayload(body string, markup *string) *app.UpdateCommentsPayload {
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

// createWorkItemComment creates a workitem comment that will be used to perform the comment operations during the tests.
func (s *CommentsSuite) createWorkItemComment(identity account.Identity, workitemId string, body string, markup *string) uuid.UUID {
	createWorkItemCommentPayload := s.newCreateWorkItemCommentsPayload(body, markup)
	userSvc, _, workitemCommentsCtrl, _ := s.securedControllers(identity)
	_, comment := test.CreateWorkItemCommentsOK(s.T(), userSvc.Context, userSvc, workitemCommentsCtrl, workitemId, createWorkItemCommentPayload)
	commentId := *comment.Data.ID
	s.T().Log(fmt.Sprintf("Created comment with id %v", commentId))
	return commentId
}

func (s *CommentsSuite) validateComment(result *app.CommentSingle, expectedBody string, expectedMarkup string) {
	require.NotNil(s.T(), result)
	require.NotNil(s.T(), result.Data)
	assert.NotNil(s.T(), result.Data.ID)
	assert.NotNil(s.T(), result.Data.Type)
	require.NotNil(s.T(), result.Data.Attributes)
	require.NotNil(s.T(), result.Data.Attributes.Body)
	assert.Equal(s.T(), expectedBody, *result.Data.Attributes.Body)
	require.NotNil(s.T(), result.Data.Attributes.Markup)
	assert.Equal(s.T(), expectedMarkup, *result.Data.Attributes.Markup)
	assert.Equal(s.T(), rendering.RenderMarkupToHTML(html.EscapeString(expectedBody), expectedMarkup), *result.Data.Attributes.BodyRendered)
	require.NotNil(s.T(), result.Data.Relationships)
	require.NotNil(s.T(), result.Data.Relationships.CreatedBy)
	require.NotNil(s.T(), result.Data.Relationships.CreatedBy.Data)
	require.NotNil(s.T(), result.Data.Relationships.CreatedBy.Data.ID)
	assert.Equal(s.T(), testsupport.TestIdentity.ID, *result.Data.Relationships.CreatedBy.Data.ID)
}

func (s *CommentsSuite) TestShowCommentWithoutAuth() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId, "body", &markdownMarkup)
	// when
	userSvc, commentsCtrl := s.unsecuredController()
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId)
	// then
	s.validateComment(result, "body", rendering.SystemMarkupMarkdown)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthWithMarkup() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId, "body", nil)
	// when
	userSvc, commentsCtrl := s.unsecuredController()
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId)
	// then
	s.validateComment(result, "body", rendering.SystemMarkupPlainText)
}

func (s *CommentsSuite) TestShowCommentWithAuth() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId, "body", &plaintextMarkup)
	// when
	userSvc, _, _, commentsCtrl := s.securedControllers(testsupport.TestIdentity)
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId)
	// then
	s.validateComment(result, "body", rendering.SystemMarkupPlainText)
}

func (s *CommentsSuite) TestShowCommentWithEscapedScriptInjection() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId, "<img src=x onerror=alert('body') />", &plaintextMarkup)
	// when
	userSvc, _, _, commentsCtrl := s.securedControllers(testsupport.TestIdentity)
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId)
	// then
	s.validateComment(result, "<img src=x onerror=alert('body') />", rendering.SystemMarkupPlainText)
}

func (s *CommentsSuite) TestUpdateCommentWithoutAuth() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId, "body", &plaintextMarkup)
	// when
	updateCommentPayload := s.newUpdateCommentsPayload("updated body", &markdownMarkup)
	userSvc, commentsCtrl := s.unsecuredController()
	test.UpdateCommentsUnauthorized(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId, updateCommentPayload)
}

func (s *CommentsSuite) TestUpdateCommentWithSameUserWithOtherMarkup() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId, "body", &plaintextMarkup)
	// when
	updateCommentPayload := s.newUpdateCommentsPayload("updated body", &markdownMarkup)
	userSvc, _, _, commentsCtrl := s.securedControllers(testsupport.TestIdentity)
	_, result := test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId, updateCommentPayload)
	s.validateComment(result, "updated body", rendering.SystemMarkupMarkdown)
}

func (s *CommentsSuite) TestUpdateCommentWithSameUserWithNilMarkup() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId, "body", &plaintextMarkup)
	// when
	updateCommentPayload := s.newUpdateCommentsPayload("updated body", nil)
	userSvc, _, _, commentsCtrl := s.securedControllers(testsupport.TestIdentity)
	_, result := test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId, updateCommentPayload)
	s.validateComment(result, "updated body", rendering.SystemMarkupDefault)
}

func (s *CommentsSuite) TestUpdateCommentWithOtherUser() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId, "body", &plaintextMarkup)
	// when
	updatedCommentBody := "An updated comment"
	updateCommentPayload := &app.UpdateCommentsPayload{
		Data: &app.Comment{
			Type: "comments",
			Attributes: &app.CommentAttributes{
				Body: &updatedCommentBody,
			},
		},
	}
	// when/then
	userSvc, _, _, commentsCtrl := s.securedControllers(testsupport.TestIdentity2)
	test.UpdateCommentsForbidden(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId, updateCommentPayload)
}

func (s *CommentsSuite) TestDeleteCommentWithSameAuthenticatedUser() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId, "body", &plaintextMarkup)
	userSvc, _, _, commentsCtrl := s.securedControllers(testsupport.TestIdentity)
	test.DeleteCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId)
}

func (s *CommentsSuite) TestDeleteCommentWithoutAuth() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId, "body", &plaintextMarkup)
	userSvc, commentsCtrl := s.unsecuredController()
	test.DeleteCommentsUnauthorized(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId)
}

func (s *CommentsSuite) TestDeleteCommentWithOtherAuthenticatedUser() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId, "body", &plaintextMarkup)
	userSvc, _, _, commentsCtrl := s.securedControllers(testsupport.TestIdentity2)
	test.DeleteCommentsForbidden(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId)
}
