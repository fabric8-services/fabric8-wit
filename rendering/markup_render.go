package rendering

import (
	"regexp"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

// IsMarkupSupported indicates if the given markup is supported
func IsMarkupSupported(markup string) bool {
	if markup == SystemMarkupDefault || markup == SystemMarkupMarkdown {
		return true
	}
	return false
}

// RenderMarkupToHTML converts the given `content` in HTML using the markup tool corresponding to the given `markup` argument
// or return nil if no tool for the given `markup` is available, or returns an `error` if the command was not found or failed.
func RenderMarkupToHTML(content, markup string) string {
	switch markup {
	case SystemMarkupPlainText:
		return content
	case SystemMarkupMarkdown:
		unsafe := blackfriday.MarkdownCommon([]byte(content))
		p := bluemonday.UGCPolicy()
		p.AllowAttrs("class").Matching(regexp.MustCompile("^language-[a-zA-Z0-9]+$")).OnElements("code")
		html := string(p.SanitizeBytes(unsafe))
		return html
	default:
		return ""

	}

}
