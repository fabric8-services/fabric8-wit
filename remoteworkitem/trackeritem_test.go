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
	// update
	err = upload(db, int(tq.ID), i)
	if err != nil {
		t.Error("Update error:", err)
	}
}
