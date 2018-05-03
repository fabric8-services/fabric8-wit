package rendering_test

import (
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderMarkupToHTML(t *testing.T) {
	t.Parallel()
	t.Run("markdown content", func(t *testing.T) {
		t.Parallel()
		content := "Hello, `World`!"
		result := rendering.RenderMarkupToHTML(content, rendering.SystemMarkupMarkdown)
		t.Log(result)
		assert.Equal(t, "<p>Hello, <code>World</code>!</p>\n", result)
	})
	t.Run("JIRA wiki content", func(t *testing.T) {
		t.Parallel()
		content := `foo`
		result := rendering.RenderMarkupToHTML(content, rendering.SystemMarkupJiraWiki)
		require.Empty(t, result)
	})
	t.Run("markdown with code fence", func(t *testing.T) {
		t.Parallel()
		content := "``` go\nfunc getTrue() bool {return true}\n```"
		result := rendering.RenderMarkupToHTML(content, rendering.SystemMarkupMarkdown)
		t.Log(result)
		assert.True(t, strings.Contains(result, "<code class=\"prettyprint language-go\">"))
	})
	t.Run("markdown with code fence highligher", func(t *testing.T) {
		t.Parallel()
		content := "``` go\nfunc getTrue() bool {return true}\n```"
		result := rendering.RenderMarkupToHTML(content, rendering.SystemMarkupMarkdown)
		t.Log(result)
		assert.True(t, strings.Contains(result, "<code class=\"prettyprint language-go\">"))
		assert.True(t, strings.Contains(result, "<span class=\"kwd\">func</span>"))
	})
}

func TestCheckValid(t *testing.T) {
	t.Parallel()
	assert.NoError(t, rendering.SystemMarkupDefault.CheckValid())
	assert.NoError(t, rendering.SystemMarkupPlainText.CheckValid())
	assert.Error(t, rendering.SystemMarkupJiraWiki.CheckValid())
	assert.Error(t, rendering.Markup("").CheckValid())
	assert.Error(t, rendering.Markup("foo").CheckValid())
}
