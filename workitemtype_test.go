package main

import (
	"testing"
	//"strings"

	"github.com/almighty/almighty-core/app/test"
	skipper "github.com/almighty/almighty-core/test"
)

func TestGetWorkItemType(t *testing.T) {
	skipper.SkiptTestIfNotUnitTest(t)

	typeController := WorkitemtypeController{}
	_, resp2 := test.ShowWorkitemtypeOK(t, nil, nil, &typeController, "1")
	if resp2 == nil {
		t.Error("Could not read type 1")
	}
}
