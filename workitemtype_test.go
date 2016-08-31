package main

import (
	"testing"
	//"strings"

	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/test/providers"
)

func TestGetWorkItemType(t *testing.T) {
	providers.Require(t, providers.UnitTest)

	typeController := WorkitemtypeController{}
	_, resp2 := test.ShowWorkitemtypeOK(t, nil, nil, &typeController, "1")
	if resp2 == nil {
		t.Error("Could not read type 1")
	}
}
