package main

import (
	"testing"

	"golang.org/x/net/context"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/remoteworkitem"
)

// Ref: https://github.com/almighty/almighty-core/issues/189
func TestCreateTrackerValidId(t *testing.T) {

	ctx := context.Background()
	ts := remoteworkitem.NewGormTransactionSupport(db)

	repo := remoteworkitem.NewTrackerRepository(ts)
	tracker, err := repo.Create(ctx, "github.com", "github")
	if err != nil {
		t.Error("Failed to setup scenario", err)
	}

	controller := TrackerController{ts: ts, tRepository: repo}
	_, created := test.ShowTrackerOK(t, nil, nil, &controller, tracker.ID)
	if created.ID != tracker.ID {
		t.Error("Failed because fetched Tracker not same as requested. Found: ", tracker.ID, " Expected, ", created.ID)
	}
}
