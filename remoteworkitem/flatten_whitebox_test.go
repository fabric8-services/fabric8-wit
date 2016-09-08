package remoteworkitem

import (
	"encoding/json"
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

func TestFlattenMapWhitebox(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	testString := []byte(`{"admins":[{"name":"aslak"}],"name":"shoubhik", "assignee":{"fixes": 2, "complete" : true,"foo":[ 1,2,3,4],"1":"sbose","2":"pranav","participants":{"4":"sbose56","5":"sbose78"}},"name":"shoubhik"}`)
	var nestedMap map[string]interface{}
	err := json.Unmarshal(testString, &nestedMap)

	if err != nil {
		t.Error("Incorrect dataset ", testString)
	}
	OneLevelMap := make(map[string]interface{})

	flatten(OneLevelMap, nestedMap, nil)

	// Test for string
	assert.Equal(t, OneLevelMap["assignee.participants.4"], "sbose56", "Incorrect mapping found for assignee.participants.4")

	// test for int
	assert.Equal(t, int(OneLevelMap["assignee.fixes"].(float64)), 2)

	// test for array
	assert.Equal(t, OneLevelMap["assignee.foo.0"], float64(1))

	// test for boolean
	assert.Equal(t, OneLevelMap["assignee.complete"], true)

	// test for array of object(s)
	assert.Equal(t, OneLevelMap["admins.0.name"], "aslak")
}

func TestConvertArrayToMap(t *testing.T) {
	testArray := []interface{}{1, 2, 3, 4}
	testMap := convertArrayToMap(testArray)

	assert.Equal(t, testMap["0"], 1)
	assert.Equal(t, testMap["3"], 4)

}
