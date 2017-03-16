package controller_test

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"
	"github.com/jinzhu/gorm"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// The workItemChildSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type workItemChildSuite struct {
	gormtestsupport.DBTestSuite

	workItemLinkTypeCtrl     *WorkItemLinkTypeController
	workItemLinkCategoryCtrl *WorkItemLinkCategoryController
	workItemLinkCtrl         *WorkItemLinkController
	workItemCtrl             *WorkitemController
	workItemRelsLinksCtrl    *WorkItemRelationshipsLinksController
	spaceCtrl                *SpaceController
	svc                      *goa.Service
	typeCtrl                 *WorkitemtypeController
	// These IDs can safely be used by all tests
	bug1ID               uint64
	bug2ID               uint64
	bug3ID               uint64
	userLinkCategoryID   uuid.UUID
	bugBlockerLinkTypeID uuid.UUID
	userSpaceID          uuid.UUID

	// Store IDs of resources that need to be removed at the beginning or end of a test
	testIdentity account.Identity
	db           *gormapplication.GormDB
	clean        func()
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemChildSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.db = gormapplication.NewGormDB(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
		return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
	}); err != nil {
		panic(err.Error())
	}

	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "test user", "test provider")
	require.Nil(s.T(), err)
	s.testIdentity = testIdentity

	priv, err := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	require.Nil(s.T(), err)

	svc := testsupport.ServiceAsUser("WorkItemLink-Service", almtoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), svc)
	s.workItemLinkCtrl = NewWorkItemLinkController(svc, s.db)
	require.NotNil(s.T(), s.workItemLinkCtrl)

	svc = testsupport.ServiceAsUser("WorkItemLinkType-Service", almtoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), svc)
	s.workItemLinkTypeCtrl = NewWorkItemLinkTypeController(svc, s.db)
	require.NotNil(s.T(), s.workItemLinkTypeCtrl)

	svc = testsupport.ServiceAsUser("WorkItemLinkCategory-Service", almtoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), svc)
	s.workItemLinkCategoryCtrl = NewWorkItemLinkCategoryController(svc, s.db)
	require.NotNil(s.T(), s.workItemLinkCategoryCtrl)

	svc = testsupport.ServiceAsUser("WorkItemType-Service", almtoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), svc)
	s.typeCtrl = NewWorkitemtypeController(svc, s.db)
	require.NotNil(s.T(), s.typeCtrl)

	svc = testsupport.ServiceAsUser("WorkItemLink-Service", almtoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), svc)
	s.workItemLinkCtrl = NewWorkItemLinkController(svc, s.db)
	require.NotNil(s.T(), s.workItemLinkCtrl)

	svc = testsupport.ServiceAsUser("WorkItemRelationshipsLinks-Service", almtoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), svc)
	s.workItemRelsLinksCtrl = NewWorkItemRelationshipsLinksController(svc, s.db)
	require.NotNil(s.T(), s.workItemRelsLinksCtrl)

	svc = testsupport.ServiceAsUser("TestWorkItem-Service", almtoken.NewManagerWithPrivateKey(priv), testIdentity)
	require.NotNil(s.T(), svc)
	s.svc = svc
	s.workItemCtrl = NewWorkitemController(svc, s.db)
	require.NotNil(s.T(), s.workItemCtrl)

	svc = testsupport.ServiceAsUser("Space-Service", almtoken.NewManagerWithPrivateKey(priv), testIdentity)
	require.NotNil(s.T(), svc)
	s.spaceCtrl = NewSpaceController(svc, s.db, wiConfiguration, &DummyResourceManager{})
	require.NotNil(s.T(), s.spaceCtrl)

}

// The SetupTest method will be run before every test in the suite.
// SetupTest ensures that none of the work item links that we will create already exist.
// It will also make sure that some resources that we rely on do exists.
func (s *workItemChildSuite) SetupTest() {
	var err error

	// Create a work item link space
	createSpacePayload := CreateSpacePayload("test-space"+uuid.NewV4().String(), "description")
	_, space := test.CreateSpaceCreated(s.T(), s.svc.Context, s.svc, s.spaceCtrl, createSpacePayload)
	s.userSpaceID = *space.Data.ID
	s.T().Logf("Created link space with ID: %s\n", *space.Data.ID)

	// Create 3 work items (bug1, bug2, and feature1)
	bug1Payload := CreateWorkItem(s.userSpaceID, workitem.SystemBug, "bug1")
	_, bug1 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.workItemCtrl, bug1Payload)
	require.NotNil(s.T(), bug1)

	s.bug1ID, err = strconv.ParseUint(*bug1.Data.ID, 10, 64)
	require.Nil(s.T(), err)
	s.T().Logf("Created bug1 with ID: %s\n", *bug1.Data.ID)

	bug2Payload := CreateWorkItem(s.userSpaceID, workitem.SystemBug, "bug2")
	_, bug2 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.workItemCtrl, bug2Payload)
	require.NotNil(s.T(), bug2)

	s.bug2ID, err = strconv.ParseUint(*bug2.Data.ID, 10, 64)
	require.Nil(s.T(), err)
	s.T().Logf("Created bug2 with ID: %s\n", *bug2.Data.ID)

	bug3Payload := CreateWorkItem(s.userSpaceID, workitem.SystemBug, "bug3")
	_, bug3 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.workItemCtrl, bug3Payload)
	require.NotNil(s.T(), bug3)

	s.bug3ID, err = strconv.ParseUint(*bug3.Data.ID, 10, 64)
	require.Nil(s.T(), err)
	s.T().Logf("Created bug3 with ID: %s\n", *bug3.Data.ID)

	// Create a work item link category
	createLinkCategoryPayload := CreateWorkItemLinkCategory("test-user" + uuid.NewV4().String())
	_, workItemLinkCategory := test.CreateWorkItemLinkCategoryCreated(s.T(), s.svc.Context, s.svc, s.workItemLinkCategoryCtrl, createLinkCategoryPayload)
	require.NotNil(s.T(), workItemLinkCategory)
	s.userLinkCategoryID = *workItemLinkCategory.Data.ID
	s.T().Logf("Created link category with ID: %s\n", *workItemLinkCategory.Data.ID)

	// Create work item link type payload
	createLinkTypePayload := createParentChildWorkItemLinkType("test-bug-blocker", workitem.SystemBug, workitem.SystemBug, s.userLinkCategoryID, s.userSpaceID)
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.workItemLinkTypeCtrl, createLinkTypePayload)
	require.NotNil(s.T(), workItemLinkType)
	s.bugBlockerLinkTypeID = *workItemLinkType.Data.ID
	s.T().Logf("Created link type with ID: %s\n", *workItemLinkType.Data.ID)
}

// The TearDownTest method will be run after every test in the suite.
func (s *workItemChildSuite) TearDownTest() {
	s.clean()
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
	pwd, err := os.Getwd()
	if err != nil {
		require.Nil(t, err)
	}
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemChildSuite{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (s *workItemChildSuite) TestListChildren() {
	createPayload := CreateWorkItemLink(s.bug1ID, s.bug2ID, s.bugBlockerLinkTypeID)
	_, workItemLink := test.CreateWorkItemLinkCreated(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload)
	require.NotNil(s.T(), workItemLink)

	createPayload2 := CreateWorkItemLink(s.bug1ID, s.bug3ID, s.bugBlockerLinkTypeID)
	_, workItemLink2 := test.CreateWorkItemLinkCreated(s.T(), s.svc.Context, s.svc, s.workItemLinkCtrl, createPayload2)
	require.NotNil(s.T(), workItemLink2)

	workItemID1 := strconv.FormatUint(s.bug1ID, 10)
	_, workItemList := test.ListWorkItemChildrenWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, workItemID1)
	fmt.Printf("\nworkItemList.Data: %#v\n", workItemList.Data)
	assert.Equal(s.T(), 2, len(workItemList.Data))
	var count int
	for _, v := range workItemList.Data {
		switch v.Attributes[workitem.SystemTitle] {
		case "bug2":
			count = count + 1
		case "bug3":
			count = count + 1
		}

	}
	assert.Equal(s.T(), 2, count)
}
