package controller_test

import (
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestNamedWorkItemsSuite struct {
	gormtestsupport.DBTestSuite
}

func TestNamedWorkItems(t *testing.T) {
	suite.Run(t, &TestNamedWorkItemsSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *TestNamedWorkItemsSuite) TestShowNamedWorkItems() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
	svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
	namedWorkItemsCtrl := NewNamedWorkItemsController(svc, s.GormDB)

	s.T().Run("ok", func(t *testing.T) {
		// when
		res := test.ShowNamedWorkItemsTemporaryRedirect(t, svc.Context, svc, namedWorkItemsCtrl, fxt.Identities[0].Username, fxt.Spaces[0].Name, fxt.WorkItems[0].Number)
		// then
		require.NotNil(t, res.Header().Get("Location"))
		assert.True(t, strings.HasSuffix(res.Header().Get("Location"), "/workitems/"+fxt.WorkItems[0].ID.String()))
	})
	s.T().Run("not found", func(t *testing.T) {
		// given
		spaceName := uuid.NewV4().String()
		username := uuid.NewV4().String()
		wiNumber := 0
		// when
		test.ShowNamedWorkItemsNotFound(t, svc.Context, svc, namedWorkItemsCtrl, username, spaceName, wiNumber)
	})
}
