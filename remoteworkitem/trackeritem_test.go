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
	i := map[string]string{"content": "some text", "id": "https://github.com/golang/go/issues/124"}

	// create
	err := upload(db, int(tr.ID), i)
	if err != nil {
		t.Error("Create error:", err)
	}
	ti1 := TrackerItem{}
	db.Where("remote_item_id = ? AND tracker_id = ?", i["id"], tr.ID).Find(&ti1)
	if ti1.Item != i["content"] {
		t.Errorf("Content not saved: %s", i["content"])
	}
	if ti1.TrackerID != tr.ID {
		t.Errorf("Tracker ID not saved: %s", tr.ID)
	}

	i = map[string]string{"content": "some text 2", "id": "https://github.com/golang/go/issues/124"}
	// update
	err = upload(db, int(tr.ID), i)
	if err != nil {
		t.Error("Update error:", err)
	}
	ti2 := TrackerItem{}
	db.Where("remote_item_id = ? AND tracker_id = ?", i["id"], tr.ID).Find(&ti2)
	if ti2.Item != i["content"] {
		t.Errorf("Content not saved: %s", i["content"])
	}
	if ti2.TrackerID != tr.ID {
		t.Errorf("Tracker ID not saved: %s", tq.ID)
	}
	var count int
	db.Model(&TrackerItem{}).Where("remote_item_id = ?", i["id"]).Count(&count)
	if count > 1 {
		t.Errorf("More records found: %d", count)
	}
}
