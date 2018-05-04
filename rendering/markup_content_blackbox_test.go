package rendering_test

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/fabric8-services/fabric8-wit/ptr"
	r "github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMarkupContent_ToMap(t *testing.T) {
	t.Parallel()
	// given
	type testDataT struct {
		Name   string
		Input  r.MarkupContent
		Output map[string]interface{}
	}
	testData := []testDataT{
		{
			Name: "without markup",
			Input: r.MarkupContent{
				Content:  "foo",
				Rendered: "bar",
			},
			Output: map[string]interface{}{
				r.ContentKey:  "foo",
				r.RenderedKey: "bar",
				r.MarkupKey:   r.SystemMarkupDefault,
			},
		},
		{
			Name: "with markup",
			Input: r.MarkupContent{
				Content:  "foo",
				Rendered: "bar",
				Markup:   r.SystemMarkupMarkdown,
			},
			Output: map[string]interface{}{
				r.ContentKey:  "foo",
				r.RenderedKey: "bar",
				r.MarkupKey:   r.SystemMarkupMarkdown,
			},
		},
	}
	for _, td := range testData {
		t.Run(td.Name, func(t *testing.T) {
			t.Parallel()
			// when
			res := td.Input.ToMap()
			// then
			require.Equal(t, td.Output, res)
		})
	}
}

func TestNewMarkupContentFromMap(t *testing.T) {
	t.Parallel()
	// given
	type testDataT struct {
		Name   string
		Input  map[string]interface{}
		Output r.MarkupContent
	}
	testData := []testDataT{
		{
			Name:   "nil",
			Input:  nil,
			Output: r.MarkupContent{Markup: r.SystemMarkupDefault},
		},
		{
			Name:   "empty map",
			Input:  map[string]interface{}{},
			Output: r.MarkupContent{Markup: r.SystemMarkupDefault},
		},
		{
			Name: "wrong map",
			Input: map[string]interface{}{
				"foo": "bar",
			},
			Output: r.MarkupContent{Markup: r.SystemMarkupDefault},
		},
		{
			Name: "complete map",
			Input: map[string]interface{}{
				r.ContentKey:  "foo",
				r.MarkupKey:   r.SystemMarkupMarkdown.String(),
				r.RenderedKey: "bar",
			},
			Output: r.MarkupContent{
				Content:  "foo",
				Markup:   r.SystemMarkupMarkdown,
				Rendered: "bar",
			},
		},
		{
			Name: "no markup key",
			Input: map[string]interface{}{
				r.ContentKey:  "foo",
				r.RenderedKey: "bar",
			},
			Output: r.MarkupContent{
				Content:  "foo",
				Markup:   r.SystemMarkupDefault,
				Rendered: "bar",
			},
		},
		{
			Name: "no markup and no rendered key",
			Input: map[string]interface{}{
				r.ContentKey: "foo",
			},
			Output: r.MarkupContent{
				Content: "foo",
				Markup:  r.SystemMarkupDefault,
			},
		},
	}
	for _, td := range testData {
		t.Run(td.Name, func(t *testing.T) {
			t.Parallel()
			// when
			res := r.NewMarkupContentFromMap(td.Input)
			// then
			require.Equal(t, td.Output, res)
		})
	}
}

func TestNewMarkupContentFromLegacy(t *testing.T) {
	t.Parallel()
	// given
	content := "foo"
	// when
	m := r.NewMarkupContentFromLegacy(content)
	// then
	require.Equal(t, r.MarkupContent{Content: content, Markup: r.SystemMarkupDefault}, m)
}

func TestNewMarkupContent(t *testing.T) {
	t.Parallel()
	// given
	content := "foo"
	markup := r.SystemMarkupMarkdown
	// when
	m := r.NewMarkupContent(content, markup)
	// then
	require.Equal(t, r.MarkupContent{Content: content, Markup: markup}, m)
}

func TestNewMarkupContentFromValue(t *testing.T) {
	t.Parallel()
	type testDataT struct {
		Name   string
		Input  interface{}
		Output *r.MarkupContent
	}
	testData := []testDataT{
		{
			Name:   "nil",
			Input:  nil,
			Output: nil,
		},
		{
			Name:  "markup from legacy",
			Input: "markup from legacy",
			Output: func() *r.MarkupContent {
				s := r.NewMarkupContentFromLegacy("markup from legacy")
				return &s
			}(),
		},
		{
			Name: "markup content",
			Input: r.MarkupContent{
				Content:  "foo",
				Markup:   r.SystemMarkupMarkdown,
				Rendered: "bar",
			},
			Output: &r.MarkupContent{
				Content:  "foo",
				Markup:   r.SystemMarkupMarkdown,
				Rendered: "bar",
			},
		},
		{
			Name: "markup map content",
			Input: map[string]interface{}{
				r.ContentKey:  "foo",
				r.MarkupKey:   r.SystemMarkupMarkdown.String(),
				r.RenderedKey: "bar",
			},
			Output: func() *r.MarkupContent {
				s := r.NewMarkupContentFromMap(map[string]interface{}{
					r.ContentKey:  "foo",
					r.MarkupKey:   r.SystemMarkupMarkdown.String(),
					r.RenderedKey: "bar",
				})
				return &s
			}(),
		},
		{
			Name:   "wrong input type",
			Input:  1234,
			Output: nil,
		},
	}
	for _, td := range testData {
		t.Run(td.Name, func(t *testing.T) {
			t.Parallel()
			// when
			m := r.NewMarkupContentFromValue(td.Input)
			// then
			require.Equal(t, td.Output, m)
		})
	}
}

func TestNilSafeGetMarkup(t *testing.T) {
	t.Parallel()
	// Input/Output test map
	testData := map[*string]r.Markup{
		nil: r.SystemMarkupDefault,
		ptr.String(r.SystemMarkupMarkdown.String()):  r.SystemMarkupMarkdown,
		ptr.String(r.SystemMarkupPlainText.String()): r.SystemMarkupPlainText,
		ptr.String(r.SystemMarkupDefault.String()):   r.SystemMarkupDefault,
		ptr.String(r.SystemMarkupJiraWiki.String()):  r.SystemMarkupDefault, // TODO(kwk): change to JIRA syntax once supported
		ptr.String("foobar"):                         r.SystemMarkupDefault, // NOTE: Stupid values default to default value
		ptr.String(""):                               r.SystemMarkupDefault,
	}
	for i, o := range testData {
		t.Run(spew.Sdump(i), func(t *testing.T) {
			t.Parallel()
			// when
			result := r.NilSafeGetMarkup(i)
			// then
			assert.Equal(t, o, result)
		})
	}
}
