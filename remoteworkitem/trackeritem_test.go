package remoteworkitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/suite"
)

type TestTrackerItemRepository struct {
	gormtestsupport.DBTestSuite
}

func TestRunTrackerItemRepository(t *testing.T) {
	suite.Run(t, &TestTrackerItemRepository{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (test *TestTrackerItemRepository) TestUpload() {
	t := test.T()
	resource.Require(t, resource.Database)

	test.DB.Exec(`DELETE FROM "tracker_items"`)
	tr := remoteworkitem.Tracker{URL: "https://api.github.com/", Type: "github"}
	test.DB.Create(&tr)
	tq := remoteworkitem.TrackerQuery{Query: "some random query", Schedule: "0 0 0 * * *", TrackerID: tr.ID}
	test.DB.Create(&tq)
	test.DB.Delete(&tq)
	test.DB.Delete(&tr)
	i := remoteworkitem.TrackerItemContent{Content: []byte("some text"), ID: "https://github.com/golang/go/issues/124"}

	// create
	err := remoteworkitem.Upload(test.DB, tr.ID, i)
	if err != nil {
		t.Error("Create error:", err)
	}
	ti1 := remoteworkitem.TrackerItem{}
	test.DB.Where("remote_item_id = ? AND tracker_id = ?", i.ID, tr.ID).Find(&ti1)
	if ti1.Item != string(i.Content) {
		t.Errorf("Content not saved: %s", i.Content)
	}
	if ti1.TrackerID != tr.ID {
		t.Errorf("Tracker ID not saved: %d", tr.ID)
	}

	i = remoteworkitem.TrackerItemContent{Content: []byte("some text 2"), ID: "https://github.com/golang/go/issues/124"}
	// update
	err = remoteworkitem.Upload(test.DB, tr.ID, i)
	if err != nil {
		t.Error("Update error:", err)
	}
	ti2 := remoteworkitem.TrackerItem{}
	test.DB.Where("remote_item_id = ? AND tracker_id = ?", i.ID, tr.ID).Find(&ti2)
	if ti2.Item != string(i.Content) {
		t.Errorf("Content not saved: %s", i.Content)
	}
	if ti2.TrackerID != tr.ID {
		t.Errorf("Tracker ID not saved: %d", tq.ID)
	}
	var count int
	test.DB.Model(&remoteworkitem.TrackerItem{}).Where("remote_item_id = ?", i.ID).Count(&count)
	if count > 1 {
		t.Errorf("More records found: %d", count)
	}
}
