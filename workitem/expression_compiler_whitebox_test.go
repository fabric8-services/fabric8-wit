package workitem

import (
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

func TestUnderscore(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	assert.Equal(t, "type_id", underscore("TypeID"))
	assert.Equal(t, "type_id", underscore("type_ID"))
	assert.Equal(t, "type_id", underscore("type_id"))
}
