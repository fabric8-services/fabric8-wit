package workitem

import (
	"fmt"
	"math"
	"sort"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/rendering"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// ValidInvalid given a bunch of tests with expected error results for each work
// item type field kind, a work item type for each kind...
type ValidInvalid struct {
	// Valid should contain more than one valid examples so a test function can
	// properly handle work item creation and updating
	Valid   []interface{}
	Invalid []interface{}
	// When the actual value is a zero (0), it will be interpreted as a float64
	// rather than an int. To compensate for that ambiguity, a kind can opt-in
	// to provide a construction function that returns the correct value.
	Compensate func(interface{}) interface{}
}

// FieldTypeTestDataMap defines a map with additional functionality on it.
type FieldTypeTestDataMap map[Kind]ValidInvalid

// GetKinds returns the keys of the test data map in sorted order.
func (s FieldTypeTestDataMap) GetKinds() []Kind {
	strArr := make([]string, len(s))
	kinds := make([]Kind, len(s))
	i := 0
	for k := range s {
		strArr[i] = k.String()
		i++
	}
	sort.Strings(strArr)
	for i := 0; i < len(strArr); i++ {
		kinds[i] = Kind(strArr[i])
	}
	return kinds
}

// GetFieldTypeTestData returns a list of legal and illegal values to be used
// with a given field type (here: the map key).
func GetFieldTypeTestData(t *testing.T) FieldTypeTestDataMap {
	res := FieldTypeTestDataMap{
		KindString: {
			Valid: []interface{}{
				"foo",
				"bar",
			},
			Invalid: []interface{}{
				"", // NOTE: an empty string is not allowed in a required field.
				nil,
				0,
				true,
				0.1,
			},
		},
		KindUser: {
			Valid: []interface{}{
				"jane doe", // TODO(kwk): do we really allow usernames with spaces?
				"john doe",
				"", // TODO(kwk): do we really allow empty usernames?
			},
			Invalid: []interface{}{
				nil,
				0,
				true,
				0.1,
			},
		},
		KindIteration: {
			Valid: []interface{}{
				"63b792d8-eede-40d9-b311-b43f84285e98",
				"058c9b12-39a9-4495-b24b-b053514a7edb",
				"", // TODO(kwk): do we really allow empty iteration names?
			},
			Invalid: []interface{}{
				nil,
				0,
				true,
				0.1,
			},
		},
		KindArea: {
			Valid: []interface{}{
				"dde919b1-3c61-4d3a-8cda-941c62813749",
				"8aab8647-35e0-4f18-9468-f5c2fbc02b3d",
				//"", // TODO(kwk): do we really allow empty area names?
			},
			Invalid: []interface{}{
				nil,
				0,
				true,
				0.1,
			},
		},
		KindRemoteTracker: {
			Valid: []interface{}{
				"00000000-0000-0000-0000-000000000000",
				"55d5d809-56f0-4d31-9607-04755b793c4a",
				uuid.FromStringOrNil("8f48eb9a-f9cb-428f-a803-7845a96f1d07"),
				uuid.Nil,
			},
			Invalid: []interface{}{
				nil,
				0,
				true,
				0.1,
				"",
			},
		},
		KindLabel: {
			Valid: []interface{}{
				"8eea6cf3-ddf2-4b93-ade2-13c6b75de1df",
				"6efaad81-b35a-44bc-898a-6c66336c7cff",
				"", // TODO(kwk): do we really allow empty label names?
			},
			Invalid: []interface{}{
				nil,
				0,
				true,
				0.1,
			},
		},
		KindBoardColumn: {
			Valid: []interface{}{
				"d05b66fb-9162-4f7b-ac0f-19d9c41324f4",
				"400956cf-741c-4b4d-a89b-a44f4dead04e",
			},
			Invalid: []interface{}{
				nil,
				0,
				true,
				0.1,
			},
		},
		KindURL: {
			Valid: []interface{}{
				"127.0.0.1",
				"http://www.openshift.io",
				"openshift.io",
				"ftp://url.with.port.and.different.protocol.port.and.parameters.com:8080/fooo?arg=bar&key=value",
			},
			Invalid: []interface{}{
				0,
				"", // NOTE: An empty URL is not allowed when the field is required (see simple_type.go:53)
				"http://url with whitespace.com",
				"http://www.example.com/foo bar",
				"localhost", // TODO(kwk): shall we disallow localhost?
				"foo",
			},
		},
		KindInteger: {
			// Compensate for wrong interpretation of 0
			Compensate: func(in interface{}) interface{} {
				v := in.(float64) // NOTE: float64 is correct here because a 0 will first and foremost be treated as float64
				if v != math.Trunc(v) {
					panic(fmt.Sprintf("value is not a whole number %v", v))
				}
				return int(v)
			},
			Valid: []interface{}{
				int(0),
				333,
				-100,
			},
			Invalid: []interface{}{
				1.23,
				nil,
				"",
				"foo",
				0.1,
				true,
				false,
			},
		},
		KindFloat: {
			Valid: []interface{}{
				0.1,
				-1111.1,
				+555.2,
			},
			Invalid: []interface{}{
				1,
				0,
				"string",
			},
		},
		KindBoolean: {
			Valid: []interface{}{
				true,
				false,
			},
			Invalid: []interface{}{
				nil,
				0,
				1,
				"",
				"yes",
				"no",
				"0",
				"1",
				"true",
				"false",
			},
		},
		KindInstant: {
			// Compensate for wrong interpretation of location value and default to UTC
			Compensate: func(in interface{}) interface{} {
				v := in.(time.Time)
				return v.UTC()
			},
			Valid: []interface{}{
				// NOTE: If we don't use UTC(), the unmarshalled JSON will
				// have a different time zone (read up on JSON an time
				// location if you don't believe me).
				func() interface{} {
					v, err := time.Parse("02 Jan 06 15:04 -0700", "02 Jan 06 15:04 -0700")
					require.NoError(t, err)
					return v.UTC()
				}(),
				func() interface{} {
					v, err := time.Parse("02 Jan 06 15:04 -0700", "03 Jan 06 15:04 -0700")
					require.NoError(t, err)
					return v.UTC()
				}(),
				// time.Now().UTC(), // TODO(kwk): Somehow this fails due to different nsec
			},
			Invalid: []interface{}{
				time.Now().String(),
				time.Now().UTC().String(),
				"2017-09-27 13:40:48.099780356 +0200 CEST", // NOTE: looks like a time.Time but is a string
				"",
				0,
				333,
				100,
				1e2,
				nil,
				"foo",
				0.1,
				true,
				false,
			},
		},
		KindMarkup: {
			Valid: []interface{}{
				rendering.MarkupContent{Content: "plain text", Markup: rendering.SystemMarkupPlainText},
				rendering.MarkupContent{Content: "default", Markup: rendering.SystemMarkupDefault},
				rendering.MarkupContent{Content: "# markdown", Markup: rendering.SystemMarkupMarkdown},
			},
			Invalid: []interface{}{
				0,
				rendering.MarkupContent{Content: "jira", Markup: rendering.SystemMarkupJiraWiki}, // TODO(kwk): not supported yet
				rendering.MarkupContent{Content: "", Markup: ""},                                 // NOTE: We allow allow empty strings
				rendering.MarkupContent{Content: "foo", Markup: "unknown markup type"},
				"",
				"foo",
			},
		},
		KindCodebase: {
			Valid: []interface{}{
				codebase.Content{
					Repository: "git://github.com/ember-cli/ember-cli.git#ff786f9f",
					Branch:     "foo",
					FileName:   "bar.js",
					LineNumber: 10,
					CodebaseID: "dunno",
				},
				codebase.Content{
					Repository: "git://github.com/pkg/error.git",
					Branch:     "master",
					FileName:   "main.go",
					LineNumber: 15,
					CodebaseID: "dunno",
				},
			},
			Invalid: []interface{}{
				// empty repository (see codebase.Content.IsValid())
				codebase.Content{
					Repository: "",
					Branch:     "foo",
					FileName:   "bar.js",
					LineNumber: 10,
					CodebaseID: "dunno",
				},
				// invalid repository URL (see codebase.Content.IsValid())
				codebase.Content{
					Repository: "/path/to/repo.git/",
					Branch:     "foo",
					FileName:   "bar.js",
					LineNumber: 10,
					CodebaseID: "dunno",
				},
				"",
				0,
				333,
				100,
				1e2,
				nil,
				"foo",
				0.1,
				true,
				false,
			},
		},
		//KindEnum:  {}, // TODO(kwk): Add test for KindEnum
		//KindList:  {}, // TODO(kwk): Add test for KindList
	}

	for k, iv := range res {
		if len(iv.Valid) < 2 {
			t.Fatalf("at least two valid examples required for kind %s but only %d given", k, len(iv.Valid))
		}
		if len(iv.Invalid) < 1 {
			t.Fatalf("at least one invalid example is required for kind %s but only %d given", k, len(iv.Invalid))
		}
	}
	return res
}
