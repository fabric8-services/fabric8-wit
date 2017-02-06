package main_test

import (
	"fmt"
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
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
	suite.Run(t, &CommentsSuite{DBTestSuite: gormsupport.NewDBTestSuite("config.yaml")})
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

func (s *CommentsSuite) unsecuredController() (*goa.Service, *CommentsController) {
	svc := goa.New("Comments-service-test")
	commentsCtrl := NewCommentsController(svc, s.db)
	return svc, commentsCtrl
}

func (s *CommentsSuite) securedControllers(identity account.Identity) (*goa.Service, *WorkitemController, *WorkItemCommentsController, *CommentsController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("Comment-Service", almtoken.NewManager(pub, priv), identity)
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
				workitem.SystemTitle: "Title",
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

// createWorkItemComment creates a workitem comment that will be used to perform the comment operations during the tests.
func (s *CommentsSuite) createWorkItemComment(identity account.Identity, workitemId string) uuid.UUID {
	createWorkItemCommentPayload := app.CreateWorkItemCommentsPayload{
		Data: &app.CreateComment{
			Type: "comments",
			Attributes: &app.CreateCommentAttributes{
				Body: "a comment",
			},
		},
	}
	userSvc, _, workitemCommentsCtrl, _ := s.securedControllers(identity)
	_, comment := test.CreateWorkItemCommentsOK(s.T(), userSvc.Context, userSvc, workitemCommentsCtrl, workitemId,
		&createWorkItemCommentPayload)
	commentId := *comment.Data.ID
	s.T().Log(fmt.Sprintf("Created comment with id %v", commentId))
	return commentId
}

func (s *CommentsSuite) TestShowCommentWithoutAuth() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId)
	// when
	userSvc, commentsCtrl := s.unsecuredController()
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId)
	// then
	require.NotNil(s.T(), result)
	require.NotNil(s.T(), result.Data)
	assert.NotNil(s.T(), result.Data.ID)
	assert.NotNil(s.T(), result.Data.Type)
	assert.Equal(s.T(), "a comment", *result.Data.Attributes.Body)
	assert.Equal(s.T(), testsupport.TestIdentity.ID, *result.Data.Relationships.CreatedBy.Data.ID)
}

func (s *CommentsSuite) TestShowCommentWithAuth() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId)
	// when
	userSvc, _, _, commentsCtrl := s.securedControllers(testsupport.TestIdentity)
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId)
	// then
	require.NotNil(s.T(), result)
	require.NotNil(s.T(), result.Data)
	assert.NotNil(s.T(), result.Data.ID)
	assert.NotNil(s.T(), result.Data.Type)
	assert.Equal(s.T(), "a comment", *result.Data.Attributes.Body)
	assert.Equal(s.T(), testsupport.TestIdentity.ID, *result.Data.Relationships.CreatedBy.Data.ID)
}

func (s *CommentsSuite) TestUpdateCommentWithoutAuth() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId)
	// when
	updatedCommentBody := "An updated comment"
	updateCommentPayload := app.UpdateCommentsPayload{
		Data: &app.Comment{
			Type: "comments",
			Attributes: &app.CommentAttributes{
				Body: &updatedCommentBody,
			},
		},
	}
	userSvc, commentsCtrl := s.unsecuredController()
	test.UpdateCommentsUnauthorized(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId, &updateCommentPayload)
}

func (s *CommentsSuite) TestUpdateCommentWithSameAuthenticatedUser() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId)
	// when
	updatedCommentBody := "An updated comment"
	updateCommentPayload := app.UpdateCommentsPayload{
		Data: &app.Comment{
			Type: "comments",
			Attributes: &app.CommentAttributes{
				Body: &updatedCommentBody,
			},
		},
	}
	userSvc, _, _, commentsCtrl := s.securedControllers(testsupport.TestIdentity)
	test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId, &updateCommentPayload)
}

func (s *CommentsSuite) TestUpdateCommentWithOtherAuthenticatedUser() {
	// given
	workitemId := s.createWorkItem(testsupport.TestIdentity)
	commentId := s.createWorkItemComment(testsupport.TestIdentity, workitemId)
	// when
	updatedCommentBody := "An updated comment"
	updateCommentPayload := app.UpdateCommentsPayload{
		Data: &app.Comment{
			Type: "comments",
			Attributes: &app.CommentAttributes{
				Body: &updatedCommentBody,
			},
		},
	}
	userSvc, _, _, commentsCtrl := s.securedControllers(testsupport.TestIdentity2)
	test.UpdateCommentsUnauthorized(s.T(), userSvc.Context, userSvc, commentsCtrl, commentId, &updateCommentPayload)
}
