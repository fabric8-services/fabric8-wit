package controller_test

import (
	"net/http"
	"strconv"
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The workItemChildSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type workItemChildSuite struct {
	suite.Suite
	db                       *gorm.DB
	workItemLinkTypeCtrl     *WorkItemLinkTypeController
	workItemLinkCategoryCtrl *WorkItemLinkCategoryController
	workItemLinkCtrl         *WorkItemLinkController
	workItemChildrenCtrl     *WorkItemChildrenController
	workItemCtrl             *WorkitemController
	workItemRelsLinksCtrl    *WorkItemRelationshipsLinksController
	spaceCtrl                *SpaceController
	workItemSvc              *goa.Service
	typeCtrl                 *WorkitemtypeController

	// These IDs can safely be used by all tests
	bug1ID               uint64
	bug2ID               uint64
	bug3ID               uint64
	feature1ID           uint64
	userLinkCategoryID   uuid.UUID
	bugBlockerLinkTypeID uuid.UUID
	userSpaceID          uuid.UUID

	// Store IDs of resources that need to be removed at the beginning or end of a test
	deleteWorkItemLinks []uuid.UUID
	deleteWorkItems     []string
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemChildSuite) SetupSuite() {
	var err error

	s.db, err = gorm.Open("postgres", wiConfiguration.GetPostgresConfigString())

	if err != nil {
		panic("Failed to connect database: " + err.Error())
	}

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if err := models.Transactional(DB, func(tx *gorm.DB) error {
		return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
	}); err != nil {
		panic(err.Error())
	}

	require.Nil(s.T(), err)
	priv, err := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	require.Nil(s.T(), err)

	svc := goa.New("TestWorkItemLinkType-Service")
	require.NotNil(s.T(), svc)
	s.workItemLinkTypeCtrl = NewWorkItemLinkTypeController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.workItemLinkTypeCtrl)

	svc = goa.New("TestWorkItemLinkCategory-Service")
	require.NotNil(s.T(), svc)
	s.workItemLinkCategoryCtrl = NewWorkItemLinkCategoryController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.workItemLinkCategoryCtrl)

	svc = goa.New("TestWorkItemLinkSpace-Service")
	require.NotNil(s.T(), svc)
	s.spaceCtrl = NewSpaceController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.spaceCtrl)

	svc = goa.New("TestWorkItemType-Service")
	s.typeCtrl = NewWorkitemtypeController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.typeCtrl)

	svc = goa.New("TestWorkItemLink-Service")
	require.NotNil(s.T(), svc)
	s.workItemLinkCtrl = NewWorkItemLinkController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.workItemLinkCtrl)

	svc = goa.New("TestWorkItemRelationshipsLinks-Service")
	require.NotNil(s.T(), svc)
	s.workItemRelsLinksCtrl = NewWorkItemRelationshipsLinksController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.workItemRelsLinksCtrl)

	svc = goa.New("TestWorkItemChildren-Service")
	require.NotNil(s.T(), svc)
	s.workItemChildrenCtrl = NewWorkItemChildrenController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.workItemChildrenCtrl)

	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(s.db, "test user", "test provider")
	require.Nil(s.T(), err)
	s.workItemSvc = testsupport.ServiceAsUser("TestWorkItem-Service", almtoken.NewManagerWithPrivateKey(priv), testIdentity)
	require.NotNil(s.T(), s.workItemSvc)
	s.workItemCtrl = NewWorkitemController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.workItemCtrl)
}

// The TearDownSuite method will run after all the tests in the suite have been run
// It tears down the database connection for all the tests in this suite.
func (s *workItemChildSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
}

// cleanup removes all DB entries that will be created or have been created
// with this test suite. We need to remove them completely and not only set the
// "deleted_at" field, which is why we need the Unscoped() function.
func (s *workItemChildSuite) cleanup() {
	db := s.db

	// First delete work item links and then the types;
	// otherwise referential integrity will be violated.
	for _, id := range s.deleteWorkItemLinks {
		db = db.Unscoped().Delete(&link.WorkItemLink{ID: id})
		require.Nil(s.T(), db.Error)
	}
	s.deleteWorkItemLinks = nil

	// Delete all work item links for now
	db.Unscoped().Delete(&link.WorkItemLink{})
	require.Nil(s.T(), db.Error)

	// Delete work item link types and categories by name.
	// They will be created during the tests but have to be deleted by name
	// rather than ID, unlike the work items or work item links.
	db = db.Unscoped().Delete(&link.WorkItemLinkType{Name: "test-bug-blocker"})
	require.Nil(s.T(), db.Error)
	db = db.Unscoped().Delete(&link.WorkItemLinkCategory{Name: "test-user"})
	require.Nil(s.T(), db.Error)
	db = db.Unscoped().Delete(&space.Space{Name: "test-space"})
	require.Nil(s.T(), db.Error)

	// Last but not least delete the work items
	for _, idStr := range s.deleteWorkItems {
		id, err := strconv.ParseUint(idStr, 10, 64)
		require.Nil(s.T(), err)
		db = db.Unscoped().Delete(&workitem.WorkItem{ID: id})
		require.Nil(s.T(), db.Error)
	}
	s.deleteWorkItems = nil

}

// The SetupTest method will be run before every test in the suite.
// SetupTest ensures that none of the work item links that we will create already exist.
// It will also make sure that some resources that we rely on do exists.
func (s *workItemChildSuite) SetupTest() {
	s.cleanup()

	var err error

	// Create a work item link space
	createSpacePayload := CreateSpacePayload("test-space", "description")
	_, space := test.CreateSpaceCreated(s.T(), s.workItemSvc.Context, s.workItemSvc, s.spaceCtrl, createSpacePayload)
	s.userSpaceID = *space.Data.ID
	s.T().Logf("Created link space with ID: %s\n", *space.Data.ID)

	payload := CreateWorkItemType(uuid.NewV4(), *space.Data.ID)
	_, wit := test.CreateWorkitemtypeCreated(s.T(), nil, nil, s.typeCtrl, &payload)

	// Create 3 work items (bug1, bug2, and feature1)
	bug1Payload := CreateWorkItem(s.userSpaceID, *wit.Data.ID, "bug1")
	_, bug1 := test.CreateWorkitemCreated(s.T(), s.workItemSvc.Context, s.workItemSvc, s.workItemCtrl, bug1Payload)
	require.NotNil(s.T(), bug1)
	s.deleteWorkItems = append(s.deleteWorkItems, *bug1.Data.ID)
	s.bug1ID, err = strconv.ParseUint(*bug1.Data.ID, 10, 64)
	require.Nil(s.T(), err)
	s.T().Logf("Created bug1 with ID: %s\n", *bug1.Data.ID)

	bug2Payload := CreateWorkItem(s.userSpaceID, *wit.Data.ID, "bug2")
	_, bug2 := test.CreateWorkitemCreated(s.T(), s.workItemSvc.Context, s.workItemSvc, s.workItemCtrl, bug2Payload)
	require.NotNil(s.T(), bug2)
	s.deleteWorkItems = append(s.deleteWorkItems, *bug2.Data.ID)
	s.bug2ID, err = strconv.ParseUint(*bug2.Data.ID, 10, 64)
	require.Nil(s.T(), err)
	s.T().Logf("Created bug2 with ID: %s\n", *bug2.Data.ID)

	bug3Payload := CreateWorkItem(s.userSpaceID, *wit.Data.ID, "bug3")
	_, bug3 := test.CreateWorkitemCreated(s.T(), s.workItemSvc.Context, s.workItemSvc, s.workItemCtrl, bug3Payload)
	require.NotNil(s.T(), bug3)
	s.deleteWorkItems = append(s.deleteWorkItems, *bug3.Data.ID)
	s.bug3ID, err = strconv.ParseUint(*bug3.Data.ID, 10, 64)
	require.Nil(s.T(), err)
	s.T().Logf("Created bug3 with ID: %s\n", *bug3.Data.ID)

	// Create a work item link category
	createLinkCategoryPayload := CreateWorkItemLinkCategory("test-user")
	_, workItemLinkCategory := test.CreateWorkItemLinkCategoryCreated(s.T(), nil, nil, s.workItemLinkCategoryCtrl, createLinkCategoryPayload)
	require.NotNil(s.T(), workItemLinkCategory)
	//s.deleteWorkItemLinkCategories = append(s.deleteWorkItemLinkCategories, *workItemLinkCategory.Data.ID)
	s.userLinkCategoryID = *workItemLinkCategory.Data.ID
	s.T().Logf("Created link category with ID: %s\n", *workItemLinkCategory.Data.ID)

	// Create work item link type payload
	createLinkTypePayload := createParentChildWorkItemLinkType("test-bug-blocker", *wit.Data.ID, *wit.Data.ID, s.userLinkCategoryID, s.userSpaceID)
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), nil, nil, s.workItemLinkTypeCtrl, createLinkTypePayload)
	require.NotNil(s.T(), workItemLinkType)
	//s.deleteWorkItemLinkTypes = append(s.deleteWorkItemLinkTypes, *workItemLinkType.Data.ID)
	s.bugBlockerLinkTypeID = *workItemLinkType.Data.ID
	s.T().Logf("Created link type with ID: %s\n", *workItemLinkType.Data.ID)
}

// The TearDownTest method will be run after every test in the suite.
func (s *workItemChildSuite) TearDownTest() {
	s.cleanup()
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

// createParentChildWorkItemLinkType defines a work item link type
func createParentChildWorkItemLinkType(name string, sourceTypeID, targetTypeID, categoryID, spaceID uuid.UUID) *app.CreateWorkItemLinkTypePayload {
	description := "Specify that one bug blocks another one."
	lt := link.WorkItemLinkType{
		Name:           name,
		Description:    &description,
		SourceTypeID:   sourceTypeID,
		TargetTypeID:   targetTypeID,
		Topology:       link.TopologyTree,
		ForwardName:    "parent of",
		ReverseName:    "child of",
		LinkCategoryID: categoryID,
		SpaceID:        spaceID,
	}
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	payload := link.ConvertLinkTypeFromModel(reqLong, lt)
	// The create payload is required during creation. Simply copy data over.
	return &app.CreateWorkItemLinkTypePayload{
		Data: payload.Data,
	}
}

//-----------------------------------------------------------------------------
// Actual tests
//-----------------------------------------------------------------------------

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemChildren(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(workItemChildSuite))
}

func (s *workItemChildSuite) TestListChildren() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	_, workItemLink := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.workItemLinkCtrl, createPayload)
	require.NotNil(s.T(), workItemLink)

	createPayload2 := CreateWorkItemLink(s.bug1ID, s.bug3ID, s.bugBlockerLinkTypeID)
	_, workItemLink2 := test.CreateWorkItemLinkCreated(s.T(), nil, nil, s.workItemLinkCtrl, createPayload2)
	require.NotNil(s.T(), workItemLink2)

	workItemID1 := strconv.FormatUint(s.bug1ID, 10)
	_, workItemList := test.ListWorkItemChildrenOK(s.T(), nil, nil, s.workItemChildrenCtrl, workItemID1)
	assert.Equal(s.T(), 2, len(workItemList.Data))
}
