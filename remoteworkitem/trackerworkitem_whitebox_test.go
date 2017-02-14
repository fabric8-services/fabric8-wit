package remoteworkitem

import (
	"context"
	"strconv"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/workitem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// a normal test function that will kick off TestSuiteTrackerWorkItems
func TestSuiteTrackerWorkItems(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TrackerWorkItemsSuite{DBTestSuite: gormsupport.NewDBTestSuite("../config.yaml")})
}

// ========== TrackerWorkItemsSuite struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
type TrackerWorkItemsSuite struct {
	gormsupport.DBTestSuite
	clean func()
}

func (s *TrackerWorkItemsSuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *TrackerWorkItemsSuite) TearDownTest() {
	s.clean()
}

func (s *TrackerWorkItemsSuite) TestCreateIdentityFromGithubTrackerWorkItem() {
	// given
	// when

}

func (s *TrackerWorkItemsSuite) TestBindExistingIdentityFromGithubTrackerWorkItem() {
	// given
	identityRepo := account.NewIdentityRepository(s.DB)
	identity := account.Identity{
		Username:     "jdoe",
		ProfileURL:   "https://api.github.com/users/jdoe",
		ProviderType: ProviderGithub,
	}
	err := identityRepo.Create(context.Background(), &identity)
	require.Nil(s.T(), err)
	trackerRepository := NewTrackerRepository(s.DB)
	tracker, err := trackerRepository.Create(context.Background(), "https://api.github.com", ProviderGithub)
	require.Nil(s.T(), err)
	// when
	remoteItemData := TrackerItemContent{
		Content: []byte(`
			{
				"title": "linking",
				"url": "http://github.com/sbose/api/testonly/1",
				"state": "closed",
				"body": "body of issue",
				"user": {
					"login": "sbose78",
					"url": "https://api.github.com/users/sbose78"
				},
				"assignee": {
					"login": "jdoe",
					"url": "https://api.github.com/users/jdoe"
				}
			}`),
		ID: "http://github.com/sbose/api/testonly/1",
	}
	trackerID, err := strconv.Atoi(tracker.ID)
	require.Nil(s.T(), err)
	workItem, err := convert(s.DB, trackerID, remoteItemData, ProviderGithub)
	// then
	require.Nil(s.T(), err)
	identities := workItem.Fields[workitem.SystemAssignees].([]interface{})
	assert.Contains(s.T(), identities, identity.ID.String())
}
