package rendering

import (
	"regexp"

	"github.com/almighty/almighty-core/workitem"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

// IsMarkupSupported indicates if the given markup is supported
func IsMarkupSupported(markup string) bool {
	if markup == "" || // assume same as default
		markup == workitem.SystemMarkupDefault ||
		markup == workitem.SystemMarkupMarkdown {
		return true
	}
	return false
}

// RenderMarkupToHTML converts the given `content` in HTML using the markup tool corresponding to the given `markup` argument
// or return nil if no tool for the given `markup` is available, or returns an `error` if the command was not found or failed.
func RenderMarkupToHTML(content, markup string) (*string, error) {
	switch markup {
	case workitem.SystemMarkupPlainText:
		return &content, nil
	case workitem.SystemMarkupMarkdown:
		unsafe := blackfriday.MarkdownCommon([]byte(content))
		p := bluemonday.UGCPolicy()
		p.AllowAttrs("class").Matching(regexp.MustCompile("^language-[a-zA-Z0-9]+$")).OnElements("code")
		html := string(p.SanitizeBytes(unsafe))
		return &html, nil
	default:
		return nil, nil

	}

}
