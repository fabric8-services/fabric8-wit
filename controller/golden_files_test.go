package controller_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/resource"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
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
	err := testableCompareWithGolden(*updateGoldenFiles, goldenFile, actualObj, false)
	if err != nil {
		t.Fatalf("%+v", err)
	}
}

// compareWithGoldenUUIDAgnostic does the same as compareWithGolden but after
// marshalling the given objects to a JSON string it replaces UUIDs in both
// strings (the golden file as well as in the actual object) before comparing
// the two strings. This should make the comparison UUID agnostic without
// loosing the locality comparison. In other words, that means we replace each
// UUID with a more generic "00000000-0000-0000-0000-000000000001",
// "00000000-0000-0000-0000-000000000002", ...,
// "00000000-0000-0000-0000-00000000000N" value.
func compareWithGoldenUUIDAgnostic(t *testing.T, goldenFile string, actualObj interface{}) {
	err := testableCompareWithGolden(*updateGoldenFiles, goldenFile, actualObj, true)
	if err != nil {
		t.Fatalf("%+v", err)
	}
}

func testableCompareWithGolden(update bool, goldenFile string, actualObj interface{}, uuidAgnostic bool) error {
	absPath, err := filepath.Abs(goldenFile)
	if err != nil {
		return errs.WithStack(err)
	}
	actual, err := json.MarshalIndent(actualObj, "", "  ")
	if err != nil {
		return errs.WithStack(err)
	}
	if update {
		tmp := string(actual)
		var err error
		// Eliminate concrete UUIDs if requested. This makes adding changes to
		// golden files much more easy in git.
		if uuidAgnostic {
			tmp, err = replaceUUIDs(tmp)
			if err != nil {
				return errs.Wrapf(err, "failed to replace UUIDs with more generic ones")
			}
		}
		err = ioutil.WriteFile(absPath, []byte(tmp), os.ModePerm)
		if err != nil {
			return errs.Wrapf(err, "failed to update golden file: %s", absPath)
		}
	}
	expected, err := ioutil.ReadFile(absPath)
	if err != nil {
		return errs.Wrapf(err, "failed to read golden file: %s", absPath)
	}

	expectedStr := string(expected)
	actualStr := string(actual)
	if uuidAgnostic {
		expectedStr, err = replaceUUIDs(expectedStr)
		if err != nil {
			return errs.Wrapf(err, "failed to replace UUIDs with more generic ones")
		}
		actualStr, err = replaceUUIDs(actualStr)
		if err != nil {
			return errs.Wrapf(err, "failed to replace UUIDs with more generic ones")
		}
	}
	if expectedStr != actualStr {
		log.Error(nil, nil, "testableCompareWithGolden: expected value %v", expectedStr)
		log.Error(nil, nil, "testableCompareWithGolden: actual value %v", actualStr)

		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(expectedStr, actualStr, false)
		log.Error(nil, nil, "testableCompareWithGolden: mismatch of actual output and golden-file %s:\n %s \n", absPath, dmp.DiffPrettyText(diffs))
		return errs.Errorf("mismatch of actual output and golden-file %s:\n %s \n", absPath, dmp.DiffPrettyText(diffs))
	}
	return nil
}

// findUUIDs returns an array of uniq UUIDs that have been found in the given
// string
func findUUIDs(str string) ([]uuid.UUID, error) {
	pattern := "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}"
	uuidRegexp, err := regexp.Compile(pattern)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to compile UUID regex pattern %s", pattern)
	}
	uniqIDs := map[uuid.UUID]struct{}{}
	var res []uuid.UUID
	for _, uuidStr := range uuidRegexp.FindAllString(str, -1) {
		ID, err := uuid.FromString(uuidStr)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to parse UUID %s", uuidStr)
		}
		_, alreadyInMap := uniqIDs[ID]
		if !alreadyInMap {
			uniqIDs[ID] = struct{}{}
			// append to array
			res = append(res, ID)
		}
	}
	return res, nil
}

// replaceUUIDs finds all UUIDs in the given string and replaces them with
// "00000000-0000-0000-0000-000000000001,
// "00000000-0000-0000-0000-000000000002", ...,
// "00000000-0000-0000-0000-00000000000N"
func replaceUUIDs(str string) (string, error) {
	replacementPattern := "00000000-0000-0000-0000-%012d"
	ids, err := findUUIDs(str)
	if err != nil {
		return "", errs.Wrapf(err, "failed to find UUIDs in string %s", str)
	}
	newStr := str
	for idx, id := range ids {
		newStr = strings.Replace(newStr, id.String(), fmt.Sprintf(replacementPattern, idx+1), -1)
	}
	return newStr, nil
}

const testInputStr = `
{
	"data": {
		"attributes": {
		"createdAt": "0001-01-01T00:00:00Z",
		"last_used_workspace": "my-last-used-workspace",
		"type": "git",
		"url": "https://github.com/fabric8-services/fabric8-wit.git"
		},
		"id": "d7a282f6-1c10-459e-bb44-55a1a6d48bdd",
		"links": {
		"edit": "http:///api/codebases/d7a282f6-1c10-459e-bb44-55a1a6d48bdd/edit",
		"related": "http:///api/codebases/d7a282f6-1c10-459e-bb44-55a1a6d48bdd",
		"self": "http:///api/codebases/d7a282f6-1c10-459e-bb44-55a1a6d48bdd"
		},
		"relationships": {
		"space": {
			"data": {
			"id": "a8bee527-12d2-4aff-9823-3511c1c8e6b9",
			"type": "spaces"
			},
			"links": {
			"related": "http:///api/spaces/a8bee527-12d2-4aff-9823-3511c1c8e6b9",
			"self": "http:///api/spaces/a8bee527-12d2-4aff-9823-3511c1c8e6b9"
			}
		}
		},
		"type": "codebases"
	}
}`

const testOutputStr = `
{
	"data": {
		"attributes": {
		"createdAt": "0001-01-01T00:00:00Z",
		"last_used_workspace": "my-last-used-workspace",
		"type": "git",
		"url": "https://github.com/fabric8-services/fabric8-wit.git"
		},
		"id": "00000000-0000-0000-0000-000000000001",
		"links": {
		"edit": "http:///api/codebases/00000000-0000-0000-0000-000000000001/edit",
		"related": "http:///api/codebases/00000000-0000-0000-0000-000000000001",
		"self": "http:///api/codebases/00000000-0000-0000-0000-000000000001"
		},
		"relationships": {
		"space": {
			"data": {
			"id": "00000000-0000-0000-0000-000000000002",
			"type": "spaces"
			},
			"links": {
			"related": "http:///api/spaces/00000000-0000-0000-0000-000000000002",
			"self": "http:///api/spaces/00000000-0000-0000-0000-000000000002"
			}
		}
		},
		"type": "codebases"
	}
}`

func TestFindUUIDs(t *testing.T) {
	t.Parallel()
	t.Run("find UUIDs", func(t *testing.T) {
		t.Parallel()
		ids, err := findUUIDs(testInputStr)
		require.NoError(t, err)
		require.Equal(t, []uuid.UUID{
			uuid.FromStringOrNil("d7a282f6-1c10-459e-bb44-55a1a6d48bdd"),
			uuid.FromStringOrNil("a8bee527-12d2-4aff-9823-3511c1c8e6b9"),
		}, ids)
	})
}

func TestReplaceUUIDs(t *testing.T) {
	t.Parallel()
	t.Run("replace UUIDs", func(t *testing.T) {
		t.Parallel()
		newStr, err := replaceUUIDs(testInputStr)
		require.NoError(t, err)
		require.Equal(t, testOutputStr, newStr)
	})
}

func TestCompareWithGolden(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	type Foo struct {
		ID  uuid.UUID
		Bar string
	}
	dummy := Foo{Bar: "hello world", ID: uuid.NewV4()}

	uuidAgnosticVals := []bool{false, true}
	for _, uuidAgnostic := range uuidAgnosticVals {
		t.Run("file not found", func(t *testing.T) {
			// given
			f := "not_existing_file.golden.json"
			// when
			err := testableCompareWithGolden(false, f, dummy, uuidAgnostic)
			// then
			require.Error(t, err)
			_, isPathError := errs.Cause(err).(*os.PathError)
			require.True(t, isPathError)
		})
		t.Run("unable to update golden file due to not existing folder", func(t *testing.T) {
			// given
			f := "not/existing/folder/file.golden.json"
			// when
			err := testableCompareWithGolden(true, f, dummy, uuidAgnostic)
			// then
			require.Error(t, err)
			_, isPathError := errs.Cause(err).(*os.PathError)
			require.True(t, isPathError)
		})
		t.Run("mismatch between expected and actual output", func(t *testing.T) {
			// given
			f := "test-files/codebase/show/ok_without_auth.golden.json"
			// when
			err := testableCompareWithGolden(false, f, dummy, uuidAgnostic)
			// then
			require.Error(t, err)
			_, isPathError := errs.Cause(err).(*os.PathError)
			require.False(t, isPathError)
		})
	}

	t.Run("comparing with existing file", func(t *testing.T) {
		// given
		f := "test-files/dummy.golden.json"
		bs, err := json.MarshalIndent(dummy, "", "  ")
		require.NoError(t, err)
		err = ioutil.WriteFile(f, bs, os.ModePerm)
		require.NoError(t, err)
		defer func() {
			err := os.Remove(f)
			require.NoError(t, err)
		}()

		t.Run("comparing with the same object", func(t *testing.T) {
			t.Run("not UUID agnostic", func(t *testing.T) {
				// when
				err = testableCompareWithGolden(false, f, dummy, false)
				// then
				require.NoError(t, err)
			})
			t.Run("UUID agnostic", func(t *testing.T) {
				// when
				err = testableCompareWithGolden(false, f, dummy, true)
				// then
				require.NoError(t, err)
			})
		})
		t.Run("comparing with the same object but modified its UUID", func(t *testing.T) {
			dummy.ID = uuid.NewV4()
			t.Run("not UUID agnostic", func(t *testing.T) {
				// when
				err = testableCompareWithGolden(false, f, dummy, false)
				// then
				require.Error(t, err)
			})
			t.Run("UUID agnostic", func(t *testing.T) {
				// when
				err = testableCompareWithGolden(false, f, dummy, true)
				// then
				require.NoError(t, err)
			})
		})
	})
}
