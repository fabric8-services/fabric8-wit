package main_test

import (
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
)

func TestSearch(t *testing.T) {
	resource.Require(t, resource.Database)

	service := goa.New("TestSearch-Service")
	wiController := NewWorkitemController(service, gormapplication.NewGormDB(DB))

	wiPayload := app.CreateWorkItemPayload{
		Type: models.SystemBug,
		Fields: map[string]interface{}{
			models.SystemTitle:       "specialwordforsearch",
			models.SystemDescription: "",
			models.SystemCreator:     "baijum",
			models.SystemState:       "closed"},
	}

	_, wiResult := test.CreateWorkitemCreated(t, nil, nil, wiController, &wiPayload)

	controller := NewSearchController(service, gormapplication.NewGormDB(DB))
	q := "specialwordforsearch"
	_, sr := test.ShowSearchOK(t, nil, nil, controller, nil, nil, &q)
	r := sr.Data[0]
	assert.Equal(t, r.Fields[models.SystemTitle], "specialwordforsearch")
	test.DeleteWorkitemOK(t, nil, nil, wiController, wiResult.ID)
}
