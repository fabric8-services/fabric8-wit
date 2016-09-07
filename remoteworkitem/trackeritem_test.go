package remoteworkitem

import (
	"testing"

	"github.com/almighty/almighty-core/resource"
)

func TestUpload(t *testing.T) {
	resource.Require(t, resource.Database)
	t := Tracker{URL: "https://api.github.com/", Type: "github"}
	db.Create(&t)
	tq := TrackerQuery{Query: "some random query", Schedule: "0 0 0 * * *", t.ID}
	db.Create(&tq)
	//ti := TrackerItem{Item: "{}", RemoteItemID: "https://github.com/golang/go/issues/124", BatchID: "1", TrackerID: tq.ID}
	i := map[string]string{"item": ""}
	upload(db, tq.ID)
}
