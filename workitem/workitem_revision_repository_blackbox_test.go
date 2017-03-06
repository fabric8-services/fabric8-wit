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

func TestRunRevisionRepositoryBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemRevisionRepositoryBlackBoxTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

type workItemRevisionRepositoryBlackBoxTest struct {
	gormsupport.DBTestSuite
	repository         workitem.WorkItemRepository
	revisionRepository workitem.RevisionRepository
	clean              func()
	testIdentity1      account.Identity
	testIdentity2      account.Identity
	testIdentity3      account.Identity
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
	s.repository = workitem.NewWorkItemRepository(s.DB)
	s.revisionRepository = workitem.NewRevisionRepository(s.DB)
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

func (s *workItemRevisionRepositoryBlackBoxTest) TestStoreRevisions() {
	// given
	// create a workitem
	workItem, err := s.repository.Create(
		context.Background(), workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, s.testIdentity1.ID)
	require.Nil(s.T(), err)
	// modify the workitem
	workItem.Fields[workitem.SystemTitle] = "Updated Title"
	workItem.Fields[workitem.SystemState] = workitem.SystemStateOpen
	workItem, err = s.repository.Save(
		context.Background(), *workItem, s.testIdentity2.ID)
	require.Nil(s.T(), err)
	// modify again the workitem
	workItem.Fields[workitem.SystemTitle] = "Updated Title2"
	workItem.Fields[workitem.SystemState] = workitem.SystemStateInProgress
	workItem, err = s.repository.Save(
		context.Background(), *workItem, s.testIdentity2.ID)
	require.Nil(s.T(), err)
	// delete the workitem
	err = s.repository.Delete(
		context.Background(), workItem.ID, s.testIdentity3.ID)
	require.Nil(s.T(), err)
	// when
	revisions, err := s.revisionRepository.List(context.Background(), workItem.ID)
	// then
	require.Nil(s.T(), err)
	require.Len(s.T(), revisions, 4)
	// revision 1
	revision1 := revisions[0]
	s.T().Log(fmt.Sprintf("Work item revision 1: modifier:%s type: %v version:%v fields:%v", revision1.ModifierIdentity, revision1.Type, revision1.WorkItemVersion, revision1.WorkItemFields))
	assert.Equal(s.T(), workItem.ID, strconv.FormatUint(revision1.WorkItemID, 10))
	assert.Equal(s.T(), workitem.RevisionTypeCreate, revision1.Type)
	assert.Equal(s.T(), workItem.Type, revision1.WorkItemTypeID)
	assert.Equal(s.T(), s.testIdentity1.ID, revision1.ModifierIdentity)
	require.NotNil(s.T(), revision1.WorkItemFields)
	assert.Equal(s.T(), "Title", revision1.WorkItemFields[workitem.SystemTitle])
	assert.Equal(s.T(), workitem.SystemStateNew, revision1.WorkItemFields[workitem.SystemState])
	// revision 2
	revision2 := revisions[1]
	s.T().Log(fmt.Sprintf("Work item revision 2: modifier:%s type: %v version:%v fields:%v", revision2.ModifierIdentity, revision2.Type, revision2.WorkItemVersion, revision2.WorkItemFields))
	assert.Equal(s.T(), workItem.ID, strconv.FormatUint(revision2.WorkItemID, 10))
	assert.Equal(s.T(), workitem.RevisionTypeUpdate, revision2.Type)
	assert.Equal(s.T(), workItem.Type, revision2.WorkItemTypeID)
	assert.Equal(s.T(), s.testIdentity2.ID, revision2.ModifierIdentity)
	require.NotNil(s.T(), revision2.WorkItemFields)
	assert.Equal(s.T(), "Updated Title", revision2.WorkItemFields[workitem.SystemTitle])
	assert.Equal(s.T(), workitem.SystemStateOpen, revision2.WorkItemFields[workitem.SystemState])
	// revision 3
	revision3 := revisions[2]
	s.T().Log(fmt.Sprintf("Work item revision 3: modifier:%s type: %v version:%v fields:%v", revision3.ModifierIdentity, revision3.Type, revision3.WorkItemVersion, revision3.WorkItemFields))
	assert.Equal(s.T(), workItem.ID, strconv.FormatUint(revision3.WorkItemID, 10))
	assert.Equal(s.T(), workitem.RevisionTypeUpdate, revision3.Type)
	assert.Equal(s.T(), workItem.Type, revision3.WorkItemTypeID)
	require.NotNil(s.T(), revision3.WorkItemFields)
	assert.Equal(s.T(), "Updated Title2", revision3.WorkItemFields[workitem.SystemTitle])
	assert.Equal(s.T(), workitem.SystemStateInProgress, revision3.WorkItemFields[workitem.SystemState])
	assert.Equal(s.T(), s.testIdentity2.ID, revision3.ModifierIdentity)
	// revision 4
	revision4 := revisions[3]
	s.T().Log(fmt.Sprintf("Work item revision 4: modifier:%s type: %v version:%v fields:%v", revision4.ModifierIdentity, revision4.Type, revision4.WorkItemVersion, revision4.WorkItemFields))
	assert.Equal(s.T(), workItem.ID, strconv.FormatUint(revision4.WorkItemID, 10))
	assert.Equal(s.T(), workitem.RevisionTypeDelete, revision4.Type)
	assert.Equal(s.T(), workItem.Type, revision4.WorkItemTypeID)
	assert.Equal(s.T(), s.testIdentity3.ID, revision4.ModifierIdentity)
	require.Empty(s.T(), revision4.WorkItemFields)
}
