package workitem_test

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"

	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRunWorkItemRevisionRepositoryBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemRevisionRepositoryBlackBoxTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

type workItemRevisionRepositoryBlackBoxTest struct {
	gormsupport.DBTestSuite
	workitemRepository         workitem.WorkItemRepository
	workitemRevisionRepository workitem.WorkItemRevisionRepository
	clean                      func()
	testIdentity1              account.Identity
	testIdentity2              account.Identity
	testIdentity3              account.Identity
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemRevisionRepositoryBlackBoxTest) SetupSuite() {
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

func (s *workItemRevisionRepositoryBlackBoxTest) SetupTest() {
	s.workitemRepository = workitem.NewWorkItemRepository(s.DB)
	s.workitemRevisionRepository = workitem.NewWorkItemRevisionRepository(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	testIdentity1, err := testsupport.CreateTestIdentity(s.DB, "jdoe1", "test")
	require.Nil(s.T(), err)
	s.testIdentity1 = testIdentity1
	testIdentity2, err := testsupport.CreateTestIdentity(s.DB, "jdoe2", "test")
	require.Nil(s.T(), err)
	s.testIdentity2 = testIdentity2
	testIdentity3, err := testsupport.CreateTestIdentity(s.DB, "jdoe3", "test")
	require.Nil(s.T(), err)
	s.testIdentity3 = testIdentity3
}

func (s *workItemRevisionRepositoryBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *workItemRevisionRepositoryBlackBoxTest) TestStoreWorkItemRevisions() {
	// given
	// create a workitem
	workItem, err := s.workitemRepository.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.testIdentity1.ID)
	require.Nil(s.T(), err)
	// modify the workitem
	workItem.Fields[workitem.SystemTitle] = "Updated Title"
	workItem.Fields[workitem.SystemState] = workitem.SystemStateOpen
	workItem, err = s.workitemRepository.Save(
		context.Background(), *workItem, s.testIdentity2.ID)
	require.Nil(s.T(), err)
	// modify again the workitem
	workItem.Fields[workitem.SystemTitle] = "Updated Title2"
	workItem.Fields[workitem.SystemState] = workitem.SystemStateInProgress
	workItem, err = s.workitemRepository.Save(
		context.Background(), *workItem, s.testIdentity2.ID)
	require.Nil(s.T(), err)
	// delete the workitem
	err = s.workitemRepository.Delete(
		context.Background(), workItem.ID, s.testIdentity3.ID)
	require.Nil(s.T(), err)
	// when
	workitemRevisions, err := s.workitemRevisionRepository.List(context.Background(), workItem.ID)
	// then
	require.Nil(s.T(), err)
	require.Len(s.T(), workitemRevisions, 4)
	// revision 1
	workitemRevision1 := workitemRevisions[0]
	s.T().Log(fmt.Sprintf("Work item revision 1: modifier:%s type: %v version:%v fields:%v", workitemRevision1.ModifierIdentity, workitemRevision1.Type, workitemRevision1.WorkItemVersion, workitemRevision1.WorkItemFields))
	assert.Equal(s.T(), workItem.ID, strconv.FormatUint(workitemRevision1.WorkItemID, 10))
	assert.Equal(s.T(), workitem.RevisionTypeWorkItemCreate, workitemRevision1.Type)
	assert.Equal(s.T(), workItem.Type, workitemRevision1.WorkItemType)
	assert.Equal(s.T(), s.testIdentity1.ID, workitemRevision1.ModifierIdentity)
	require.NotNil(s.T(), workitemRevision1.WorkItemFields)
	assert.Equal(s.T(), "Title", workitemRevision1.WorkItemFields[workitem.SystemTitle])
	assert.Equal(s.T(), workitem.SystemStateNew, workitemRevision1.WorkItemFields[workitem.SystemState])
	// revision 2
	workitemRevision2 := workitemRevisions[1]
	s.T().Log(fmt.Sprintf("Work item revision 2: modifier:%s type: %v version:%v fields:%v", workitemRevision2.ModifierIdentity, workitemRevision2.Type, workitemRevision2.WorkItemVersion, workitemRevision2.WorkItemFields))
	assert.Equal(s.T(), workItem.ID, strconv.FormatUint(workitemRevision2.WorkItemID, 10))
	assert.Equal(s.T(), workitem.RevisionTypeWorkItemUpdate, workitemRevision2.Type)
	assert.Equal(s.T(), workItem.Type, workitemRevision2.WorkItemType)
	assert.Equal(s.T(), s.testIdentity2.ID, workitemRevision2.ModifierIdentity)
	require.NotNil(s.T(), workitemRevision2.WorkItemFields)
	assert.Equal(s.T(), "Updated Title", workitemRevision2.WorkItemFields[workitem.SystemTitle])
	assert.Equal(s.T(), workitem.SystemStateOpen, workitemRevision2.WorkItemFields[workitem.SystemState])
	// revision 3
	workitemRevision3 := workitemRevisions[2]
	s.T().Log(fmt.Sprintf("Work item revision 3: modifier:%s type: %v version:%v fields:%v", workitemRevision3.ModifierIdentity, workitemRevision3.Type, workitemRevision3.WorkItemVersion, workitemRevision3.WorkItemFields))
	assert.Equal(s.T(), workItem.ID, strconv.FormatUint(workitemRevision3.WorkItemID, 10))
	assert.Equal(s.T(), workitem.RevisionTypeWorkItemUpdate, workitemRevision3.Type)
	assert.Equal(s.T(), workItem.Type, workitemRevision3.WorkItemType)
	require.NotNil(s.T(), workitemRevision3.WorkItemFields)
	assert.Equal(s.T(), "Updated Title2", workitemRevision3.WorkItemFields[workitem.SystemTitle])
	assert.Equal(s.T(), workitem.SystemStateInProgress, workitemRevision3.WorkItemFields[workitem.SystemState])
	assert.Equal(s.T(), s.testIdentity2.ID, workitemRevision3.ModifierIdentity)
	// revision 4
	workitemRevision4 := workitemRevisions[3]
	s.T().Log(fmt.Sprintf("Work item revision 4: modifier:%s type: %v version:%v fields:%v", workitemRevision4.ModifierIdentity, workitemRevision4.Type, workitemRevision4.WorkItemVersion, workitemRevision4.WorkItemFields))
	assert.Equal(s.T(), workItem.ID, strconv.FormatUint(workitemRevision4.WorkItemID, 10))
	assert.Equal(s.T(), workitem.RevisionTypeWorkItemDelete, workitemRevision4.Type)
	assert.Equal(s.T(), workItem.Type, workitemRevision4.WorkItemType)
	assert.Equal(s.T(), s.testIdentity3.ID, workitemRevision4.ModifierIdentity)
	require.Empty(s.T(), workitemRevision4.WorkItemFields)
}
