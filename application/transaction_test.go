package application_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestTransaction struct {
	gormtestsupport.DBTestSuite
	db *gormapplication.GormDB
}

func TestRunTransaction(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestTransaction{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (test *TestTransaction) SetupTest() {
	test.DBTestSuite.SetupTest()
	test.db = gormapplication.NewGormDB(test.DB)
}

func (test *TestTransaction) TestTransactionInTime() {
	// given
	computeTime := 10 * time.Second
	// then
	err := application.Transactional(test.db, func(appl application.Application) error {
		time.Sleep(computeTime)
		return nil
	})
	// then
	require.NoError(test.T(), err)
}

func (test *TestTransaction) TestTransactionOut() {
	// given
	computeTime := 6 * time.Minute
	application.SetDatabaseTransactionTimeout(5 * time.Second)
	// then
	err := application.Transactional(test.db, func(appl application.Application) error {
		time.Sleep(computeTime)
		return nil
	})
	// then
	require.Error(test.T(), err)
	assert.Contains(test.T(), err.Error(), "database transaction timeout!")
}
