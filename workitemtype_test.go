// +build integration

package main

import (
	"strconv"
	"testing"
	"time"
	//"strings"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/models"
)

func reandomUniqueString() string {
	// generates random string using timestamp
	now := time.Now().UnixNano() / 1000000
	return strconv.FormatInt(now, 10)
}

func TestCreateShowWorkItemType(t *testing.T) {
	// test-1
	// create a WIT (using workitem_repo)
	// try to create anohter WIT with same name - should Fail
	// remove the WIT (using GORM raw query)

	ts := models.NewGormTransactionSupport(db)
	witr := models.NewWorkItemTypeRepository(ts)
	controller := WorkitemtypeController{ts: ts, witRepository: witr}
	name := reandomUniqueString()

	st := app.FieldType{
		Kind: "user",
	}
	fd := app.FieldDefinition{
		Type:     &st,
		Required: true,
	}
	fields := map[string]*app.FieldDefinition{
		"system.owner": &fd,
	}
	payload := app.CreateWorkitemtypePayload{
		ExtendedTypeName: nil,
		Name:             name,
		Fields:           fields,
	}
	t.Log("creating WIT now.")
	_, created := test.CreateWorkitemtypeCreated(t, nil, nil, &controller, &payload)

	if created.Name == "" {
		t.Error("no Name")
	}
	t.Log("WIT created, Name=", created.Name)

	t.Log("Fetch recently created WIT")
	_, showWIT := test.ShowWorkitemtypeOK(t, nil, nil, &controller, name)

	if showWIT == nil {
		t.Error("Can not fetch WIT", name)
	}

	t.Log("Started cleanup for ", created.Name)
	db.Table("work_item_types").Where("name=?", name).Delete(&models.WorkItemType{})
	t.Log("Cleanup complete")
}
