// +build integration

package main

import (
	"testing"
	//"strings"

	"github.com/almighty/almighty-core/app/test"
)

func TestGetWorkItemType(t *testing.T) {
	t.Skip("work in progress hence not testing now.")
	typeController := WorkitemtypeController{}
	_, resp2 := test.ShowWorkitemtypeOK(t, nil, nil, &typeController, "1")
	if resp2 == nil {
		t.Error("Could not read type 1")
	}
}
