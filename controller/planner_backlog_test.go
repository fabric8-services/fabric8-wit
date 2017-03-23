package controller_test

import (
	"crypto/rsa"
	"fmt"
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	config "github.com/almighty/almighty-core/configuration"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/suite"
)

var wibConfig *config.ConfigurationData

func init() {
	var err error
	wibConfig, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

type TestPlannerBlacklogREST struct {
	suite.Suite
	db           *gorm.DB
	clean        func()
	priKey       *rsa.PrivateKey
	testIdentity account.Identity
	ctx          context.Context
}

func TestRunPlannerBlacklogREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(TestPlannerBlacklogREST))
}

func (rest *TestPlannerBlacklogREST) SetupTest() {
	var err error
	rest.db, err = gorm.Open("postgres", wibConfig.GetPostgresConfigString())
	if err != nil {
		panic("Failed to connect database: " + err.Error())
	}
	rest.priKey, _ = almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if wibConfig.GetPopulateCommonTypes() {
		if err := models.Transactional(rest.db, func(tx *gorm.DB) error {
			rest.ctx = migration.NewMigrationContext(context.Background())
			return migration.PopulateCommonTypes(rest.ctx, tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
	rest.clean = cleaner.DeleteCreatedEntities(rest.db)

	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(rest.db, "test user", "test provider")
	require.Nil(rest.T(), err)
	rest.testIdentity = testIdentity
}

func (rest *TestPlannerBlacklogREST) TearDownTest() {
	rest.clean()
	if rest.db != nil {
		rest.db.Close()
	}
}

func (rest *TestPlannerBlacklogREST) UnSecuredController() (*goa.Service, *PlannerBacklogController) {
	svc := goa.New("PlannerBlacklog-Service")
	return svc, NewPlannerBacklogController(svc, gormapplication.NewGormDB(rest.db))
}

func (rest *TestPlannerBlacklogREST) TestSuccessListPlannerBacklogWorkItems() {
	t := rest.T()
	resource.Require(t, resource.Database)

	var fatherIteration, childIteration *iteration.Iteration
	application.Transactional(gormapplication.NewGormDB(rest.db), func(app application.Application) error {
		repo := app.Iterations()

		fatherIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: space.SystemSpace,
			State:   iteration.IterationStateNew,
		}
		repo.Create(rest.ctx, fatherIteration)

		childIteration = &iteration.Iteration{
			Name:    "Child Iteration",
			SpaceID: space.SystemSpace,
			Path:    append(fatherIteration.Path, fatherIteration.ID),
			State:   iteration.IterationStateStart,
		}
		repo.Create(rest.ctx, childIteration)

		fields := map[string]interface{}{
			workitem.SystemTitle:     "fatherIteration Test",
			workitem.SystemState:     "new",
			workitem.SystemIteration: fatherIteration.ID.String(),
		}
		app.WorkItems().Create(rest.ctx, space.SystemSpace, workitem.SystemBug, fields, rest.testIdentity.ID)

		fields2 := map[string]interface{}{
			workitem.SystemTitle:     "childIteration Test",
			workitem.SystemState:     "closed",
			workitem.SystemIteration: childIteration.ID.String(),
		}
		app.WorkItems().Create(rest.ctx, space.SystemSpace, workitem.SystemPlannerItem, fields2, rest.testIdentity.ID)

		return nil
	})

	svc, ctrl := rest.UnSecuredController()

	offset := "0"
	filter := ""
	limit := -1
	_, cs := test.ListPlannerBacklogOK(t, svc.Context, svc, ctrl, space.SystemSpace.String(), &filter, nil, nil, nil, &limit, &offset)

	// Two iteration have to be found
	assert.Len(t, cs.Data, 1)

	for _, workItem := range cs.Data {
		assert.Equal(t, "fatherIteration Test", workItem.Attributes[workitem.SystemTitle])
		assert.Equal(t, space.SystemSpace.String(), workItem.Relationships.Space.Data.ID.String())
		assert.Equal(t, "fatherIteration Test", workItem.Attributes[workitem.SystemTitle])
		assert.Equal(t, "new", workItem.Attributes[workitem.SystemState])
		assert.Equal(t, fatherIteration.ID.String(), *workItem.Relationships.Iteration.Data.ID)
	}
}

func (rest *TestPlannerBlacklogREST) TestFailListPlannerBacklogByMissingSpace() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	filter := ""
	limit := 2
	test.ListPlannerBacklogNotFound(t, svc.Context, svc, ctrl, "xxxxx", &filter, nil, nil, nil, &limit, &offset)
}
