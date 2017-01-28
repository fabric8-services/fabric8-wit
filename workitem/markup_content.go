package workitem

import "github.com/almighty/almighty-core/rendering"

// MarkupContent defines the raw content of a field along with the markup language used to input the content.
type MarkupContent struct {
	Content string `json:"content"`
	Markup  string `json:"markup"`
}

const (
	// the key for the 'content' field when the MarkupContent is converted into/from a Map
	ContentKey = "content"
	// the key for the 'markup' field when the MarkupContent is converted into/from a Map
	MarkupKey = "markup"
)

func (markupContent *MarkupContent) toMap() map[string]interface{} {
	result := make(map[string]interface{})
	result[ContentKey] = markupContent.Content
	if markupContent.Markup == "" {
		result[MarkupKey] = rendering.SystemMarkupDefault
	} else {
		result[MarkupKey] = markupContent.Markup
	}
	return result
}

// NewMarkupContentFromMap creates a MarkupContent from the given Map,
// filling the 'Markup' field with the default value if no entry was found in the input or if the given markup is not supported.
// This avoids filling the DB with invalid markup types.
func NewMarkupContentFromMap(value map[string]interface{}) MarkupContent {
	content := value[ContentKey].(string)
	markup := rendering.SystemMarkupDefault
	if m, ok := value[MarkupKey]; ok {
		markup = m.(string)
		// use default markup if the input is not supported
		if !rendering.IsMarkupSupported(markup) {
			markup = rendering.SystemMarkupDefault
		}
	}
	return MarkupContent{Content: content, Markup: markup}
}

// NewMarkupContentFromLegacy creates a MarkupContent from the given content, using the default markup.
func NewMarkupContentFromLegacy(content string) MarkupContent {
	return MarkupContent{Content: content, Markup: rendering.SystemMarkupDefault}
}

// NewMarkupContent creates a MarkupContent from the given content, using the default markup.
func NewMarkupContent(content, markup string) MarkupContent {
	return MarkupContent{Content: content, Markup: markup}
}

// NewMarkupContentFromValue creates a MarkupContent from the given value,
// by converting a 'string' or casting a 'MarkupContent'. Otherwise, it returns nil.
func NewMarkupContentFromValue(value interface{}) *MarkupContent {
	if value == nil {
		return nil
	}
	switch value.(type) {
	case string:
		result := NewMarkupContentFromLegacy(value.(string))
		return &result
	case MarkupContent:
		result := value.(MarkupContent)
		return &result
	default:
		return nil
	}
}
