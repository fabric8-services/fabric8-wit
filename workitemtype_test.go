// +build unit

package main

import (
	"testing"
	//"strings"

	"github.com/almighty/almighty-core/app/test"
)

func TestGetWorkItemType(t *testing.T) {

	typeController := WorkitemtypeController{}
	_, resp2 := test.ShowWorkitemtypeOK(t, nil, nil, &typeController, "1")
	if resp2 == nil {
		t.Error("Could not read type 1")
	}
}
