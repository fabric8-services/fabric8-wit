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
	t.Log("Initial multi level map : ", nestedMap)
	t.Log("Flattened map : ", OneLevelMap)
}
