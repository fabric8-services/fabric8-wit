package remoteworkitem

import (
	"testing"

	"github.com/almighty/almighty-core/test/resource"
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
	i := TrackerItemContent{Content: []byte("some text"), ID: "https://github.com/golang/go/issues/124"}

	// create
	err := upload(db, int(tr.ID), i)
	if err != nil {
		t.Error("Create error:", err)
	}
	ti1 := TrackerItem{}
	db.Where("remote_item_id = ? AND tracker_id = ?", i.ID, tr.ID).Find(&ti1)
	if ti1.Item != string(i.Content) {
		t.Errorf("Content not saved: %s", i.Content)
	}
	if ti1.TrackerID != tr.ID {
		t.Errorf("Tracker ID not saved: %d", tr.ID)
	}

	i = TrackerItemContent{Content: []byte("some text 2"), ID: "https://github.com/golang/go/issues/124"}
	// update
	err = upload(db, int(tr.ID), i)
	if err != nil {
		t.Error("Update error:", err)
	}
	ti2 := TrackerItem{}
	db.Where("remote_item_id = ? AND tracker_id = ?", i.ID, tr.ID).Find(&ti2)
	if ti2.Item != string(i.Content) {
		t.Errorf("Content not saved: %s", i.Content)
	}
	if ti2.TrackerID != tr.ID {
		t.Errorf("Tracker ID not saved: %d", tq.ID)
	}
	var count int
	db.Model(&TrackerItem{}).Where("remote_item_id = ?", i.ID).Count(&count)
	if count > 1 {
		t.Errorf("More records found: %d", count)
	}
}
