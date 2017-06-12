package controller_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"path/filepath"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/stretchr/testify/require"
)

var updateGoldenFiles = flag.Bool("update", false, "when set, rewrite the golden files")

// compareWithGolden compares the actual object against the one from a golden
// file. The comparison is done by marshalling the output to JSON and comparing
// on string level If the comparison fails, the given test will fail. If the
// -update flag is given, that golden file is overwritten with the current
// actual object. When adding new tests you first must run them with the -update
// flag in order to create an initial golden version.
func compareWithGolden(t *testing.T, goldenFile string, actualObj interface{}) {
	absPath, err := filepath.Abs(goldenFile)
	require.Nil(t, err)
	actual, err := json.MarshalIndent(actualObj, "", "  ")
	require.Nil(t, err)
	if *updateGoldenFiles {
		err = ioutil.WriteFile(absPath, actual, os.ModePerm)
		require.Nil(t, err, "failed to update golden file: %s", absPath)
	}
	expected, err := ioutil.ReadFile(absPath)
	require.Nil(t, err, "failed to read golden file: %s", absPath)
	expectedStr := string(expected)
	actualStr := string(actual)
	if expectedStr != actualStr {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(expectedStr, actualStr, false)
		fmt.Println(dmp.DiffPrettyText(diffs))
		t.Log(dmp.DiffPrettyText(diffs))
	}
	require.Equal(t, expectedStr, actualStr, "mismatch of actual output and golden-file %s", absPath)

}
