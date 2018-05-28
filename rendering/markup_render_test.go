package rendering_test

import (
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/rendering"
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
	assert.True(t, strings.Contains(result, "<code class=\"prettyprint language-go\">"))
}

func TestRenderMarkdownContentWithFenceHighlighter(t *testing.T) {
	content := "``` go\nfunc getTrue() bool {return true}\n```"
	result := rendering.RenderMarkupToHTML(content, rendering.SystemMarkupMarkdown)
	t.Log(result)
	require.NotNil(t, result)
	assert.True(t, strings.Contains(result, "<code class=\"prettyprint language-go\">"))
	assert.True(t, strings.Contains(result, "<span class=\"kwd\">func</span>"))
}

func TestRenderMarkdownContentWithCheckboxItems(t *testing.T) {
	t.Run("star lists", func(t *testing.T) {
		content := "* [ ] Some Item 0\n* [ ] Some Item 1\n* [X] Some Item 2\n* [x] Some Item 3"
		result := rendering.RenderMarkupToHTML(content, rendering.SystemMarkupMarkdown)
		t.Log(result)
		require.NotNil(t, result)
		assert.True(t, strings.Contains(result, "<input class=\"markdown-checkbox\" type=\"checkbox\" data-checkbox-index=\"0\"></input>Some Item 0"))
		assert.True(t, strings.Contains(result, "<input class=\"markdown-checkbox\" type=\"checkbox\" data-checkbox-index=\"1\"></input>Some Item 1"))
		assert.True(t, strings.Contains(result, "<input class=\"markdown-checkbox\" type=\"checkbox\" checked=\"\" data-checkbox-index=\"2\"></input>Some Item 2"))
		assert.True(t, strings.Contains(result, "<input class=\"markdown-checkbox\" type=\"checkbox\" checked=\"\" data-checkbox-index=\"3\"></input>Some Item 3"))
	})
	t.Run("dash lists", func(t *testing.T) {
		content := "- [ ] Some Item 0\n- [ ] Some Item 1\n- [X] Some Item 2\n- [x] Some Item 3"
		result := rendering.RenderMarkupToHTML(content, rendering.SystemMarkupMarkdown)
		t.Log(result)
		require.NotNil(t, result)
		assert.True(t, strings.Contains(result, "<input class=\"markdown-checkbox\" type=\"checkbox\" data-checkbox-index=\"0\"></input>Some Item 0"))
		assert.True(t, strings.Contains(result, "<input class=\"markdown-checkbox\" type=\"checkbox\" data-checkbox-index=\"1\"></input>Some Item 1"))
		assert.True(t, strings.Contains(result, "<input class=\"markdown-checkbox\" type=\"checkbox\" checked=\"\" data-checkbox-index=\"2\"></input>Some Item 2"))
		assert.True(t, strings.Contains(result, "<input class=\"markdown-checkbox\" type=\"checkbox\" checked=\"\" data-checkbox-index=\"3\"></input>Some Item 3"))
	})
	t.Run("antipatterns", func(t *testing.T) {
		content := "- [ ]Some Item 0\n- [] Some Item 1\n- [X]Some Item 2\n- [x]Some Item 3"
		result := rendering.RenderMarkupToHTML(content, rendering.SystemMarkupMarkdown)
		t.Log(result)
		require.NotNil(t, result)
		assert.False(t, strings.Contains(result, "<input class=\"markdown-checkbox\" type=\"checkbox\" data-checkbox-index=\"0\"></input>Some Item 0"))
		assert.False(t, strings.Contains(result, "<input class=\"markdown-checkbox\" type=\"checkbox\" data-checkbox-index=\"1\"></input>Some Item 1"))
		assert.False(t, strings.Contains(result, "<input class=\"markdown-checkbox\" type=\"checkbox\" checked=\"\" data-checkbox-index=\"2\"></input>Some Item 2"))
		assert.False(t, strings.Contains(result, "<input class=\"markdown-checkbox\" type=\"checkbox\" checked=\"\" data-checkbox-index=\"3\"></input>Some Item 3"))
	})
}

func TestIsMarkupSupported(t *testing.T) {
	assert.True(t, rendering.IsMarkupSupported(rendering.SystemMarkupDefault))
	assert.True(t, rendering.IsMarkupSupported(rendering.SystemMarkupPlainText))
	assert.True(t, rendering.IsMarkupSupported(rendering.SystemMarkupMarkdown))
	assert.False(t, rendering.IsMarkupSupported(""))
	assert.False(t, rendering.IsMarkupSupported("foo"))
}
