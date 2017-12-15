package workitem

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/stretchr/testify/require"
)

// given a bunch of tests with expected error results for each work item
// type field kind, a work item type for each kind...
type ValidInvalid struct {
	Valid   []interface{}
	Invalid []interface{}
	// When the actual value is a zero (0), it will be interpreted as a
	// float64 rather than an int. To compensate for that ambiguity, a
	// kind can opt-in to provide an construction function that returns
	// the correct value.
	Compensate func(interface{}) interface{}
}

// GetFieldTypeTestData returns a list of legal and illegal values to be used
// with a given field type (here: the map key).
func GetFieldTypeTestData(t *testing.T) map[Kind]ValidInvalid {
	// helper function to convert a string into a duration and handling the
	// error
	validDuration := func(s string) time.Duration {
		d, err := time.ParseDuration(s)
		if err != nil {
			require.NoError(t, err, "we expected the duration to be valid: %s", s)
		}
		return d
	}

	return map[Kind]ValidInvalid{
		KindString: {
			Valid: []interface{}{
				"foo",
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
				"",         // TODO(kwk): do we really allow empty usernames?
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
				"some iteration name",
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
				"some are name",
				"", // TODO(kwk): do we really allow empty area names?
			},
			Invalid: []interface{}{
				nil,
				0,
				true,
				0.1,
			},
		},
		KindLabel: {
			Valid: []interface{}{
				"some label name",
				"", // TODO(kwk): do we really allow empty label names?
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
				0,
				333,
				-100,
			},
			Invalid: []interface{}{
				1e2,
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
				-1111.0,
				+555.0,
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
		KindDuration: {
			// Compensate for wrong interpretation of 0
			Compensate: func(in interface{}) interface{} {
				i := in.(float64)
				return time.Duration(int64(i))
			},
			Valid: []interface{}{
				validDuration("0"),
				validDuration("300ms"),
				validDuration("-1.5h"),
				// 0, // TODO(kwk): should work because an untyped integer constant can be converted to time.Duration's underlying type: int64
			},
			Invalid: []interface{}{
				// 0, // TODO(kwk): 0 doesn't fit in legal nor illegal
				nil,
				"1e2",
				"4000",
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
}
