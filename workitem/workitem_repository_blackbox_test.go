package workitem_test

import (
	"testing"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/workitem"
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
	suite.Run(t, &workItemRepoBlackBoxTest{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

func (s *workItemRepoBlackBoxTest) SetupTest() {
	s.repo = workitem.NewWorkItemRepository(s.DB)
}

func (s *workItemRepoBlackBoxTest) TestFailDeleteZeroID() {
	defer gormsupport.DeleteCreatedEntities(s.DB)()

	// Create at least 1 item to avoid RowsEffectedCheck
	_, err := s.repo.Create(
		context.Background(), "system_bug",
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, "xx")

	if err != nil {
		s.T().Error("Could not create workitem", err)
	}

	err = s.repo.Delete(context.Background(), "0")
	require.IsType(s.T(), errors.NotFoundError{}, err)
}

func (s *workItemRepoBlackBoxTest) TestFailSaveZeroID() {
	defer gormsupport.DeleteCreatedEntities(s.DB)()

	// Create at least 1 item to avoid RowsEffectedCheck
	wi, err := s.repo.Create(
		context.Background(), "system_bug",
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, "xx")

	if err != nil {
		s.T().Error("Could not create workitem", err)
	}
	wi.ID = "0"

	_, err = s.repo.Save(context.Background(), *wi)
	require.IsType(s.T(), errors.NotFoundError{}, err)
}

func (s *workItemRepoBlackBoxTest) TestFaiLoadZeroID() {
	defer gormsupport.DeleteCreatedEntities(s.DB)()

	// Create at least 1 item to avoid RowsEffectedCheck
	_, err := s.repo.Create(
		context.Background(), "system_bug",
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, "xx")

	if err != nil {
		s.T().Error("Could not create workitem", err)
	}

	_, err = s.repo.Load(context.Background(), "0")
	require.IsType(s.T(), errors.NotFoundError{}, err)
}

func (s *workItemRepoBlackBoxTest) TestSaveAssignees() {
	defer gormsupport.DeleteCreatedEntities(s.DB)()

	wi, err := s.repo.Create(
		context.Background(), "system_bug",
		map[string]interface{}{
			workitem.SystemTitle:     "Title",
			workitem.SystemState:     workitem.SystemStateNew,
			workitem.SystemAssignees: []string{"A", "B"},
		}, "xx")

	if err != nil {
		s.T().Error("Could not create workitem", err)
	}

	wi, err = s.repo.Load(context.Background(), wi.ID)

	assert.Equal(s.T(), "A", wi.Fields[workitem.SystemAssignees].([]interface{})[0])
}

func (s *workItemRepoBlackBoxTest) TestSaveForUnchangedCreatedDate() {
	defer gormsupport.DeleteCreatedEntities(s.DB)()

	wi, err := s.repo.Create(
		context.Background(), "system_bug",
		map[string]interface{}{
			workitem.SystemTitle: "Title",
			workitem.SystemState: workitem.SystemStateNew,
		}, "xx")

	if err != nil {
		s.T().Error("Could not create workitem", err)
	}

	wi, err = s.repo.Load(context.Background(), wi.ID)

	wiNew, err := s.repo.Save(context.Background(), *wi)

	assert.Equal(s.T(), wi.Fields[workitem.SystemCreatedAt], wiNew.Fields[workitem.SystemCreatedAt])
}
