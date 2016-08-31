package main

import (
	"testing"
	//"strings"

	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/resource"
)

func TestGetWorkItemType(t *testing.T) {
	resource.Require(t, resource.None)

	typeController := WorkitemtypeController{}
	_, resp2 := test.ShowWorkitemtypeOK(t, nil, nil, &typeController, "1")
	if resp2 == nil {
		t.Error("Could not read type 1")
	}
}
