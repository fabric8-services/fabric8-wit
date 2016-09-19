// Package resource is used to manage which tests shall be executed.
// Tests can specify which resources they require. If such a resource
// is not available at runtime, the test will be skipped.
// The availability of resources is determined by the presence of an
// environment variable that doesn't evaluate to false (e.g. "0", "no", "false").
// See strconv.ParseBool for more information what evaluates to false.
package resource

import (
	"os"
	"strconv"
	"testing"
)

const (
	// UnitTest refers to the name of the environment variable that is used to
	// specify that unit tests shall be run. Unless this environment variable
	// is explicitly set to evaluate to false ("0", "no", or "false"), unit
	// tests are executed all the time.
	UnitTest = "ALMIGHTY_RESOURCE_UNIT_TEST"
	// Database refers to the name of the environment variable that is used to
	// specify that test can be run that require a database.
	Database               = "ALMIGHTY_RESOURCE_DATABASE"
	stSkipReasonValueFalse = "Skipping test because environment variable %s evaluates to false: %s"
	stSkipReasonNotSet     = "Skipping test because environment variable %s is no set."
	stSkipReasonParseError = "Unable to parse value of environment variable %s as bool: %s"
)

// Require checks if all the given environment variables ("envVars") are set
// and if one is not set it will skip the test ("t"). The only exception is
// that the unit test resource is always considered to be available unless
// is is explicitly set to false (e.g. "no", "0", "false").
func Require(t *testing.T, envVars ...string) {
	for _, envVar := range envVars {
		v, isSet := os.LookupEnv(envVar)

		// If we don't explicitly opt out from unit tests
		// by specifying ALMIGHTY_RESOURCE_UNIT_TEST=0
		// we're going to run them
		if !isSet && envVar == UnitTest {
			continue
		}

		// Skip test if environment variable is not set.
		if !isSet {
			t.Skipf(stSkipReasonNotSet, envVar)
			return
		}
		// Try to convert to boolean value
		isTrue, err := strconv.ParseBool(v)
		if err != nil {
			t.Skipf(stSkipReasonParseError, envVar, v)
			return
		}

		if !isTrue {
			t.Skipf(stSkipReasonValueFalse, envVar, v)
			return
		}
	}
}
