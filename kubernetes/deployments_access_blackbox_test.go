package kubernetes_test

import (
	"testing"

	"github.com/dnaeon/go-vcr/recorder"
	"github.com/stretchr/testify/require"
)

func TestCanGetSpace(t *testing.T) {
	testCases := []struct {
		testName       string
		cassetteName   string
		expectedResult bool
		shouldFail     bool
	}{
		{
			testName:       "Basic",
			cassetteName:   "can-i",
			expectedResult: true,
		},
		{
			testName:       "No Builds",
			cassetteName:   "can-i-no-builds",
			expectedResult: false,
		},
		{
			testName:       "No Deployment Config",
			cassetteName:   "can-i-no-dc",
			expectedResult: false,
		},
		{
			testName:     "Missing Status",
			cassetteName: "can-i-no-status",
			shouldFail:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			result, err := kc.CanGetSpace()
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
				require.Equal(t, testCase.expectedResult, result, "Expected different authorization result")
			}
		})
	}
}

func TestCanGetApplication(t *testing.T) {
	testCases := []struct {
		testName       string
		cassetteName   string
		expectedResult bool
		shouldFail     bool
	}{
		{
			testName:       "Basic",
			cassetteName:   "can-i",
			expectedResult: true,
		},
		{
			testName:       "No Builds",
			cassetteName:   "can-i-no-builds",
			expectedResult: false,
		},
		{
			testName:       "No Deployment Config",
			cassetteName:   "can-i-no-dc",
			expectedResult: false,
		},
		{
			testName:     "Missing Status",
			cassetteName: "can-i-no-status",
			shouldFail:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			result, err := kc.CanGetApplication()
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
				require.Equal(t, testCase.expectedResult, result, "Expected different authorization result")
			}
		})
	}
}

func TestCanGetDeployment(t *testing.T) {
	testCases := []struct {
		testName       string
		cassetteName   string
		envName        string
		expectedResult bool
		shouldFail     bool
	}{
		{
			testName:       "Basic",
			envName:        "run",
			cassetteName:   "can-i",
			expectedResult: true,
		},
		{
			testName:       "No Builds",
			envName:        "run",
			cassetteName:   "can-i-no-builds",
			expectedResult: false,
		},
		{
			testName:       "No Deployment Config",
			envName:        "run",
			cassetteName:   "can-i-no-dc",
			expectedResult: false,
		},
		{
			testName:     "Missing Status",
			envName:      "run",
			cassetteName: "can-i-no-status",
			shouldFail:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			result, err := kc.CanGetDeployment(testCase.envName)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
				require.Equal(t, testCase.expectedResult, result, "Expected different authorization result")
			}
		})
	}
}

func TestCanScaleDeployment(t *testing.T) {
	testCases := []struct {
		testName       string
		cassetteName   string
		envName        string
		expectedResult bool
		shouldFail     bool
	}{
		{
			testName:       "Basic",
			envName:        "run",
			cassetteName:   "can-i",
			expectedResult: true,
		},
		{
			testName:       "No Builds",
			envName:        "run",
			cassetteName:   "can-i-no-builds",
			expectedResult: false,
		},
		{
			testName:       "No Deployment Config",
			envName:        "run",
			cassetteName:   "can-i-no-dc",
			expectedResult: false,
		},
		{
			testName:     "Missing Status",
			envName:      "run",
			cassetteName: "can-i-no-status",
			shouldFail:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			result, err := kc.CanScaleDeployment(testCase.envName)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
				require.Equal(t, testCase.expectedResult, result, "Expected different authorization result")
			}
		})
	}
}

func TestCanDeleteDeployment(t *testing.T) {
	testCases := []struct {
		testName       string
		cassetteName   string
		envName        string
		expectedResult bool
		shouldFail     bool
	}{
		{
			testName:       "Basic",
			envName:        "run",
			cassetteName:   "can-i",
			expectedResult: true,
		},
		{
			testName:       "No Builds",
			envName:        "run",
			cassetteName:   "can-i-no-builds",
			expectedResult: false,
		},
		{
			testName:       "No Deployment Config",
			envName:        "run",
			cassetteName:   "can-i-no-dc",
			expectedResult: false,
		},
		{
			testName:     "Missing Status",
			envName:      "run",
			cassetteName: "can-i-no-status",
			shouldFail:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			result, err := kc.CanDeleteDeployment(testCase.envName)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
				require.Equal(t, testCase.expectedResult, result, "Expected different authorization result")
			}
		})
	}
}

func TestCanGetEnvironments(t *testing.T) {
	testCases := []struct {
		testName       string
		cassetteName   string
		expectedResult bool
		shouldFail     bool
	}{
		{
			testName:       "Basic",
			cassetteName:   "can-i",
			expectedResult: true,
		},
		{
			testName:       "No Resource Quotas",
			cassetteName:   "can-i-no-quotas",
			expectedResult: false,
		},
		{
			testName:     "Missing Status",
			cassetteName: "can-i-no-status",
			shouldFail:   true,
		},
		{
			testName:     "Missing Rules",
			cassetteName: "can-i-no-rules",
			shouldFail:   true,
		},
		{
			testName:     "Bad Rule",
			cassetteName: "can-i-bad-rule",
			shouldFail:   true,
		},
		{
			testName:       "Bad Verbs",
			cassetteName:   "can-i-bad-verbs",
			expectedResult: true, // Skips bad verbs
		},
		{
			testName:       "Bad Resource",
			cassetteName:   "can-i-bad-resource",
			expectedResult: true, // Skips bad resources
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			result, err := kc.CanGetEnvironments()
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
				require.Equal(t, testCase.expectedResult, result, "Expected different authorization result")
			}
		})
	}
}

func TestCanGetDeploymentStats(t *testing.T) {
	testCases := []struct {
		testName       string
		cassetteName   string
		envName        string
		expectedResult bool
		shouldFail     bool
	}{
		{
			testName:       "Basic",
			envName:        "run",
			cassetteName:   "can-i",
			expectedResult: true,
		},
		{
			testName:       "No Builds",
			envName:        "run",
			cassetteName:   "can-i-no-builds",
			expectedResult: false,
		},
		{
			testName:       "No Deployment Config",
			envName:        "run",
			cassetteName:   "can-i-no-dc",
			expectedResult: false,
		},
		{
			testName:     "Missing Status",
			envName:      "run",
			cassetteName: "can-i-no-status",
			shouldFail:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			result, err := kc.CanGetDeploymentStats(testCase.envName)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
				require.Equal(t, testCase.expectedResult, result, "Expected different authorization result")
			}
		})
	}
}

func TestCanGetDeploymentStatSeries(t *testing.T) {
	testCases := []struct {
		testName       string
		cassetteName   string
		envName        string
		expectedResult bool
		shouldFail     bool
	}{
		{
			testName:       "Basic",
			envName:        "run",
			cassetteName:   "can-i",
			expectedResult: true,
		},
		{
			testName:       "No Builds",
			envName:        "run",
			cassetteName:   "can-i-no-builds",
			expectedResult: false,
		},
		{
			testName:       "No Deployment Config",
			envName:        "run",
			cassetteName:   "can-i-no-dc",
			expectedResult: false,
		},
		{
			testName:     "Missing Status",
			envName:      "run",
			cassetteName: "can-i-no-status",
			shouldFail:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			result, err := kc.CanGetDeploymentStatSeries(testCase.envName)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
				require.Equal(t, testCase.expectedResult, result, "Expected different authorization result")
			}
		})
	}
}
