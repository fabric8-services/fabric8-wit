package remoteworkitem

import (
	"encoding/json"
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

func TestFlattenMap(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// test string contains int, list, string, null, boolean values to test
	testString := []byte(`{"name":"shoubhik", "contact":{"email": null}, "assignee":{"fixes": 2, "complete" : true,"foo":[1,2,3,4],"1":"sbose","2":"pranav","participants":{"4":"sbose56","5":"sbose78"}}}`)
	var nestedMap map[string]interface{}
	err := json.Unmarshal(testString, &nestedMap)

	if err != nil {
		t.Fatal("Incorrect dataset ", testString)
	}

	OneLevelMap := Flatten(nestedMap)

	// Test for string
	assert.Equal(t, OneLevelMap["assignee.participants.4"], "sbose56", "Incorrect mapping found for assignee.participants.4")

	// test for int
	assert.Equal(t, int(OneLevelMap["assignee.fixes"].(float64)), 2)

	// test for array - TODO: Need a better way to handle the automatic conversion of int to float64 during implicit unmarshal )
	refArray := []interface{}{float64(1), float64(2), float64(3), float64(4)}
	assert.Equal(t, OneLevelMap["assignee.foo"], refArray)

	// test for boolean
	assert.Equal(t, OneLevelMap["assignee.complete"], true)

	// test for NULL
	assert.Equal(t, nil, OneLevelMap["contact.email"])
}
