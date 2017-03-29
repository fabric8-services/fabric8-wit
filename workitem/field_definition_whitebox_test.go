package workitem

import (
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

func TestCompatibleFields(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	a := FieldDefinition{
		Label:       "a",
		Description: "description for 'a'",
		Required:    true,
		Type: ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindString},
		},
	}
	b := FieldDefinition{
		Label:       "b",
		Description: "description for 'b'",
		Required:    true,
		Type: ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindString},
		},
	}
	assert.True(t, compatibleFields(a, b))
}
