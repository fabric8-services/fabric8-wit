package workitem_test

import (
	"testing"

	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/workitem"
	"github.com/stretchr/testify/assert"
)

func TestNewMarkupContentFromMapWithValidMarkup(t *testing.T) {
	// given
	input := make(map[string]interface{})
	input[workitem.ContentKey] = "foo"
	input[workitem.MarkupKey] = rendering.SystemMarkupMarkdown
	// when
	result := workitem.NewMarkupContentFromMap(input)
	// then
	assert.Equal(t, input[workitem.ContentKey].(string), result.Content)
	assert.Equal(t, input[workitem.MarkupKey].(string), result.Markup)
}

func TestNewMarkupContentFromMapWithInvalidMarkup(t *testing.T) {
	// given
	input := make(map[string]interface{})
	input[workitem.ContentKey] = "foo"
	input[workitem.MarkupKey] = "bar"
	// when
	result := workitem.NewMarkupContentFromMap(input)
	// then
	assert.Equal(t, input[workitem.ContentKey].(string), result.Content)
	assert.Equal(t, rendering.SystemMarkupDefault, result.Markup)
}

func TestNewMarkupContentFromMapWithMissingMarkup(t *testing.T) {
	// given
	input := make(map[string]interface{})
	input[workitem.ContentKey] = "foo"
	// when
	result := workitem.NewMarkupContentFromMap(input)
	// then
	assert.Equal(t, input[workitem.ContentKey].(string), result.Content)
	assert.Equal(t, rendering.SystemMarkupDefault, result.Markup)
}

func TestNewMarkupContentFromMapWithEmptyMarkup(t *testing.T) {
	// given
	input := make(map[string]interface{})
	input[workitem.ContentKey] = "foo"
	input[workitem.MarkupKey] = ""
	// when
	result := workitem.NewMarkupContentFromMap(input)
	// then
	assert.Equal(t, input[workitem.ContentKey].(string), result.Content)
	assert.Equal(t, rendering.SystemMarkupDefault, result.Markup)
}
