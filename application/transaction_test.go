package application_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestTransaction struct {
	gormtestsupport.DBTestSuite
}

func TestRunTransaction(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestTransaction{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (test *TestTransaction) SetupTest() {
	test.DBTestSuite.SetupTest()
}

func (test *TestTransaction) TestTransactionInTime() {
	// given
	computeTime := 10 * time.Second
	// then
	err := application.Transactional(test.GormDB, func(appl application.Application) error {
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
	err := application.Transactional(test.GormDB, func(appl application.Application) error {
		time.Sleep(computeTime)
		return nil
	})
	// then
	require.Error(test.T(), err)
	assert.Contains(test.T(), err.Error(), "database transaction timeout")
}

func (test *TestTransaction) TestTransactionPanicAndRecoverWithStack() {
	// then
	err := application.Transactional(test.GormDB, func(appl application.Application) error {
		bar := func(a, b interface{}) {
			// This comparison while legal at compile time will cause a runtime
			// error like this: "comparing uncomparable type
			// map[string]interface {}". The transaction will panic and recover
			// but you will probably never find out where the error came from if
			// the stack is not captured in the transaction recovery. This test
			// ensures that the stack is captured.
			if a == b {
				fmt.Printf("never executed")
			}
		}
		foo := func() {
			a := map[string]interface{}{}
			b := map[string]interface{}{}
			bar(a, b)
		}
		foo()
		return nil
	})
	// then
	require.Error(test.T(), err)
	// ensure there's a proper stack trace that contains the name of this test
	require.Contains(test.T(), err.Error(), "(*TestTransaction).TestTransactionPanicAndRecoverWithStack.func1(")
}
