// +build integration

package main

import (
	"testing"
	//"strings"

	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/models"
)

func TestGetWorkItemType(t *testing.T) {
	ts := models.NewGormTransactionSupport(db)
	witRepo := models.NewWorkItemTypeRepository(ts)
	typeController := WorkitemtypeController{ts: ts, witRepository: witRepo}
	_, resp2 := test.ShowWorkitemtypeOK(t, nil, nil, &typeController, "system.issue")
	if resp2 == nil {
		t.Error("Could not read type system.issue")
	}
}
