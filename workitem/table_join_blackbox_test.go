package workitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type tableJoinTestSuite struct {
	gormtestsupport.DBTestSuite
}

func TestTableJoinSuite(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &tableJoinTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *tableJoinTestSuite) TestIsValidate() {
	// given
	j := workitem.TableJoin{
		Active:           true,
		TableName:        "iterations",
		TableAlias:       "iter",
		On:               workitem.JoinOnJSONField(workitem.SystemIteration, "iter.ID"),
		PrefixActivators: []string{"iteration."},
	}
	s.T().Run("valid", func(t *testing.T) {
		//given
		j.HandledFields = []string{"name"}
		// when/then
		require.NoError(t, j.Validate(s.DB))
	})
	s.T().Run("not valid", func(t *testing.T) {
		//given
		j.HandledFields = []string{"some_field_that_definitively_does_not_exist_in_the_iterations_table"}
		// when/then
		require.Error(t, j.Validate(s.DB))
	})
}
