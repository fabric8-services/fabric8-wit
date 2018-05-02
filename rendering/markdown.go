package rendering

import (
	"fmt"

	"bytes"
	"strings"

	"github.com/russross/blackfriday"
	"github.com/sourcegraph/syntaxhighlight"
)

const (
	commonHTMLFlags = 0 |
		blackfriday.HTML_USE_XHTML |
		blackfriday.HTML_USE_SMARTYPANTS |
		blackfriday.HTML_SMARTYPANTS_FRACTIONS |
		blackfriday.HTML_SMARTYPANTS_DASHES |
		blackfriday.HTML_SMARTYPANTS_LATEX_DASHES |
		blackfriday.HTML_HREF_TARGET_BLANK

	commonExtensions = 0 |
		blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
		blackfriday.EXTENSION_TABLES |
		blackfriday.EXTENSION_FENCED_CODE |
		blackfriday.EXTENSION_AUTOLINK |
		blackfriday.EXTENSION_STRIKETHROUGH |
		blackfriday.EXTENSION_SPACE_HEADERS |
		blackfriday.EXTENSION_HEADER_IDS |
		blackfriday.EXTENSION_BACKSLASH_LINE_BREAK |
		blackfriday.EXTENSION_DEFINITION_LISTS
)

// MarkdownCommonHighlighter uses the blackfriday.MarkdownCommon setup but also includes
// code-prettify formatting of BlockCode segments
func MarkdownCommonHighlighter(input []byte) []byte {
	renderer := highlightHTMLRenderer{blackfriday.HtmlRenderer(commonHTMLFlags, "", ""), 0}
	return blackfriday.MarkdownOptions(input, &renderer, blackfriday.Options{
		Extensions: commonExtensions})
}

type highlightHTMLRenderer struct {
	blackfriday.Renderer
	checkboxIndex int8
}

// ListItem overrides the default ListItem render and adds support for GH-style
// checkboxes on service side. For the contents of the list item beyond the
// checkbox prefix, the default Html.ListItem is called.
// This adds a data-checkbox-index attribute that contains the ordinal of the
// checkbox in the rendered Markdown. This can be used by the ui to find the
// reference to the original Markdown code from a rendered HTML.
func (h *highlightHTMLRenderer) ListItem(out *bytes.Buffer, text []byte, flags int) {
	switch {
	case bytes.HasPrefix(text, []byte("[] ")):
		text = append([]byte(fmt.Sprintf(`<input class="markdown-checkbox" type="checkbox" data-checkbox-index="%d"></input>`, h.checkboxIndex)), text[3:]...)
		h.checkboxIndex++
	case bytes.HasPrefix(text, []byte("[ ] ")):
		text = append([]byte(fmt.Sprintf(`<input class="markdown-checkbox" type="checkbox" data-checkbox-index="%d"></input>`, h.checkboxIndex)), text[4:]...)
		h.checkboxIndex++
	case bytes.HasPrefix(text, []byte("[x] ")) || bytes.HasPrefix(text, []byte("[X] ")):
		text = append([]byte(fmt.Sprintf(`<input class="markdown-checkbox" type="checkbox" checked="" data-checkbox-index="%d"></input>`, h.checkboxIndex)), text[4:]...)
		h.checkboxIndex++
	}
	h.Renderer.ListItem(out, text, flags)
}

// BlackCode overrides the standard Html Renderer to add support for prettify of source code within block
// If highlighter fail, normal Html.BlockCode is called
func (h highlightHTMLRenderer) BlockCode(out *bytes.Buffer, text []byte, lang string) {
	highlighted, err := syntaxhighlight.AsHTML(text)
	if err != nil {
		h.Renderer.BlockCode(out, text, lang)
	} else {

		if out.Len() > 0 {
			out.WriteByte('\n')
		}

		// parse out the language names/classes
		count := 0
		for _, elt := range strings.Fields(lang) {
			if elt[0] == '.' {
				elt = elt[1:]
			}
			if len(elt) == 0 {
				continue
			}
			if count == 0 {
				out.WriteString("<pre><code class=\"prettyprint language-")
			} else {
				out.WriteByte(' ')
			}
			out.Write([]byte(elt)) // attrEscape(out, []byte(elt))
			count++
		}

		if count == 0 {
			out.WriteString("<pre><code class=\"prettyprint\">")
		} else {
			out.WriteString("\">")
		}

		out.Write(highlighted)
		out.WriteString("</code></pre>\n")
	}
}
