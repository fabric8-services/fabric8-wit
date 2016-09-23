package main_test

import (
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
)

func TestGetWorkItem(t *testing.T) {
	resource.Require(t, resource.Database)

	ts := models.NewGormTransactionSupport(DB)
	wir := models.NewWorkItemTypeRepository(ts)
	repo := models.NewWorkItemRepository(ts, wir)
	svc := goa.New("TestGetWorkItem-Service")
	assert.NotNil(t, svc)
	controller := NewWorkitemController(svc, repo, ts)
	assert.NotNil(t, controller)
	payload := app.CreateWorkitemPayload{
		Type: "system.bug",
		Fields: map[string]interface{}{
			"system.title":   "Test WI",
			"system.creator": "aslak",
			"system.state":   "closed"},
	}

	_, result := test.CreateWorkitemCreated(t, nil, nil, controller, &payload)

	_, wi := test.ShowWorkitemOK(t, nil, nil, controller, result.ID)

	if wi == nil {
		t.Fatalf("Work Item '%s' not present", result.ID)
	}

	if wi.ID != result.ID {
		t.Errorf("Id should be %s, but is %s", result.ID, wi.ID)
	}

	wi.Fields["system.creator"] = "thomas"
	payload2 := app.UpdateWorkitemPayload{
		Type:    wi.Type,
		Version: wi.Version,
		Fields:  wi.Fields,
	}
	_, updated := test.UpdateWorkitemOK(t, nil, nil, controller, wi.ID, &payload2)
	if updated.Version != result.Version+1 {
		t.Errorf("expected version %d, but got %d", result.Version+1, updated.Version)
	}
	if updated.ID != result.ID {
		t.Errorf("id has changed from %s to %s", result.ID, updated.ID)
	}
	if updated.Fields["system.creator"] != "thomas" {
		t.Errorf("expected creator %s, but got %s", "thomas", updated.Fields["system.creator"])
	}

	test.DeleteWorkitemOK(t, nil, nil, controller, result.ID)
}

func TestCreateWI(t *testing.T) {
	resource.Require(t, resource.Database)
	ts := models.NewGormTransactionSupport(DB)
	wir := models.NewWorkItemTypeRepository(ts)
	repo := models.NewWorkItemRepository(ts, wir)
	svc := goa.New("TestCreateWI-Service")
	assert.NotNil(t, svc)
	controller := NewWorkitemController(svc, repo, ts)
	assert.NotNil(t, controller)
	payload := app.CreateWorkitemPayload{
		Type: "system.bug",
		Fields: map[string]interface{}{
			"system.title":   "Test WI",
			"system.creator": "tmaeder",
			"system.state":   "new",
		},
	}

	_, created := test.CreateWorkitemCreated(t, nil, nil, controller, &payload)
	if created.ID == "" {
		t.Error("no id")
	}
}

func TestListByFields(t *testing.T) {
	resource.Require(t, resource.Database)
	ts := models.NewGormTransactionSupport(DB)
	wir := models.NewWorkItemTypeRepository(ts)
	repo := models.NewWorkItemRepository(ts, wir)
	svc := goa.New("TestListByFields-Service")
	assert.NotNil(t, svc)
	controller := NewWorkitemController(svc, repo, ts)
	assert.NotNil(t, controller)
	payload := app.CreateWorkitemPayload{
		Type: "system.bug",
		Fields: map[string]interface{}{
			"system.title":   "run integration test",
			"system.creator": "aslak",
			"system.state":   "closed"},
	}

	_, wi := test.CreateWorkitemCreated(t, nil, nil, controller, &payload)

	filter := "{\"system.title\":\"run integration test\"}"
	page := "0,1"
	_, result := test.ListWorkitemOK(t, nil, nil, controller, &filter, &page)

	if result == nil {
		t.Errorf("nil result")
	}

	if len(result) != 1 {
		t.Errorf("unexpected length, should be %d but is %d", 1, len(result))
	}

	filter = "{\"system.creator\":\"aslak\"}"
	_, result = test.ListWorkitemOK(t, nil, nil, controller, &filter, &page)

	if result == nil {
		t.Errorf("nil result")
	}

	if len(result) != 1 {
		t.Errorf("unexpected length, should be %d but is %d ", 1, len(result))
	}

	test.DeleteWorkitemOK(t, nil, nil, controller, wi.ID)
}
