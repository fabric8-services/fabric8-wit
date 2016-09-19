package remoteworkitem

import (
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

func TestConvertArrayToMap(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	testArray := []interface{}{1, 2, 3, 4}
	testMap := convertArrayToMap(testArray)

	assert.Equal(t, testMap["0"], 1)
	assert.Equal(t, testMap["1"], 2)
	assert.Equal(t, testMap["2"], 3)
	assert.Equal(t, testMap["3"], 4)

}
