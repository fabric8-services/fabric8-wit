package workitem_test

import (
	"os"
	"testing"

	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type workItemRepoBlackBoxTest struct {
	gormsupport.DBTestSuite
	repo workitem.WorkItemRepository
}

func TestRunWorkTypeRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &workItemRepoBlackBoxTest{DBTestSuite: gormsupport.NewDBTestSuite("../" + config.GetDefaultConfigurationFile())})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
func (s *workItemRepoBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if _, c := os.LookupEnv(resource.Database); c != false {
		if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
}

func (s *workItemRepoBlackBoxTest) SetupTest() {
	s.repo = workitem.NewWorkItemRepository(s.DB)
}

func (s *workItemRepoBlackBoxTest) TestFailDeleteZeroID() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	// Create at least 1 item to avoid RowsEffectedCheck
	_, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, "xx")
	require.Nil(s.T(), err, "Could not create work item")

	err = s.repo.Delete(context.Background(), "0")
	require.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
}

func (s *workItemRepoBlackBoxTest) TestFailSaveZeroID() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	// Create at least 1 item to avoid RowsEffectedCheck
	wi, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, "xx")
	require.Nil(s.T(), err, "Could not create workitem")
	wi.ID = "0"
	_, err = s.repo.Save(context.Background(), *wi)
	require.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
}

func (s *workItemRepoBlackBoxTest) TestFaiLoadZeroID() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	// Create at least 1 item to avoid RowsEffectedCheck
	_, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, "xx")
	require.Nil(s.T(), err, "Could not create workitem")

	_, err = s.repo.Load(context.Background(), "0")
	require.IsType(s.T(), errors.NotFoundError{}, errs.Cause(err))
}

func (s *workItemRepoBlackBoxTest) TestSaveAssignees() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	wi, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:     "Title",
			workitem.SystemState:     workitem.SystemStateNew,
			workitem.SystemAssignees: []string{"A", "B"},
		}, "xx")
	require.Nil(s.T(), err, "Could not create workitem")

	wi, err = s.repo.Load(context.Background(), wi.ID)
	require.Nil(s.T(), err)

	assert.Equal(s.T(), "A", wi.Fields[workitem.SystemAssignees].([]interface{})[0])
}

func (s *workItemRepoBlackBoxTest) TestSaveForUnchangedCreatedDate() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	wi, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, "xx")
	require.Nil(s.T(), err, "Could not create workitem")

	wi, err = s.repo.Load(context.Background(), wi.ID)
	require.Nil(s.T(), err)

	wiNew, err := s.repo.Save(context.Background(), *wi)
	require.Nil(s.T(), err)
	assert.Equal(s.T(), wi.Fields[workitem.SystemCreatedAt], wiNew.Fields[workitem.SystemCreatedAt])
}

func (s *workItemRepoBlackBoxTest) TestCreateWorkItemWithDescriptionNoMarkup() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	wi, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "Title",
			workitem.SystemDescription: rendering.NewMarkupContentFromLegacy("Description"),
			workitem.SystemState:       workitem.SystemStateNew,
		}, "xx")
	require.Nil(s.T(), err, "Could not create workitem")

	wi, err = s.repo.Load(context.Background(), wi.ID)
	require.Nil(s.T(), err)
	// app.WorkItem does not contain the markup associated with the description (yet)
	assert.Equal(s.T(), rendering.NewMarkupContentFromLegacy("Description"), wi.Fields[workitem.SystemDescription])
}

func (s *workItemRepoBlackBoxTest) TestCreateWorkItemWithDescriptionMarkup() {
	defer cleaner.DeleteCreatedEntities(s.DB)()
	wi, err := s.repo.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "Title",
			workitem.SystemDescription: rendering.NewMarkupContent("Description", rendering.SystemMarkupMarkdown),
			workitem.SystemState:       workitem.SystemStateNew,
		}, "xx")
	require.Nil(s.T(), err, "Could not create workitem")
	wi, err = s.repo.Load(context.Background(), wi.ID)
	require.Nil(s.T(), err)
	// app.WorkItem does not contain the markup associated with the description (yet)
	assert.Equal(s.T(), rendering.NewMarkupContent("Description", rendering.SystemMarkupMarkdown), wi.Fields[workitem.SystemDescription])
}

// TestTypeChangeIsNotProhibitedOnDBLayer tests that you can change the type of
// a work item. NOTE: This functionality only works on the DB layer and is not
// exposed to REST.
func (s *workItemRepoBlackBoxTest) TestTypeChangeIsNotProhibitedOnDBLayer() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	// Create at least 1 item to avoid RowsAffectedCheck
	wi, err := s.repo.Create(
		context.Background(), "bug",
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, "xx")

	require.Nil(s.T(), err)

	wi.Type = "feature"

	newWi, err := s.repo.Save(context.Background(), *wi)
	require.Nil(s.T(), err)
	require.Equal(s.T(), "feature", newWi.Type)
}
