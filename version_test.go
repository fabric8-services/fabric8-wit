package main

import (
	"testing"

	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/resource"
)

func TestShowVersionOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	controller := VersionController{}
	_, res := test.ShowVersionOK(t, nil, nil, &controller)

	if res.Commit != "0" {
		t.Error("Commit not found")
	}
}
