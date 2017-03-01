package workitem

import (
	"testing"

	. "github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/resource"
	. "github.com/almighty/almighty-core/workitem"
	"github.com/stretchr/testify/assert"
)

func TestUnderscore(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	assert.Equal(t, unserscore("TypeID"), "type_id")
	assert.Equal(t, unserscore("type_ID"), "type_id")
	assert.Equal(t, unserscore("type_id"), "type_id")
}
