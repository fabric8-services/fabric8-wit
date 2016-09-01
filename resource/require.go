package resource

import (
	"testing"
	"os"
	"strconv"
)

const (
	UnitTest = "ALMIGHTY_RESOURCE_UNIT_TEST"
	Database = "ALMIGHTY_RESOURCE_DATABASE"
	StSkipReasonValueFalse = "Skipping test because environment variable %s evaluates to false: %s"
	StSkipReasonNotSet = "Skipping test because environment variable %s is no set."
	StSkipReasonParseError = "Unable to parse value of environment variable %s as bool: %s"
)

// Require checks if all the given environment variables ("envVars") are set
// and if one is not set it will skip the test ("t").
func Require(t *testing.T, envVars ...string) {
	for _, envVar := range envVars {
		v, isSet := os.LookupEnv(envVar);

		// If we don't explicitly opt out from unit tests
		// by specifying ALMIGHTY_RESOURCE_UNIT_TEST=0
		// we're going to run them
		if !isSet && envVar == UnitTest {
			continue
		}

		// Skip test if environment variable is not set.
		if !isSet {
			t.Skipf(StSkipReasonNotSet, envVar)
			return
		}
		// Try to convert to boolean value
		isTrue, err := strconv.ParseBool(v)
		if err != nil {
			t.Skipf(StSkipReasonParseError, envVar, v)
			return
		}

		if !isTrue {
			t.Skipf(StSkipReasonValueFalse, envVar, v)
			return
		}
	}
}


