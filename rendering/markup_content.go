package rendering

// MarkupContent defines the raw content of a field along with the markup language used to input the content.
type MarkupContent struct {
	Content  string `json:"content"`
	Markup   Markup `json:"markup"`
	Rendered string `json:"rendered,omitempty"`
}

const (
	// ContentKey is the key for the 'content' field when the MarkupContent is
	// converted into/from a Map
	ContentKey = "content"
	// MarkupKey is the key for the 'markup' field when the MarkupContent is
	// converted into/from a Map
	MarkupKey = "markup"
	// RenderedKey is the key for the 'rendered' field when the MarkupContent is
	// converted into/from a Map
	RenderedKey = "rendered"
)

// ToMap returns the markup content object as a map
func (markupContent MarkupContent) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		ContentKey:  markupContent.Content,
		RenderedKey: markupContent.Rendered,
		MarkupKey:   SystemMarkupDefault,
	}
	if markupContent.Markup != "" {
		result[MarkupKey] = markupContent.Markup
	}
	return result
}

// NewMarkupContentFromMap creates a MarkupContent from the given Map,
// filling the 'Markup' field with the default value if no entry was found in the input or if the given markup is not supported.
// This avoids filling the DB with invalid markup types.
func NewMarkupContentFromMap(value map[string]interface{}) MarkupContent {
	res := MarkupContent{
		Markup: SystemMarkupDefault,
	}
	if value == nil {
		return res
	}
	if content, ok := value[ContentKey].(string); ok {
		res.Content = content
	}
	if rendered, ok := value[RenderedKey].(string); ok {
		res.Rendered = rendered
	}
	if markupVal, ok := value[MarkupKey]; ok {
		if markupStr, ok := markupVal.(string); ok {
			markup := Markup(markupStr)
			// use default markup if the input is not supported
			if err := markup.CheckValid(); err == nil {
				res.Markup = markup
			}
		}
	}
	return res
}

// NewMarkupContentFromLegacy creates a MarkupContent from the given content, using the default markup.
func NewMarkupContentFromLegacy(content string) MarkupContent {
	return MarkupContent{Content: content, Markup: SystemMarkupDefault}
}

// NewMarkupContent creates a MarkupContent from the given content, using the default markup.
func NewMarkupContent(content string, markup Markup) MarkupContent {
	return MarkupContent{Content: content, Markup: markup}
}

// NewMarkupContentFromValue creates a MarkupContent from the given value, by
// converting a 'string', a 'map[string]interface{}', or a 'MarkupContent';
// otherwise, it returns nil.
func NewMarkupContentFromValue(value interface{}) *MarkupContent {
	if value == nil {
		return nil
	}
	switch t := value.(type) {
	case string:
		result := NewMarkupContentFromLegacy(t)
		return &result
	case MarkupContent:
		return &t
	case map[string]interface{}:
		result := NewMarkupContentFromMap(t)
		return &result
	default:
		return nil
	}
}

// NilSafeGetMarkup returns the given markup if it is not nil, nor empty, nor
// invalid; otherwise it returns the default markup.
func NilSafeGetMarkup(markup *string) Markup {
	res := SystemMarkupDefault
	if markup == nil {
		return res
	}
	if *markup == "" {
		return res
	}
	m := Markup(*markup)
	if m.CheckValid() != nil {
		return res
	}
	return m
}
