package main_test

import (
	"crypto/rsa"
	"fmt"
	"testing"

	"golang.org/x/net/context"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// a normal test function that will kick off TestSuiteComments
func TestSuiteComments(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(CommentsSuite))
}

// ========== TestSuiteComments struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
type CommentsSuite struct {
	suite.Suite
	workitemCommentsCtrl app.WorkItemCommentsController
	workitemCtrl         app.WorkitemController
	defaultSvc           *goa.Service
	userSvc              *goa.Service
	userSvc2             *goa.Service
	db                   *gorm.DB
	pubKey               *rsa.PublicKey
	priKey               *rsa.PrivateKey
	clean                func()
	commentId            uuid.UUID
}

func (s *CommentsSuite) SetupSuite() {
	var err error
	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	s.db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())
	if err != nil {
		panic("Failed to connect database: " + err.Error())
	}
	// default service (without auth)
	s.defaultSvc = goa.New("Comments-service-test")
	// user service (with auth)
	s.pubKey, _ = almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	s.priKey, _ = almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	s.userSvc = testsupport.ServiceAsUser("CommentsSuite-Service", almtoken.NewManager(s.pubKey, s.priKey), account.TestIdentity)
	s.userSvc2 = testsupport.ServiceAsUser("CommentsSuite-Service", almtoken.NewManager(s.pubKey, s.priKey), account.TestIdentity2)
	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if configuration.GetPopulateCommonTypes() {
		if err := models.Transactional(s.db, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
	s.clean = cleaner.DeleteCreatedEntities(s.db)
	s.workitemCtrl = NewWorkitemController(s.userSvc, gormapplication.NewGormDB(s.db))
	s.workitemCommentsCtrl = NewWorkItemCommentsController(s.userSvc, gormapplication.NewGormDB(s.db))
}

func (s *CommentsSuite) TearDownSuite() {
	s.clean()
	if s.db != nil {
		s.db.Close()
	}
}

func (s *CommentsSuite) SetupTest() {
	// create the workitem that will be used to perform the comment operations during the tests.
	// given
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
	_, wi := test.CreateWorkitemCreated(s.T(), s.userSvc.Context, s.userSvc, s.workitemCtrl, &createWorkitemPayload)
	workitemId := *wi.Data.ID
	// now, create a comment for the workitem
	createWorkItemCommentPayload := app.CreateWorkItemCommentsPayload{
		Data: &app.CreateComment{
			Type: "comments",
			Attributes: &app.CreateCommentAttributes{
				Body: "a comment",
			},
		},
	}
	_, comment := test.CreateWorkItemCommentsOK(s.T(), s.userSvc.Context, s.userSvc, s.workitemCommentsCtrl, workitemId,
		&createWorkItemCommentPayload)
	s.commentId = *comment.Data.ID
	s.T().Log(fmt.Sprintf("Created comment with id %v", s.commentId))
}

func (s *CommentsSuite) TearDownTest() {
}

func (s *CommentsSuite) TestShowCommentWithoutAuth() {
	// given
	commentsCtrl := NewCommentsController(s.defaultSvc, gormapplication.NewGormDB(s.db))
	// when
	_, result := test.ShowCommentsOK(s.T(), s.defaultSvc.Context, s.defaultSvc, commentsCtrl, s.commentId)
	// then
	require.NotNil(s.T(), result)
	require.NotNil(s.T(), result.Data)
	assert.NotNil(s.T(), result.Data.ID)
	assert.NotNil(s.T(), result.Data.Type)
	assert.Equal(s.T(), "a comment", *result.Data.Attributes.Body)
	assert.Equal(s.T(), account.TestIdentity.ID, *result.Data.Relationships.CreatedBy.Data.ID)
}

func (s *CommentsSuite) TestShowCommentWithAuth() {
	// given
	commentsCtrl := NewCommentsController(s.userSvc, gormapplication.NewGormDB(s.db))
	// when
	_, result := test.ShowCommentsOK(s.T(), s.userSvc.Context, s.userSvc, commentsCtrl, s.commentId)
	// then
	require.NotNil(s.T(), result)
	require.NotNil(s.T(), result.Data)
	assert.NotNil(s.T(), result.Data.ID)
	assert.NotNil(s.T(), result.Data.Type)
	assert.Equal(s.T(), "a comment", *result.Data.Attributes.Body)
	assert.Equal(s.T(), account.TestIdentity.ID, *result.Data.Relationships.CreatedBy.Data.ID)
}

func (s *CommentsSuite) TestUpdateCommentWithoutAuth() {
	// given
	commentsCtrl := NewCommentsController(s.defaultSvc, gormapplication.NewGormDB(s.db))
	updatedCommentBody := "An updated comment"
	updateCommentPayload := app.UpdateCommentsPayload{
		Data: &app.Comment{
			Type: "comments",
			Attributes: &app.CommentAttributes{
				Body: &updatedCommentBody,
			},
		},
	}
	// when/then
	test.UpdateCommentsUnauthorized(s.T(), s.defaultSvc.Context, s.defaultSvc, commentsCtrl, s.commentId, &updateCommentPayload)
}

func (s *CommentsSuite) TestUpdateCommentWithSameAuthenticatedUser() {
	// given
	commentsCtrl := NewCommentsController(s.userSvc, gormapplication.NewGormDB(s.db))
	updatedCommentBody := "An updated comment"
	updateCommentPayload := app.UpdateCommentsPayload{
		Data: &app.Comment{
			Type: "comments",
			Attributes: &app.CommentAttributes{
				Body: &updatedCommentBody,
			},
		},
	}
	// when/then
	test.UpdateCommentsOK(s.T(), s.userSvc.Context, s.userSvc, commentsCtrl, s.commentId, &updateCommentPayload)
}

func (s *CommentsSuite) TestUpdateCommentWithOtherAuthenticatedUser() {
	// given
	commentsCtrl := NewCommentsController(s.userSvc2, gormapplication.NewGormDB(s.db))
	updatedCommentBody := "An updated comment"
	updateCommentPayload := app.UpdateCommentsPayload{
		Data: &app.Comment{
			Type: "comments",
			Attributes: &app.CommentAttributes{
				Body: &updatedCommentBody,
			},
		},
	}
	// when/then
	test.UpdateCommentsUnauthorized(s.T(), s.userSvc2.Context, s.userSvc2, commentsCtrl, s.commentId, &updateCommentPayload)
}
