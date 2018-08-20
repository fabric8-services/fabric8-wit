package workitem

import (
	"math"
	"sort"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/rendering"
	uuid "github.com/satori/go.uuid"
)

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

// A FieldTypeTestDataMap can hold valid and invalid tests data stored for each
// work item field kind
type FieldTypeTestDataMap map[Kind]ValidInvalid

// ValidInvalid given a bunch of tests with expected error results for each work
// item type field kind, a work item type for each kind...
type ValidInvalid struct {
	// Valid should contain more than one valid examples so a test function can
	// properly handle work item creation and updating
	Valid               []InputOutput
	Invalid             []interface{}
	InvalidWhenRequired []interface{}
}

// InputOutput defines how an input is represented in model an in storage space
type InputOutput struct {
	// Input is what a client or Go code can input when creating a work item.
	Input interface{}
	// Output is what the Go code in model space will be for the given input.
	Output interface{}
	// Storage is what is stored in the DB for the given input.
	Storage interface{}
}

func inAsOut(in interface{}) InputOutput {
	return InputOutput{Input: in, Output: in, Storage: in}
}

// GetFieldTypeTestData returns a list of legal and illegal values to be used
// with a given field type (here: the map key).
func GetFieldTypeTestData(t *testing.T) FieldTypeTestDataMap {
	// create a time value
	timeNow := time.Now()
	_ = timeNow

	// structure that can be used by all kinds that store UUIDs
	uuidVal := uuid.NewV4()
	validInvalidForUUID := ValidInvalid{
		Valid: []InputOutput{
			inAsOut(uuid.NewV4()),
			{Input: uuidVal.String(), Output: uuidVal, Storage: uuidVal},
			{Input: uuid.Nil, Output: uuid.Nil, Storage: uuid.Nil},
		},
		Invalid: []interface{}{
			"john doe", // users have to be IDs
			0,
			true,
			0.1,
			"",
		},
		InvalidWhenRequired: []interface{}{
			nil,
		},
	}
	_ = validInvalidForUUID

	res := FieldTypeTestDataMap{
		KindString: {
			Valid: []InputOutput{
				inAsOut("foo"),
				inAsOut("bar"),
			},
			Invalid: []interface{}{
				0,
				true,
				0.1,
			},
			InvalidWhenRequired: []interface{}{
				"",
				nil,
			},
		},
		KindUser:        validInvalidForUUID,
		KindIteration:   validInvalidForUUID,
		KindArea:        validInvalidForUUID,
		KindLabel:       validInvalidForUUID,
		KindBoardColumn: validInvalidForUUID,
		KindURL: {
			Valid: []InputOutput{
				inAsOut("127.0.0.1"),
				inAsOut("http://www.openshift.io"),
				inAsOut("openshift.io"),
				inAsOut("ftp://url.with.port.and.different.protocol.port.and.parameters.com:8080/fooo?arg=bar&key=value"),
			},
			Invalid: []interface{}{
				0,
				"http://url with whitespace.com",
				"http://www.example.com/foo bar",
				"localhost", // TODO(kwk): shall we disallow localhost?
				"foo",
				"",
			},
			InvalidWhenRequired: []interface{}{
				nil,
			},
		},
		KindInteger: {
			Valid: []InputOutput{
				{0, int32(0), int32(0)},
				{333, int32(333), int32(333)},
				{-100, int32(-100), int32(-100)},
				{222.0, int32(222), int32(222)},
				{"0", int32(0), int32(0)},
				{"123", int32(123), int32(123)},
				{int(123), int32(123), int32(123)},
				{int8(8), int32(8), int32(8)},
				{int16(16), int32(16), int32(16)},
				{int32(32), int32(32), int32(32)},
				{int64(64), int32(64), int32(64)},
				{int(-123), int32(-123), int32(-123)},
				{int8(-8), int32(-8), int32(-8)},
				{int16(-16), int32(-16), int32(-16)},
				{int32(-32), int32(-32), int32(-32)},
				{int64(-64), int32(-64), int32(-64)},
				{uint(123), int32(123), int32(123)},
				{uint8(8), int32(8), int32(8)},
				{uint16(16), int32(16), int32(16)},
				{uint32(32), int32(32), int32(32)},
				{uint64(64), int32(64), int32(64)},
				{float32(32), int32(32), int32(32)},
				{float64(64), int32(64), int32(64)},
				// min
				{int64(math.MinInt32), int32(math.MinInt32), int32(math.MinInt32)},
				{int32(math.MinInt32), int32(math.MinInt32), int32(math.MinInt32)},
				{float64(math.MinInt32), int32(math.MinInt32), int32(math.MinInt32)},
				// max
				{uint32(math.MaxInt32), int32(math.MaxInt32), int32(math.MaxInt32)},
				{uint64(math.MaxInt32), int32(math.MaxInt32), int32(math.MaxInt32)},
				{int64(math.MaxInt32), int32(math.MaxInt32), int32(math.MaxInt32)},
				{int32(math.MaxInt32), int32(math.MaxInt32), int32(math.MaxInt32)},
				{float64(math.MaxInt32), int32(math.MaxInt32), int32(math.MaxInt32)},
			},
			Invalid: []interface{}{
				int64(math.MaxInt32) + 1,
				int64(math.MinInt32) - 1,
				1.2,
				"foo",
				"123.2",
				0.1,
				true,
				false,
			},
			InvalidWhenRequired: []interface{}{
				"",
				nil,
			},
		},
		KindFloat: {
			Valid: []InputOutput{
				{0.1, float64(0.1), float64(0.1)},
				{-1111.1, float64(-1111.1), float64(-1111.1)},
				{+555.2, float64(555.2), float64(555.2)},
				{123, float64(123), float64(123)},
				{0, float64(0), float64(0)},
				{"123", float64(123), float64(123)},
				{"123.2", float64(123.2), float64(123.2)},
				{"0", float64(0), float64(0)},
				{int(123), float64(123), float64(123)},
				{int8(8), float64(8), float64(8)},
				{int16(16), float64(16), float64(16)},
				{int32(32), float64(32), float64(32)},
				{int64(64), float64(64), float64(64)},
				{int(-123), float64(-123), float64(-123)},
				{int8(-8), float64(-8), float64(-8)},
				{int16(-16), float64(-16), float64(-16)},
				{int32(-32), float64(-32), float64(-32)},
				{int64(-64), float64(-64), float64(-64)},
				{uint(123), float64(123), float64(123)},
				{uint8(8), float64(8), float64(8)},
				{uint16(16), float64(16), float64(16)},
				{uint32(32), float64(32), float64(32)},
				{uint64(64), float64(64), float64(64)},
				// min/max
				{float64(math.MaxFloat64), float64(math.MaxFloat64), float64(math.MaxFloat64)},
				{float64(-math.MaxFloat64), float64(-math.MaxFloat64), float64(-math.MaxFloat64)},
				{maxAcurateInt64InFloat64, float64(maxAcurateInt64InFloat64), float64(maxAcurateInt64InFloat64)},
				{minAcurateInt64InFloat64, float64(minAcurateInt64InFloat64), float64(minAcurateInt64InFloat64)},
			},
			Invalid: []interface{}{
				"string",
				true,
				false,
				minAcurateInt64InFloat64 - 1,
				maxAcurateInt64InFloat64 + 1,
				int64(math.MaxInt64),
				int64(math.MinInt64),
			},
			InvalidWhenRequired: []interface{}{
				"",
				nil,
			},
		},
		KindBoolean: {
			Valid: []InputOutput{
				inAsOut(true),
				inAsOut(false),
			},
			Invalid: []interface{}{
				0,
				1,
				"yes",
				"no",
				"0",
				"1",
				"true",
				"false",
			},
			InvalidWhenRequired: []interface{}{
				"",
				nil,
			},
		},
		KindInstant: {
			Valid: []InputOutput{
				{
					timeNow.UTC().String(),
					time.Unix(timeNow.Unix(), 0).UTC(),
					time.Unix(timeNow.Unix(), 0).UTC().Unix(),
				},
				{
					timeNow.UTC(),
					time.Unix(timeNow.Unix(), 0).UTC(),
					time.Unix(timeNow.Unix(), 0).UTC().Unix(),
				},
				{
					timeNow.Unix(),
					time.Unix(timeNow.Unix(), 0).UTC(),
					time.Unix(timeNow.Unix(), 0).UTC().Unix(),
				},
				{
					timeNow.UnixNano(),
					time.Unix(timeNow.Unix(), 0).UTC(),
					time.Unix(timeNow.Unix(), 0).UTC().Unix(),
				},
				{
					float64(timeNow.Unix()),
					time.Unix(timeNow.Unix(), 0).UTC(),
					time.Unix(timeNow.Unix(), 0).UTC().Unix(),
				},
				{
					float64(timeNow.UnixNano()),
					time.Unix(timeNow.Unix(), 0).UTC(),
					time.Unix(timeNow.Unix(), 0).UTC().Unix(),
				},
				{
					"2006-01-02",
					time.Date(2006, 01, 02, 0, 0, 0, 0, time.UTC),
					time.Date(2006, 01, 02, 0, 0, 0, 0, time.UTC).Unix(),
				},
				{
					"Monday, 02-Jan-06 15:04:05 UTC",
					time.Date(2006, 01, 02, 15, 4, 5, 0, time.UTC),
					time.Date(2006, 01, 02, 15, 4, 5, 0, time.UTC).Unix(),
				},
				{
					"2009-11-10 23:00:00 +0000 UTC m=+0.000000001",
					time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC),
					time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC).Unix(),
				},
			},
			Invalid: []interface{}{
				0,
			},
			InvalidWhenRequired: []interface{}{
				"",
				nil,
			},
		},
		KindMarkup: {
			Valid: []InputOutput{
				{
					rendering.MarkupContent{Content: "plain text", Markup: rendering.SystemMarkupPlainText},
					rendering.MarkupContent{Content: "plain text", Markup: rendering.SystemMarkupPlainText},
					rendering.MarkupContent{Content: "plain text", Markup: rendering.SystemMarkupPlainText}.ToMap(),
				},
				{
					rendering.MarkupContent{Content: "default", Markup: rendering.SystemMarkupDefault},
					rendering.MarkupContent{Content: "default", Markup: rendering.SystemMarkupDefault},
					rendering.MarkupContent{Content: "default", Markup: rendering.SystemMarkupDefault}.ToMap(),
				},
				{
					rendering.MarkupContent{Content: "# markdown", Markup: rendering.SystemMarkupMarkdown},
					rendering.MarkupContent{Content: "# markdown", Markup: rendering.SystemMarkupMarkdown},
					rendering.MarkupContent{Content: "# markdown", Markup: rendering.SystemMarkupMarkdown}.ToMap(),
				},
			},
			Invalid: []interface{}{
				0,
				rendering.MarkupContent{Content: "jira", Markup: rendering.SystemMarkupJiraWiki}, // TODO(kwk): JIRA markup not supported yet
				rendering.MarkupContent{Content: "", Markup: ""},                                 // NOTE: We allow allow empty strings
				rendering.MarkupContent{Content: "foo", Markup: "unknown markup type"},
				"foo",
			},
			InvalidWhenRequired: []interface{}{
				"",
				nil,
			},
		},
		KindCodebase: {
			Valid: []InputOutput{
				{
					codebase.Content{
						Repository: "git://github.com/ember-cli/ember-cli.git#ff786f9f",
						Branch:     "foo",
						FileName:   "bar.js",
						LineNumber: 10,
						CodebaseID: "dunno",
					},
					codebase.Content{
						Repository: "git://github.com/ember-cli/ember-cli.git#ff786f9f",
						Branch:     "foo",
						FileName:   "bar.js",
						LineNumber: 10,
						CodebaseID: "dunno",
					},
					codebase.Content{
						Repository: "git://github.com/ember-cli/ember-cli.git#ff786f9f",
						Branch:     "foo",
						FileName:   "bar.js",
						LineNumber: 10,
						CodebaseID: "dunno",
					}.ToMap(),
				},
				{
					codebase.Content{
						Repository: "git://github.com/pkg/error.git",
						Branch:     "master",
						FileName:   "main.go",
						LineNumber: 15,
						CodebaseID: "dunno",
					},
					codebase.Content{
						Repository: "git://github.com/pkg/error.git",
						Branch:     "master",
						FileName:   "main.go",
						LineNumber: 15,
						CodebaseID: "dunno",
					},
					codebase.Content{
						Repository: "git://github.com/pkg/error.git",
						Branch:     "master",
						FileName:   "main.go",
						LineNumber: 15,
						CodebaseID: "dunno",
					}.ToMap(),
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
				0,
				333,
				100,
				1e2,
				"foo",
				0.1,
				true,
				false,
			},
			InvalidWhenRequired: []interface{}{
				"",
				nil,
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
