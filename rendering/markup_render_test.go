package rendering_test

import (
	"strings"
	"testing"

	"github.com/almighty/almighty-core/rendering"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderMarkdownContent(t *testing.T) {
	content := "Hello, `World`!"
	result := rendering.RenderMarkupToHTML(content, rendering.SystemMarkupMarkdown)
	t.Log(result)
	require.NotNil(t, result)
	assert.Equal(t, "<p>Hello, <code>World</code>!</p>\n", result)
}

func TestRenderMarkdownContentWithFence(t *testing.T) {
	content := "``` go\nfunc getTrue() bool {return true}\n```"
	result := rendering.RenderMarkupToHTML(content, rendering.SystemMarkupMarkdown)
	t.Log(result)
	require.NotNil(t, result)
	assert.True(t, strings.Contains(result, "<code class=\"language-go\">"))
}

func TestIsMarkupSupported(t *testing.T) {
	assert.True(t, rendering.IsMarkupSupported(""))
	assert.True(t, rendering.IsMarkupSupported(rendering.SystemMarkupDefault))
	assert.True(t, rendering.IsMarkupSupported(rendering.SystemMarkupPlainText))
	assert.True(t, rendering.IsMarkupSupported(rendering.SystemMarkupMarkdown))
	assert.False(t, rendering.IsMarkupSupported("foo"))
}
