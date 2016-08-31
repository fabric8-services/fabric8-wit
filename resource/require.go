package resource

import (
	"testing"
	"os"
)

const (
	UnitTest = "ALMIGHTY_RESOURCE_UNIT_TEST"
	Database = "ALMIGHTY_RESOURCE_DATABASE"
	StSkipReason = "Skipping unit test because environment variable %s is no set."
)

// Require checks if all the given environment variables ("envVars") are set
// and if one is not set it will skip the test ("t").
func Require(t *testing.T, envVars ...string) {
	for _, envVar := range envVars {
		if _, c := os.LookupEnv(envVar); c == false {
			t.Skipf(StSkipReason, envVar)
			return
		}
	}
}


