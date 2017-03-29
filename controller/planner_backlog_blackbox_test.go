package controller_test

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestPlannerBlacklogREST struct {
	gormtestsupport.DBTestSuite
	clean        func()
	testIdentity account.Identity
	ctx          context.Context
}

func TestRunPlannerBlacklogREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(TestPlannerBlacklogREST))
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (rest *TestPlannerBlacklogREST) SetupSuite() {
	rest.DBTestSuite.SetupSuite()
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if err := models.Transactional(rest.DB, func(tx *gorm.DB) error {
		rest.ctx = migration.NewMigrationContext(context.Background())
		return migration.PopulateCommonTypes(rest.ctx, tx, workitem.NewWorkItemTypeRepository(tx))
	}); err != nil {
		panic(err.Error())
	}
}

func (rest *TestPlannerBlacklogREST) SetupTest() {
	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "test user", "test provider")
	require.Nil(rest.T(), err)
	rest.testIdentity = testIdentity

}

func (rest *TestPlannerBlacklogREST) TearDownTest() {
	//rest.clean()
}

func (rest *TestPlannerBlacklogREST) UnSecuredController() (*goa.Service, *PlannerBacklogController) {
	svc := goa.New("PlannerBlacklog-Service")
	return svc, NewPlannerBacklogController(svc, gormapplication.NewGormDB(rest.DB), rest.Configuration)
}

func (rest *TestPlannerBlacklogREST) setupPlannerBacklogWorkItems() (testSpace *space.Space, parentIteration, childIteration *iteration.Iteration) {
	application.Transactional(gormapplication.NewGormDB(rest.DB), func(app application.Application) error {
		spacesRepo := app.Spaces()
		testSpace = &space.Space{
			Name: "PlannerBacklogWorkItems-" + uuid.NewV4().String(),
		}
		_, err := spacesRepo.Create(rest.ctx, testSpace)
		require.Nil(rest.T(), err)
		require.NotNil(rest.T(), testSpace.ID)
		rest.T().Log("Created space with ID=", testSpace.ID)
		workitemTypesRepo := app.WorkItemTypes()
		//extended := workitem.SystemBug
		workitemType, err := workitemTypesRepo.Create(rest.ctx, testSpace.ID, nil, &workitem.SystemPlannerItem, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
		// workitemType, err = workitemTypesRepo.Create(rest.ctx, testSpace.ID,
		// 	&workitem.SystemPlannerItem, nil, "foo", nil, "foo",
		// 	map[string]workitem.FieldDefinition{
		// 		"aListType": {
		// 			Label:       "some list type",
		// 			Description: "description for 'some list type'",
		// 			Type: workitem.EnumType{
		// 				BaseType:   workitem.SimpleType{Kind: workitem.KindString},
		// 				SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
		// 				Values:     []interface{}{"open", "done", "closed"},
		// 			},
		// 			Required: true,
		// 		},
		// 	})
		require.Nil(rest.T(), err)
		rest.T().Log("Created workitem type with ID=", workitemType.ID)

		iterationsRepo := app.Iterations()
		parentIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: testSpace.ID,
			State:   iteration.IterationStateNew,
		}
		iterationsRepo.Create(rest.ctx, parentIteration)
		rest.T().Log("Created parent iteration with ID=", parentIteration.ID)

		childIteration = &iteration.Iteration{
			Name:    "Child Iteration",
			SpaceID: testSpace.ID,
			Path:    append(parentIteration.Path, parentIteration.ID),
			State:   iteration.IterationStateStart,
		}
		iterationsRepo.Create(rest.ctx, childIteration)
		rest.T().Log("Created child iteration with ID=", childIteration.ID)

		fields := map[string]interface{}{
			workitem.SystemTitle:     "parentIteration Test",
			workitem.SystemState:     "new",
			workitem.SystemIteration: parentIteration.ID.String(),
		}
		app.WorkItems().Create(rest.ctx, testSpace.ID, workitemType.ID, fields, rest.testIdentity.ID)

		fields2 := map[string]interface{}{
			workitem.SystemTitle:     "childIteration Test",
			workitem.SystemState:     "closed",
			workitem.SystemIteration: childIteration.ID.String(),
		}
		app.WorkItems().Create(rest.ctx, testSpace.ID, workitemType.ID, fields2, rest.testIdentity.ID)

		return nil
	})
	return
}

func assertPlannerBacklogWorkItems(t *testing.T, workitems *app.WorkItemList, testSpace *space.Space, parentIteration *iteration.Iteration) {
	// Two iteration have to be found
	require.NotNil(t, workitems)
	assert.Len(t, workitems.Data, 1)
	for _, workItem := range workitems.Data {
		assert.Equal(t, "parentIteration Test", workItem.Attributes[workitem.SystemTitle])
		assert.Equal(t, testSpace.ID.String(), workItem.Relationships.Space.Data.ID.String())
		assert.Equal(t, "parentIteration Test", workItem.Attributes[workitem.SystemTitle])
		assert.Equal(t, "new", workItem.Attributes[workitem.SystemState])
		assert.Equal(t, parentIteration.ID.String(), *workItem.Relationships.Iteration.Data.ID)
	}
}

func generateWorkitemsTag(workitems *app.WorkItemList) string {
	entities := make([]app.ConditionalResponseEntity, len(workitems.Data))
	for i, wi := range workitems.Data {
		entities[i] = workitem.WorkItem{
			ID:      *wi.ID,
			Version: wi.Attributes["version"].(int),
			Fields: map[string]interface{}{
				workitem.SystemUpdatedAt: wi.Attributes[workitem.SystemUpdatedAt],
			},
		}
	}
	return app.GenerateEntitiesTag(entities)
}

func (rest *TestPlannerBlacklogREST) TestListPlannerBacklogWorkItemsOK() {
	// given
	testSpace, parentIteration, _ := rest.setupPlannerBacklogWorkItems()
	svc, ctrl := rest.UnSecuredController()
	// when
	offset := "0"
	filter := ""
	limit := -1
	res, workitems := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, testSpace.ID.String(), &filter, nil, nil, nil, &limit, &offset, nil, nil)
	// then
	assertPlannerBacklogWorkItems(rest.T(), workitems, testSpace, parentIteration)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBlacklogREST) TestListPlannerBacklogWorkItemsOkUsingExpiredIfModifiedSinceHeader() {
	// given
	testSpace, parentIteration, _ := rest.setupPlannerBacklogWorkItems()
	svc, ctrl := rest.UnSecuredController()
	// when
	offset := "0"
	filter := ""
	limit := -1
	ifModifiedSince := app.ToHTTPTime(parentIteration.UpdatedAt.Add(-1 * time.Hour))
	res, workitems := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, testSpace.ID.String(), &filter, nil, nil, nil, &limit, &offset, &ifModifiedSince, nil)
	// then
	assertPlannerBacklogWorkItems(rest.T(), workitems, testSpace, parentIteration)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBlacklogREST) TestListPlannerBacklogWorkItemsOkUsingExpiredIfNoneMatchHeader() {
	// given
	testSpace, parentIteration, _ := rest.setupPlannerBacklogWorkItems()
	svc, ctrl := rest.UnSecuredController()
	// when
	offset := "0"
	filter := ""
	limit := -1
	ifNoneMatch := "foo"
	res, workitems := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, testSpace.ID.String(), &filter, nil, nil, nil, &limit, &offset, nil, &ifNoneMatch)
	// then
	assertPlannerBacklogWorkItems(rest.T(), workitems, testSpace, parentIteration)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBlacklogREST) TestListPlannerBacklogWorkItemsNotModifiedUsingIfModifiedSinceHeader() {
	// given
	testSpace, parentIteration, _ := rest.setupPlannerBacklogWorkItems()
	svc, ctrl := rest.UnSecuredController()
	// when
	offset := "0"
	filter := ""
	limit := -1
	ifModifiedSince := app.ToHTTPTime(parentIteration.UpdatedAt)
	res := test.ListPlannerBacklogNotModified(rest.T(), svc.Context, svc, ctrl, testSpace.ID.String(), &filter, nil, nil, nil, &limit, &offset, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBlacklogREST) TestListPlannerBacklogWorkItemsNotModifiedUsingIfNoneMatchHeader() {
	// given
	testSpace, _, _ := rest.setupPlannerBacklogWorkItems()
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	filter := ""
	limit := -1
	_, workitems := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, testSpace.ID.String(), &filter, nil, nil, nil, &limit, &offset, nil, nil)
	// when
	ifNoneMatch := generateWorkitemsTag(workitems)
	res := test.ListPlannerBacklogNotModified(rest.T(), svc.Context, svc, ctrl, testSpace.ID.String(), &filter, nil, nil, nil, &limit, &offset, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBlacklogREST) TestSuccessEmptyListPlannerBacklogWorkItems() {
	var spaceID uuid.UUID
	var parentIteration *iteration.Iteration
	application.Transactional(gormapplication.NewGormDB(rest.DB), func(app application.Application) error {
		iterationsRepo := app.Iterations()
		newSpace := space.Space{
			Name: "TestSuccessEmptyListPlannerBacklogWorkItems" + uuid.NewV4().String(),
		}
		p, err := app.Spaces().Create(rest.ctx, &newSpace)
		if err != nil {
			rest.T().Error(err)
		}
		spaceID = p.ID
		parentIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: spaceID,
			State:   iteration.IterationStateNew,
		}
		iterationsRepo.Create(rest.ctx, parentIteration)

		fields := map[string]interface{}{
			workitem.SystemTitle:     "parentIteration Test",
			workitem.SystemState:     "new",
			workitem.SystemIteration: parentIteration.ID.String(),
		}
		app.WorkItems().Create(rest.ctx, spaceID, workitem.SystemPlannerItem, fields, rest.testIdentity.ID)

		return nil
	})

	svc, ctrl := rest.UnSecuredController()

	offset := "0"
	filter := ""
	limit := -1
	_, workitems := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, spaceID.String(), &filter, nil, nil, nil, &limit, &offset, nil, nil)
	// The list has to be empty
	assert.Len(rest.T(), workitems.Data, 0)
}

func (rest *TestPlannerBlacklogREST) TestFailListPlannerBacklogByMissingSpace() {
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	filter := ""
	limit := 2
	test.ListPlannerBacklogNotFound(rest.T(), svc.Context, svc, ctrl, "xxxxx", &filter, nil, nil, nil, &limit, &offset, nil, nil)
}
