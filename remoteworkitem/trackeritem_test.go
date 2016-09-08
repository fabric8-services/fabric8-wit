package remoteworkitem

import (
	"testing"

	"github.com/almighty/almighty-core/resource"
)

func TestUpload(t *testing.T) {
	resource.Require(t, resource.Database)
	db.Exec(`DELETE FROM "tracker_items"`)
	tr := Tracker{URL: "https://api.github.com/", Type: "github"}
	db.Create(&tr)
	tq := TrackerQuery{Query: "some random query", Schedule: "0 0 0 * * *", TrackerID: tr.ID}
	db.Create(&tq)
	db.Delete(&tq)
	db.Delete(&tr)
	i := map[string]string{"content": "some text", "id": "https://github.com/golang/go/issues/124", "batch_id": "10"}

	// create
	err := upload(db, int(tq.ID), i)
	if err != nil {
		t.Error("Create error:", err)
	}
	ti1 := TrackerItem{}
	db.Where("remote_item_id = ?", i["id"]).Find(&ti1)
	if ti1.Item != i["content"] {
		t.Errorf("Content not saved: %s", i["content"])
	}
	if ti1.BatchID != i["batch_id"] {
		t.Errorf("Batch ID not saved: %s", i["batch_id"])
	}
	if ti1.TrackerQueryID != tq.ID {
		t.Errorf("Tracker query ID not saved: %s", tq.ID)
	}

	i = map[string]string{"content": "some text 2", "id": "https://github.com/golang/go/issues/124", "batch_id": "11"}
	// update
	err = upload(db, int(tq.ID), i)
	if err != nil {
		t.Error("Update error:", err)
	}
	ti2 := TrackerItem{}
	db.Where("remote_item_id = ?", i["id"]).Find(&ti2)
	if ti2.Item != i["content"] {
		t.Errorf("Content not saved: %s", i["content"])
	}
	if ti2.BatchID != i["batch_id"] {
		t.Errorf("Batch ID not saved: %s", i["batch_id"])
	}
	if ti2.TrackerQueryID != tq.ID {
		t.Errorf("Tracker query ID not saved: %s", tq.ID)
	}
	var count int
	db.Model(&TrackerItem{}).Where("remote_item_id = ?", i["id"]).Count(&count)
	if count > 1 {
		t.Errorf("More records found: %d", count)
	}
}
