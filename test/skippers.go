package test

import (
	"testing"
	"os"
	"fmt"
)

const (
	EnvVarRunUnitTests = "ALMIGHTY_RUN_UNIT_TESTS"
	EnvVarRunIntegrationTests = "ALMIGHTY_RUN_INTEGRATION_TESTS"
	stSkipReason = "Skipping unit test because environment variable %s is no set."
)

// SkiptTestIfNotSet skips the test "t" if the given environment variable is not set.
func SkiptTestIfNotSet(t *testing.T, environmentVar string) {
	if _, c := os.LookupEnv(environmentVar); c == false {
		t.Skipf(stSkipReason, environmentVar)
	}
}

// SkipIfNotUnitTest skips the test "t" if the environment variable "ALMIGHTY_RUN_UNIT_TESTS" is not set.
func SkiptTestIfNotUnitTest(t *testing.T) {
	SkiptTestIfNotSet(t, EnvVarRunUnitTests)
}

// SkipIfNotIntegrationTest skips the test "t" if the environment variable "ALMIGHTY_RUN_INTEGRATION_TESTS" is not set.
func SkipTestIfNotIntegrationTest(t *testing.T) {
	SkiptTestIfNotSet(t, EnvVarRunIntegrationTests)
}


