package remoteworkitem

import (
	"encoding/json"
	"testing"
)

func TestFlattenMap(t *testing.T) {
	testString := []byte(`{"name":"shoubhik","assignee":{"1":"sbose","2":"pranav","participants":{"4":"sbose56","5":"sbose78"}},"name":"shoubhik"}`)
	var nestedMap map[string]interface{}
	err := json.Unmarshal(testString, &nestedMap)

	if err != nil {
		t.Error("Incorrect dataset ", testString)
	}

	OneLevelMap := Flatten(nestedMap)

	if OneLevelMap["assignee.participants.4"] != "sbose56" {
		t.Error("Incorrect mapping found for assignee.participants.4")
	}
}
