package main

import (
	"testing"

	"time"

	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
)

func TestShowStatusOK(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.Database)
	controller := StatusController{db: DB}
	_, res := test.ShowStatusOK(t, nil, nil, &controller)

	if res.Commit != "0" {
		t.Error("Commit not found")
	}
	if res.StartTime != StartTime {
		t.Error("StartTime is not correct")
	}
	_, err := time.Parse("2006-01-02T15:04:05Z", res.StartTime)
	if err != nil {
		t.Error("Incorrect layout of StartTime: ", err.Error())
	}
}

func TestNewStatusController(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	svc := goa.New("TestNewStatusControllerService")
	assert.NotNil(t, NewStatusController(svc, nil))
}
