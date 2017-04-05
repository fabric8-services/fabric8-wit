package controller_test

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The workItemLinkSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type workItemLinkSuite struct {
	gormtestsupport.DBTestSuite
	svc                      *goa.Service
	workItemLinkTypeCtrl     *WorkItemLinkTypeController
	workItemLinkCategoryCtrl *WorkItemLinkCategoryController
	workItemLinkCtrl         *WorkItemLinkController
	workItemCtrl             *WorkitemController
	workItemRelsLinksCtrl    *WorkItemRelationshipsLinksController
	spaceCtrl                *SpaceController
	typeCtrl                 *WorkitemtypeController
	// These IDs can safely be used by all tests
	bug1ID               uint64
	bug2ID               uint64
	bug3ID               uint64
	feature1ID           uint64
	userLinkCategoryID   uuid.UUID
	bugBlockerLinkTypeID uuid.UUID
	userSpaceID          uuid.UUID
}

// The TearDownSuite method will run after all the tests in the suite have been run
// It tears down the database connection for all the tests in this suite.
func (s *workItemLinkSuite) TearDownSuite() {
	if s.DB != nil {
		s.DB.Unscoped().Delete(&account.Identity{Username: "test user"})
		s.DB.Close()
	}
}

// cleanup removes all DB entries that will be created or have been created
// with this test suite. We need to remove them completely and not only set the
// "deleted_at" field, which is why we need the Unscoped() function.
func (s *workItemLinkSuite) cleanup() {
	// Delete all work item links for now
	db := s.DB.Unscoped().Delete(&link.WorkItemLink{})
	require.Nil(s.T(), db.Error)

	// Delete work item link types and categories by name.
	// They will be created during the tests but have to be deleted by name
	// rather than ID, unlike the work items or work item links.
	db = db.Unscoped().Delete(&link.WorkItemLinkType{Name: "test-bug-blocker"})
	require.Nil(s.T(), db.Error)
	db = db.Unscoped().Delete(&link.WorkItemLinkCategory{Name: "test-user"})
	require.Nil(s.T(), db.Error)
	if s.userSpaceID != uuid.Nil {
		db = db.Unscoped().Delete(&space.Space{ID: s.userSpaceID})
		require.Nil(s.T(), db.Error)
	}
}

// The SetupTest method will be run before every test in the suite.
// SetupTest ensures that none of the work item links that we will create already exist.
// It will also make sure that some resources that we rely on do exists.
func (s *workItemLinkSuite) SetupTest() {
	s.cleanup()
	priv, err := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	require.Nil(s.T(), err)
	svc := goa.New("TestWorkItemLinkType-Service")
	require.NotNil(s.T(), svc)
	s.workItemLinkTypeCtrl = NewWorkItemLinkTypeController(svc, gormapplication.NewGormDB(s.DB))
	require.NotNil(s.T(), s.workItemLinkTypeCtrl)

	svc = goa.New("TestWorkItemLinkCategory-Service")
	require.NotNil(s.T(), svc)
	s.workItemLinkCategoryCtrl = NewWorkItemLinkCategoryController(svc, gormapplication.NewGormDB(s.DB))
	require.NotNil(s.T(), s.workItemLinkCategoryCtrl)

	svc = goa.New("TestWorkItemLinkSpace-Service")
	require.NotNil(s.T(), svc)
	s.spaceCtrl = NewSpaceController(svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})
	require.NotNil(s.T(), s.spaceCtrl)

	svc = goa.New("TestWorkItemType-Service")
	s.typeCtrl = NewWorkitemtypeController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	require.NotNil(s.T(), s.typeCtrl)

	svc = goa.New("TestWorkItemLink-Service")
	require.NotNil(s.T(), svc)
	s.workItemLinkCtrl = NewWorkItemLinkController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	require.NotNil(s.T(), s.workItemLinkCtrl)

	svc = goa.New("TestWorkItemRelationshipsLinks-Service")
	require.NotNil(s.T(), svc)
	s.workItemRelsLinksCtrl = NewWorkItemRelationshipsLinksController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	require.NotNil(s.T(), s.workItemRelsLinksCtrl)

	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "test user", "test provider")
	require.Nil(s.T(), err)
	s.svc = testsupport.ServiceAsUser("TestWorkItem-Service", almtoken.NewManagerWithPrivateKey(priv), testIdentity)
	require.NotNil(s.T(), s.svc)
	s.workItemCtrl = NewWorkitemController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	require.NotNil(s.T(), s.workItemCtrl)
	// Create a work item link space
	createSpacePayload := CreateSpacePayload("test-space", "description")
	_, space := test.CreateSpaceCreated(s.T(), s.svc.Context, s.svc, s.spaceCtrl, createSpacePayload)
	s.userSpaceID = *space.Data.ID
	s.T().Logf("Created link space with ID: %s\n", *space.Data.ID)

	payload := CreateWorkItemType(uuid.NewV4(), *space.Data.ID)
	_, wit := test.CreateWorkitemtypeCreated(s.T(), s.svc.Context, s.svc, s.typeCtrl, s.userSpaceID.String(), &payload)

	payload2 := CreateWorkItemType(uuid.NewV4(), *space.Data.ID)
	_, wit2 := test.CreateWorkitemtypeCreated(s.T(), s.svc.Context, s.svc, s.typeCtrl, s.userSpaceID.String(), &payload2)

	// Create 3 work items (bug1, bug2, and feature1)
	bug1Payload := CreateWorkItem(s.userSpaceID, *wit.Data.ID, "bug1")
	_, bug1 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.workItemCtrl, bug1Payload.Data.Relationships.Space.Data.ID.String(), bug1Payload)
	require.NotNil(s.T(), bug1)
	s.bug1ID, err = strconv.ParseUint(*bug1.Data.ID, 10, 64)
	require.Nil(s.T(), err)
	s.T().Logf("Created bug1 with ID: %s\n", *bug1.Data.ID)

	bug2Payload := CreateWorkItem(s.userSpaceID, *wit.Data.ID, "bug2")
	_, bug2 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.workItemCtrl, bug2Payload.Data.Relationships.Space.Data.ID.String(), bug2Payload)
	require.NotNil(s.T(), bug2)
	s.bug2ID, err = strconv.ParseUint(*bug2.Data.ID, 10, 64)
	require.Nil(s.T(), err)
	s.T().Logf("Created bug2 with ID: %s\n", *bug2.Data.ID)

	bug3Payload := CreateWorkItem(s.userSpaceID, *wit.Data.ID, "bug3")
	_, bug3 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.workItemCtrl, bug3Payload.Data.Relationships.Space.Data.ID.String(), bug3Payload)
	require.NotNil(s.T(), bug3)
	s.bug3ID, err = strconv.ParseUint(*bug3.Data.ID, 10, 64)
	require.Nil(s.T(), err)
	s.T().Logf("Created bug3 with ID: %s\n", *bug3.Data.ID)

	feature1Payload := CreateWorkItem(s.userSpaceID, *wit2.Data.ID, "feature1")
	_, feature1 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.workItemCtrl, feature1Payload.Data.Relationships.Space.Data.ID.String(), feature1Payload)
	require.NotNil(s.T(), feature1)
	s.feature1ID, err = strconv.ParseUint(*feature1.Data.ID, 10, 64)
	require.Nil(s.T(), err)
	s.T().Logf("Created workitem with ID: %s\n", *feature1.Data.ID)

	// Create a work item link category
	createLinkCategoryPayload := CreateWorkItemLinkCategory("test-user")
	_, workItemLinkCategory := test.CreateWorkItemLinkCategoryCreated(s.T(), s.svc.Context, s.svc, s.workItemLinkCategoryCtrl, createLinkCategoryPayload)
	require.NotNil(s.T(), workItemLinkCategory)
	//s.deleteWorkItemLinkCategories = append(s.deleteWorkItemLinkCategories, *workItemLinkCategory.Data.ID)
	s.userLinkCategoryID = *workItemLinkCategory.Data.ID
	s.T().Logf("Created link category with ID: %s\n", *workItemLinkCategory.Data.ID)

	// Create work item link type payload
	createLinkTypePayload := CreateWorkItemLinkType("test-bug-blocker", *wit.Data.ID, *wit.Data.ID, s.userLinkCategoryID, s.userSpaceID)
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.workItemLinkTypeCtrl, s.userSpaceID.String(), createLinkTypePayload)
	require.NotNil(s.T(), workItemLinkType)
	//s.deleteWorkItemLinkTypes = append(s.deleteWorkItemLinkTypes, *workItemLinkType.Data.ID)
	s.bugBlockerLinkTypeID = *workItemLinkType.Data.ID
	s.T().Logf("Created link type with ID: %s\n", *workItemLinkType.Data.ID)

}

// The TearDownTest method will be run after every test in the suite.
func (s *workItemLinkSuite) TearDownTest() {
	s.cleanup()
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

// CreateWorkItemLinkCategory creates a work item link category
func CreateWorkItemLinkCategory(name string) *app.CreateWorkItemLinkCategoryPayload {
	description := "This work item link category is managed by an admin user."
	// Use the goa generated code to create a work item link category
	return &app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			Type: link.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}
}

// CreateWorkItem defines a work item link
func CreateWorkItem(spaceID uuid.UUID, workItemType uuid.UUID, title string) *app.CreateWorkitemPayload {
	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(spaceID.String()))
	witSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.WorkitemtypeHref(spaceID.String(), workItemType))
	payload := app.CreateWorkitemPayload{
		Data: &app.WorkItem{
			Attributes: map[string]interface{}{
				workitem.SystemTitle: title,
				workitem.SystemState: workitem.SystemStateClosed,
			},
			Relationships: &app.WorkItemRelationships{
				BaseType: &app.RelationBaseType{
					Data: &app.BaseTypeData{
						ID:   workItemType,
						Type: "workitemtypes",
					},
					Links: &app.GenericLinks{
						Self: &witSelfURL,
					},
				},
				Space: app.NewSpaceRelation(spaceID, spaceSelfURL),
			},
			Type: "workitems",
		},
	}
	return &payload
}

// CreateWorkItemLinkType defines a work item link type
func CreateWorkItemLinkType(name string, sourceTypeID, targetTypeID, categoryID, spaceID uuid.UUID) *app.CreateWorkItemLinkTypePayload {
	description := "Specify that one bug blocks another one."
	lt := link.WorkItemLinkType{
		Name:           name,
		Description:    &description,
		SourceTypeID:   sourceTypeID,
		TargetTypeID:   targetTypeID,
		Topology:       link.TopologyNetwork,
		ForwardName:    "forward name string for " + name,
		ReverseName:    "reverse name string for " + name,
		LinkCategoryID: categoryID,
		SpaceID:        spaceID,
	}
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	payload := ConvertWorkItemLinkTypeFromModel(reqLong, lt)
	// The create payload is required during creation. Simply copy data over.
	return &app.CreateWorkItemLinkTypePayload{
		Data: payload.Data,
	}
}

// CreateWorkItemLink defines a work item link
func CreateWorkItemLink(sourceID uint64, targetID uint64, linkTypeID uuid.UUID) *app.CreateWorkItemLinkPayload {
	lt := link.WorkItemLink{
		SourceID:   sourceID,
		TargetID:   targetID,
		LinkTypeID: linkTypeID,
	}
	payload := ConvertLinkFromModel(lt)
	// The create payload is required during creation. Simply copy data over.
	return &app.CreateWorkItemLinkPayload{
		Data: payload.Data,
	}
}

//-----------------------------------------------------------------------------
// Actual tests
//-----------------------------------------------------------------------------

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemLink(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(workItemLinkSuite))
}

func (s *workItemLinkSuite) TestCreateAndDeleteWorkItemLink() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	_, workItemLink := test.CreateWorkItemLinkCreated(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload)
	require.NotNil(s.T(), workItemLink)

	// Test if related resources are included in the response
	toBeFound := 2
	for i := 0; i < len(workItemLink.Included) && toBeFound > 0; i++ {
		switch v := workItemLink.Included[i].(type) {
		case *app.WorkItemLinkCategoryData:
			if *v.ID == s.userLinkCategoryID {
				s.T().Log("Found work item link category in \"included\" element: ", *v.ID)
				toBeFound--
			}
		case *app.Space:
			if *v.ID == s.userSpaceID {
				s.T().Log("Found work item link space in \"included\" element: ", *v.ID)
				toBeFound--
			}
		case *app.WorkItemLinkTypeData:
			if *v.ID == s.bugBlockerLinkTypeID {
				s.T().Log("Found work item link type in \"included\" element: ", *v.ID)
				toBeFound--
			}
		// TODO(kwk): Check for source WI (once #559 is merged)
		// TODO(kwk): Check for target WI (once #559 is merged)
		// case *app.WorkItemData:
		// TODO(kwk): Check for WITs (once #559 is merged)
		// case *app.WorkItemTypeData:
		default:
			s.T().Errorf("Object of unknown type included in work item link list response: %T", workItemLink.Included[i])
		}
	}
	require.Exactly(s.T(), 0, toBeFound, "Not all required included elements where found.")

	_ = test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, *workItemLink.Data.ID)
}

// Check if #586 is fixed.
func (s *workItemLinkSuite) TestCreateAndDeleteWorkItemLinkBadRequestDueToUniqueViolation() {
	createPayload1 := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	_, workItemLink1 := test.CreateWorkItemLinkCreated(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload1)
	require.NotNil(s.T(), workItemLink1)
	createPayload2 := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	_, _ = test.CreateWorkItemLinkBadRequest(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload2)
}

// Same for /api/workitems/:id/relationships/links
func (s *workItemLinkSuite) TestCreateAndDeleteWorkItemRelationshipsLink() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	_, workItemLink := test.CreateWorkItemRelationshipsLinksCreated(s.T(), s.svc.Context, s.svc, s.workItemRelsLinksCtrl, s.userSpaceID.String(), strconv.FormatUint(s.bug1ID, 10), createPayload)
	require.NotNil(s.T(), workItemLink)
}

func (s *workItemLinkSuite) TestCreateWorkItemLinkBadRequestDueToInvalidLinkTypeID() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, uuid.Nil)
	_, _ = test.CreateWorkItemLinkBadRequest(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload)
}

// Same for /api/workitems/:id/relationships/links
func (s *workItemLinkSuite) TestCreateWorkItemRelationshipsLinksBadRequestDueToInvalidLinkTypeID() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, uuid.Nil)
	_, _ = test.CreateWorkItemRelationshipsLinksBadRequest(s.T(), s.svc.Context, s.svc, s.workItemRelsLinksCtrl, s.userSpaceID.String(), strconv.FormatUint(s.bug1ID, 10), createPayload)
}

func (s *workItemLinkSuite) TestCreateWorkItemLinkBadRequestDueToNotFoundLinkType() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, uuid.FromStringOrNil("11122233-871b-43a6-9166-0c4bd573e333"))
	_, _ = test.CreateWorkItemLinkBadRequest(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload)
}

// Same for /api/workitems/:id/relationships/links
func (s *workItemLinkSuite) TestCreateWorkItemRelationshipLinksBadRequestDueToNotFoundLinkType() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, uuid.FromStringOrNil("11122233-871b-43a6-9166-0c4bd573e333"))
	_, _ = test.CreateWorkItemRelationshipsLinksBadRequest(s.T(), s.svc.Context, s.svc, s.workItemRelsLinksCtrl, s.userSpaceID.String(), strconv.FormatUint(s.bug1ID, 10), createPayload)
}

func (s *workItemLinkSuite) TestCreateWorkItemLinkBadRequestDueToNotFoundSource() {
	createPayload := CreateWorkItemLink(666666, s.bug2ID, s.bugBlockerLinkTypeID)
	_, _ = test.CreateWorkItemLinkBadRequest(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload)
}

// Same for /api/workitems/:id/relationships/links
func (s *workItemLinkSuite) TestCreateWorkItemRelationshipsLinksBadRequestDueToNotFoundSource() {
	createPayload := CreateWorkItemLink(666666, s.bug2ID, s.bugBlockerLinkTypeID)
	_, _ = test.CreateWorkItemRelationshipsLinksBadRequest(s.T(), s.svc.Context, s.svc, s.workItemRelsLinksCtrl, s.userSpaceID.String(), strconv.FormatUint(s.bug2ID, 10), createPayload)
}

func (s *workItemLinkSuite) TestCreateWorkItemLinkBadRequestDueToNotFoundTarget() {
	createPayload := CreateWorkItemLink(s.bug1ID, 666666, s.bugBlockerLinkTypeID)
	_, _ = test.CreateWorkItemLinkBadRequest(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload)
}

// Same for /api/workitems/:id/relationships/links
func (s *workItemLinkSuite) TestCreateWorkItemRelationshipsLinksBadRequestDueToNotFoundTarget() {
	createPayload := CreateWorkItemLink(s.bug1ID, 666666, s.bugBlockerLinkTypeID)
	_, _ = test.CreateWorkItemRelationshipsLinksBadRequest(s.T(), s.svc.Context, s.svc, s.workItemRelsLinksCtrl, s.userSpaceID.String(), strconv.FormatUint(s.bug1ID, 10), createPayload)
}

func (s *workItemLinkSuite) TestCreateWorkItemLinkBadRequestDueToBadSourceType() {
	// Linking a bug and a feature isn't allowed for the bug blocker link type,
	// thererfore this will cause a bad parameter error (which results in a bad request error).
	createPayload := CreateWorkItemLink(s.feature1ID, s.bug1ID, s.bugBlockerLinkTypeID)
	_, _ = test.CreateWorkItemLinkBadRequest(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload)
}

// Same for /api/workitems/:id/relationships/links
func (s *workItemLinkSuite) TestCreateWorkItemRelationshipsLinksBadRequestDueToBadSourceType() {
	// Linking a bug and a feature isn't allowed for the bug blocker link type,
	// thererfore this will cause a bad parameter error (which results in a bad request error).
	createPayload := CreateWorkItemLink(s.feature1ID, s.bug1ID, s.bugBlockerLinkTypeID)
	_, _ = test.CreateWorkItemRelationshipsLinksBadRequest(s.T(), s.svc.Context, s.svc, s.workItemRelsLinksCtrl, s.userSpaceID.String(), strconv.FormatUint(s.feature1ID, 10), createPayload)
}

func (s *workItemLinkSuite) TestCreateWorkItemLinkBadRequestDueToBadTargetType() {
	// Linking a bug and a feature isn't allowed for the bug blocker link type,
	// thererfore this will cause a bad parameter error (which results in a bad request error).
	createPayload := CreateWorkItemLink(s.bug1ID, s.feature1ID, s.bugBlockerLinkTypeID)
	_, _ = test.CreateWorkItemLinkBadRequest(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload)
}

// Same for /api/workitems/:id/relationships/links
func (s *workItemLinkSuite) TestCreateWorkItemRelationshipsLinksBadRequestDueToBadTargetType() {
	// Linking a bug and a feature isn't allowed for the bug blocker link type,
	// thererfore this will cause a bad parameter error (which results in a bad request error).
	createPayload := CreateWorkItemLink(s.bug1ID, s.feature1ID, s.bugBlockerLinkTypeID)
	_, _ = test.CreateWorkItemRelationshipsLinksBadRequest(s.T(), s.svc.Context, s.svc, s.workItemRelsLinksCtrl, s.userSpaceID.String(), strconv.FormatUint(s.bug1ID, 10), createPayload)
}

func (s *workItemLinkSuite) TestDeleteWorkItemLinkNotFound() {
	test.DeleteWorkItemLinkNotFound(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, uuid.FromStringOrNil("1e9a8b53-73a6-40de-b028-5177add79ffa"))
}

func (s *workItemLinkSuite) TestUpdateWorkItemLinkNotFound() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.userLinkCategoryID)
	notExistingId := uuid.FromStringOrNil("46bbce9c-8219-4364-a450-dfd1b501654e")
	createPayload.Data.ID = &notExistingId
	// Wrap data portion in an update payload instead of a create payload
	updateLinkPayload := &app.UpdateWorkItemLinkPayload{
		Data: createPayload.Data,
	}
	test.UpdateWorkItemLinkNotFound(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, *updateLinkPayload.Data.ID, updateLinkPayload)
}

func (s *workItemLinkSuite) TestUpdateWorkItemLinkOK() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	_, workItemLink := test.CreateWorkItemLinkCreated(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload)
	require.NotNil(s.T(), workItemLink)
	// Specify new description for link type that we just created
	// Wrap data portion in an update payload instead of a create payload
	updateLinkPayload := &app.UpdateWorkItemLinkPayload{
		Data: workItemLink.Data,
	}
	updateLinkPayload.Data.Relationships.Target.Data.ID = strconv.FormatUint(s.bug3ID, 10)
	_, l := test.UpdateWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, *updateLinkPayload.Data.ID, updateLinkPayload)
	require.NotNil(s.T(), l.Data)
	require.NotNil(s.T(), l.Data.Relationships)
	require.NotNil(s.T(), l.Data.Relationships.Target.Data)
	require.Equal(s.T(), strconv.FormatUint(s.bug3ID, 10), l.Data.Relationships.Target.Data.ID)
}

func (s *workItemLinkSuite) createWorkItemLink() *app.WorkItemLinkSingle {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	_, workItemLink := test.CreateWorkItemLinkCreated(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload)
	require.NotNil(s.T(), workItemLink)
	require.NotNil(s.T(), workItemLink.Data.Attributes.UpdatedAt)
	return workItemLink
}

func assertWorkItemLink(t *testing.T, expectedWorkItemLink *app.WorkItemLinkSingle, wiLink *app.WorkItemLinkSingle) {
	require.NotNil(t, wiLink)
	expected, err := ConvertLinkToModel(*expectedWorkItemLink)
	require.Nil(t, err)
	// Convert to model space and use equal function
	actual, err := ConvertLinkToModel(*wiLink)
	require.Nil(t, err)
	require.Equal(t, *expected, *actual)
	require.NotNil(t, wiLink.Data.Links, "The link MUST include a self link")
	require.NotEmpty(t, wiLink.Data.Links.Self, "The link MUST include a self link that's not empty")
}

// TestShowWorkItemLinkOK tests if we can fetch the "system" work item link
func (s *workItemLinkSuite) TestShowWorkItemLinkOK() {
	// given
	createdWorkItemLink := s.createWorkItemLink()
	// when
	_, retrievedWorkItemLink := test.ShowWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, *createdWorkItemLink.Data.ID, nil, nil)
	// then
	assertWorkItemLink(s.T(), createdWorkItemLink, retrievedWorkItemLink)
}

// TestShowWorkItemLinkOKUsingExpiredIfModifiedSinceHeader
func (s *workItemLinkSuite) TestShowWorkItemLinkOKUsingExpiredIfModifiedSinceHeader() {
	// given
	createdWorkItemLink := s.createWorkItemLink()
	// when
	ifModifiedSince := app.ToHTTPTime(createdWorkItemLink.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
	res, retrievedWorkItemLink := test.ShowWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, *createdWorkItemLink.Data.ID, &ifModifiedSince, nil)
	// then
	assertWorkItemLink(s.T(), createdWorkItemLink, retrievedWorkItemLink)
	assertResponseHeaders(s.T(), res)
}

// TestShowWorkItemLinkOKUsingExpiredIfModifiedSinceHeader
func (s *workItemLinkSuite) TestShowWorkItemLinkOKUsingExpiredIfNoneMatchHeader() {
	// given
	createdWorkItemLink := s.createWorkItemLink()
	// when
	ifNoneMatch := "foo"
	res, retrievedWorkItemLink := test.ShowWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, *createdWorkItemLink.Data.ID, nil, &ifNoneMatch)
	// then
	assertWorkItemLink(s.T(), createdWorkItemLink, retrievedWorkItemLink)
	assertResponseHeaders(s.T(), res)
}

// TestShowWorkItemLinkOKUsingExpiredIfModifiedSinceHeader
func (s *workItemLinkSuite) TestShowWorkItemLinkNotModifiedUsingIfModifiedSinceHeader() {
	// given
	createdWorkItemLink := s.createWorkItemLink()
	// when
	ifModifiedSince := app.ToHTTPTime(*createdWorkItemLink.Data.Attributes.UpdatedAt)
	res := test.ShowWorkItemLinkNotModified(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, *createdWorkItemLink.Data.ID, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(s.T(), res)
}

// TestShowWorkItemLinkOKUsingExpiredIfModifiedSinceHeader
func (s *workItemLinkSuite) TestShowWorkItemLinkNotModifiedUsingIfNoneMatchHeader() {
	// given
	createdWorkItemLink := s.createWorkItemLink()
	// when
	modelWorkItemLink, err := ConvertLinkToModel(*createdWorkItemLink)
	require.Nil(s.T(), err)
	ifNoneMatch := app.GenerateEntityTag(modelWorkItemLink)
	res := test.ShowWorkItemLinkNotModified(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, *createdWorkItemLink.Data.ID, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(s.T(), res)
}

// TestShowWorkItemLinkNotFound tests if we can fetch a non existing work item link
func (s *workItemLinkSuite) TestShowWorkItemLinkNotFound() {
	test.ShowWorkItemLinkNotFound(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, uuid.FromStringOrNil("88727441-4a21-4b35-aabe-007f8273cd19"), nil, nil)
}

func (s *workItemLinkSuite) createSomeLinks() (*app.WorkItemLinkSingle, *app.WorkItemLinkSingle) {
	createPayload1 := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	_, workItemLink1 := test.CreateWorkItemLinkCreated(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload1)
	require.NotNil(s.T(), workItemLink1)
	_, err := ConvertLinkToModel(*workItemLink1)
	require.Nil(s.T(), err)

	createPayload2 := CreateWorkItemLink(s.bug2ID, s.bug3ID, s.bugBlockerLinkTypeID)
	_, workItemLink2 := test.CreateWorkItemLinkCreated(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload2)
	require.NotNil(s.T(), workItemLink2)
	_, err = ConvertLinkToModel(*workItemLink2)
	require.Nil(s.T(), err)

	return workItemLink1, workItemLink2
}

// validateSomeLinks validates that workItemLink1 and workItemLink2 are in the
// linkCollection and that all resources are included
func (s *workItemLinkSuite) validateSomeLinks(linkCollection *app.WorkItemLinkList, workItemLink1, workItemLink2 *app.WorkItemLinkSingle) {
	require.NotNil(s.T(), linkCollection)
	require.Nil(s.T(), linkCollection.Validate())
	// Check the number of found work item links
	require.NotNil(s.T(), linkCollection.Data)
	require.Condition(s.T(), func() bool {
		return (len(linkCollection.Data) >= 2)
	}, "At least two work item links must exist (%s and %s), but only %d exist.", *workItemLink1.Data.ID, *workItemLink2.Data.ID, len(linkCollection.Data))
	// Search for the work item types that must exist at minimum
	toBeFound := 2
	for i := 0; i < len(linkCollection.Data) && toBeFound > 0; i++ {
		actualLink := *linkCollection.Data[i]
		var expectedLink *app.WorkItemLinkData

		switch *actualLink.ID {
		case *workItemLink1.Data.ID:
			expectedLink = workItemLink1.Data
		case *workItemLink2.Data.ID:
			expectedLink = workItemLink2.Data
		}

		if expectedLink != nil {
			s.T().Log("Found work item link in collection: ", *expectedLink.ID)
			toBeFound--

			// Check JSONAPI "type"" field (should be "workitemlinks")
			require.Equal(s.T(), expectedLink.Type, actualLink.Type)

			// Check work item link type
			require.Equal(s.T(), expectedLink.Relationships.LinkType.Data.ID, actualLink.Relationships.LinkType.Data.ID)
			require.Equal(s.T(), expectedLink.Relationships.LinkType.Data.Type, actualLink.Relationships.LinkType.Data.Type)

			// Check source type
			require.Equal(s.T(), expectedLink.Relationships.Source.Data.ID, actualLink.Relationships.Source.Data.ID, "Wrong source ID for the link")
			require.Equal(s.T(), expectedLink.Relationships.Source.Data.Type, actualLink.Relationships.Source.Data.Type, "Wrong source JSONAPI type for the link")

			// Check target type
			require.Equal(s.T(), expectedLink.Relationships.Target.Data.ID, actualLink.Relationships.Target.Data.ID, "Wrong target ID for the link")
			require.Equal(s.T(), expectedLink.Relationships.Target.Data.Type, actualLink.Relationships.Target.Data.Type, "Wrong target JSONAPI type for the link")
		}
	}
	require.Exactly(s.T(), 0, toBeFound, "Not all required work item links (%s and %s) where found.", *workItemLink1.Data.ID, *workItemLink2.Data.ID)

	toBeFound = 5 // 1 x link category, 1 x link type, 3 x work items
	for i := 0; i < len(linkCollection.Included) && toBeFound > 0; i++ {
		switch v := linkCollection.Included[i].(type) {
		case *app.WorkItemLinkCategoryData:
			if *v.ID == s.userLinkCategoryID {
				s.T().Log("Found work item link category in \"included\" element: ", *v.ID)
				toBeFound--
			}
		case *app.Space:
			if *v.ID == s.userSpaceID {
				s.T().Log("Found work item link space in \"included\" element: ", *v.ID)
				toBeFound--
			}
		case *app.WorkItemLinkTypeData:
			if *v.ID == s.bugBlockerLinkTypeID {
				s.T().Log("Found work item link type in \"included\" element: ", *v.ID)
				toBeFound--
			}
		case *app.WorkItem:
			wid, err := strconv.ParseUint(*v.ID, 10, 64)
			require.Nil(s.T(), err)
			if wid == s.bug1ID || wid == s.bug2ID || wid == s.bug3ID {
				s.T().Log("Found work item in \"included\" element: ", *v.ID)
				toBeFound--
			}
		// TODO(kwk): Check for WITs (once #559 is merged)
		// case *app.WorkItemTypeData:
		default:
			s.T().Errorf("Object of unknown type included in work item link list response: %T", linkCollection.Included[i])
		}
	}
	require.Exactly(s.T(), 0, toBeFound, "Not all required included elements where found.")
}

// TestListWorkItemLinkOK tests if we can find the work item links
// "test-bug-blocker" and "related" in the list of work item links
func (s *workItemLinkSuite) TestListWorkItemLinkOK() {
	// given
	link1, link2 := s.createSomeLinks()
	// when
	_, linkCollection := test.ListWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, nil, nil)
	// then
	s.validateSomeLinks(linkCollection, link1, link2)
}

func (s *workItemLinkSuite) TestListWorkItemLinkOKUsingExpiredIfModifiedSinceHeader() {
	// given
	link1, link2 := s.createSomeLinks()
	// when
	ifModifiedSince := app.ToHTTPTime(link2.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
	res, linkCollection := test.ListWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, &ifModifiedSince, nil)
	// then
	s.validateSomeLinks(linkCollection, link1, link2)
	assertResponseHeaders(s.T(), res)
}

func (s *workItemLinkSuite) TestListWorkItemLinkOKUsingExpiredIfNoneMatchHeader() {
	// given
	link1, link2 := s.createSomeLinks()
	// when
	ifNoneMatch := "foo"
	res, linkCollection := test.ListWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, nil, &ifNoneMatch)
	// then
	s.validateSomeLinks(linkCollection, link1, link2)
	assertResponseHeaders(s.T(), res)
}

func (s *workItemLinkSuite) TestListWorkItemLinkNotModifiedUsingIfModifiedSinceHeader() {
	// given
	_, link2 := s.createSomeLinks()
	// when
	ifModifiedSince := app.ToHTTPTime(*link2.Data.Attributes.UpdatedAt)
	res := test.ListWorkItemLinkNotModified(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(s.T(), res)
}

func (s *workItemLinkSuite) TestListWorkItemLinkNotModifiedUsingIfNoneMatchHeader() {
	// given
	link1, link2 := s.createSomeLinks()
	// when
	modelLink1, _ := ConvertLinkToModel(*link1)
	modelLink2, _ := ConvertLinkToModel(*link2)
	ifNoneMatch := app.GenerateEntitiesTag([]app.ConditionalResponseEntity{modelLink1, modelLink2})
	res := test.ListWorkItemLinkNotModified(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(s.T(), res)
}

// Same as TestListWorkItemLinkOK, for /api/workitems/:id/relationships/links
func (s *workItemLinkSuite) TestListWorkItemRelationshipsLinksOK() {
	link1, link2 := s.createSomeLinks()
	filterByWorkItemID := strconv.FormatUint(s.bug2ID, 10)
	_, linkCollection := test.ListWorkItemRelationshipsLinksOK(s.T(), s.svc.Context, s.svc, s.workItemRelsLinksCtrl, s.userSpaceID.String(), filterByWorkItemID, nil, nil)
	s.validateSomeLinks(linkCollection, link1, link2)
}

func (s *workItemLinkSuite) TestListWorkItemRelationshipsLinksNotFound() {
	filterByWorkItemID := strconv.FormatUint(math.MaxUint32, 10) // not existing bug ID
	spaceRandomID := strconv.FormatUint(math.MaxUint32, 10)      // not existing space ID
	_, _ = test.ListWorkItemRelationshipsLinksNotFound(s.T(), s.svc.Context, s.svc, s.workItemRelsLinksCtrl, spaceRandomID, filterByWorkItemID, nil, nil)
}

func (s *workItemLinkSuite) TestListWorkItemRelationshipsLinksNotFoundDueToInvalidID() {
	filterByWorkItemID := "invalid uint64"
	_, _ = test.ListWorkItemRelationshipsLinksNotFound(s.T(), s.svc.Context, s.svc, s.workItemRelsLinksCtrl, s.userSpaceID.String(), filterByWorkItemID, nil, nil)
}

func (s *workItemLinkSuite) getWorkItemLinkTestDataFunc() func(t *testing.T) []testSecureAPI {
	return func(t *testing.T) []testSecureAPI {
		privatekey, err := jwt.ParseRSAPrivateKeyFromPEM(s.Configuration.GetTokenPrivateKey())
		require.Nil(t, err, "Could not parse private key")
		differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
		require.Nil(t, err, "Could not parse private key")
		createWorkItemLinkPayloadString := bytes.NewBuffer([]byte(`
		{
			"data": {
				"attributes": {
					"version": 0
				},
				"id": "40bbdd3d-8b5d-4fd6-ac90-7236b669af04",
				"relationships": {
					"link_type": {
						"data": {
						"id": "6c5610be-30b2-4880-9fec-81e4f8e4fd76",
						"type": "workitemlinktypes"
						}
					},
					"source": {
						"data": {
						"id": "1234",
						"type": "workitems"
						}
					},
					"target": {
						"data": {
						"id": "1234",
						"type": "workitems"
						}
					}
				},
				"type": "workitemlinks"
			}
		}
  		`))

		testWorkItemLinksAPI := []testSecureAPI{
			// Create Work Item API with different parameters
			{
				method:             http.MethodPost,
				url:                endpointWorkItemLinks,
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                endpointWorkItemLinks,
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                endpointWorkItemLinks,
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodPost,
				url:                endpointWorkItemLinks,
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           "",
			},
			// Update Work Item API with different parameters
			{
				method:             http.MethodPatch,
				url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPatch,
				url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPatch,
				url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodPatch,
				url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           "",
			},
			// Delete Work Item API with different parameters
			{
				method:             http.MethodDelete,
				url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodDelete,
				url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodDelete,
				url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodDelete,
				url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           "",
			},
			// Try fetching a random work item link
			// We do not have security on GET hence this should return 404 not found
			{
				method:             http.MethodGet,
				url:                endpointWorkItemLinks + "/fc591f38-a805-4abd-bfce-2460e49d8cc4",
				expectedStatusCode: http.StatusNotFound,
				expectedErrorCode:  jsonapi.ErrorCodeNotFound,
				payload:            nil,
				jwtToken:           "",
			},
		}
		return testWorkItemLinksAPI
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func (s *workItemLinkSuite) TestUnauthorizeWorkItemLinkCUD() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), s.getWorkItemLinkTestDataFunc(), func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWorkItemLink-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkItemLinkController(service, gormapplication.NewGormDB(s.DB), s.Configuration)
		app.MountWorkItemLinkController(service, controller)
		return nil
	})
}

// The work item ID will be used to construct /api/workitems/:id/relationships/links endpoints
func (s *workItemLinkSuite) getWorkItemRelationshipLinksTestData(spaceID, wiID string) func(t *testing.T) []testSecureAPI {
	return func(t *testing.T) []testSecureAPI {
		privatekey, err := jwt.ParseRSAPrivateKeyFromPEM(s.Configuration.GetTokenPrivateKey())
		if err != nil {
			t.Fatal("Could not parse Key ", err)
		}
		differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
		if err != nil {
			t.Fatal("Could not parse different private key ", err)
		}

		createWorkItemLinkPayloadString := bytes.NewBuffer([]byte(`
		{
			"data": {
				"attributes": {
					"version": 0
				},
				"id": "40bbdd3d-8b5d-4fd6-ac90-7236b669af04",
				"relationships": {
					"link_type": {
						"data": {
						"id": "6c5610be-30b2-4880-9fec-81e4f8e4fd76",
						"type": "workitemlinktypes"
						}
					},
					"source": {
						"data": {
						"id": "1234",
						"type": "workitems"
						}
					},
					"target": {
						"data": {
						"id": "1234",
						"type": "workitems"
						}
					}
				},
				"type": "workitemlinks"
			}
		}
  		`))

		relationshipsEndpoint := fmt.Sprintf(endpointWorkItemRelationshipsLinks, spaceID, wiID)
		testWorkItemLinksAPI := []testSecureAPI{
			// Create Work Item API with different parameters
			{
				method:             http.MethodPost,
				url:                relationshipsEndpoint,
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                relationshipsEndpoint,
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                relationshipsEndpoint,
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodPost,
				url:                relationshipsEndpoint,
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           "",
			},
		}
		return testWorkItemLinksAPI
	}
}

func (s *workItemLinkSuite) TestUnauthorizeWorkItemRelationshipsLinksCUD() {
	wiID := strconv.FormatUint(s.bug1ID, 10)
	UnauthorizeCreateUpdateDeleteTest(s.T(), s.getWorkItemRelationshipLinksTestData(space.SystemSpace.String(), wiID), func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWorkItemRelationshipsLinks-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkItemRelationshipsLinksController(service, gormapplication.NewGormDB(s.DB), s.Configuration)
		app.MountWorkItemRelationshipsLinksController(service, controller)
		return nil
	})
}

func TestNewWorkItemLinkControllerDBNull(t *testing.T) {
	require.Panics(t, func() {
		NewWorkItemLinkController(nil, nil, nil)
	})
}

func TestNewWorkItemRelationshipsLinksControllerDBNull(t *testing.T) {
	require.Panics(t, func() {
		NewWorkItemRelationshipsLinksController(nil, nil, nil)
	})
}
