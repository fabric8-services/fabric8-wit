package rendering_test

import (
	"strings"
	"testing"

	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/workitem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderMarkdownContent(t *testing.T) {
	content := "Hello, `World`!"
	result, err := rendering.RenderMarkupToHTML(content, workitem.SystemMarkupMarkdown)
	if err != nil {
		t.Error(err)
	}
	t.Log(*result)
	require.NotNil(t, result)
	assert.Equal(t, *result, "<p>Hello, <code>World</code>!</p>\n")
}

func TestRenderMarkdownContentWithFence(t *testing.T) {
	content := "``` go\nfunc getTrue() bool {return true}\n```"
	result, err := rendering.RenderMarkupToHTML(content, workitem.SystemMarkupMarkdown)
	if err != nil {
		t.Error(err)
	}
	t.Log(*result)
	require.NotNil(t, result)
	assert.True(t, strings.Contains(*result, "<code class=\"language-go\">"))
}

func TestIsMarkupSupported(t *testing.T) {
	assert.True(t, rendering.IsMarkupSupported(""))
	assert.True(t, rendering.IsMarkupSupported(workitem.SystemMarkupDefault))
	assert.True(t, rendering.IsMarkupSupported(workitem.SystemMarkupPlainText))
	assert.True(t, rendering.IsMarkupSupported(workitem.SystemMarkupMarkdown))
	assert.False(t, rendering.IsMarkupSupported("foo"))
}
