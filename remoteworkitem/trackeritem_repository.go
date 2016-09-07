package remoteworkitem

import "github.com/jinzhu/gorm"

// upload imports the items into database
func upload(db *gorm.DB, tqID int, item map[string]string) error {
	ti := TrackerItem{Item: item["content"], RemoteItemID: item["id"], BatchID: item["batch_id"], TrackerQueryID: uint64(tqID)}
	db.Where("remote_item_id = ?", item["id"]).Find(&ti)
	var err error
	if ti.ID == 0 {
		ti1 := TrackerItem{Item: item["content"], RemoteItemID: item["id"], BatchID: item["batch_id"], TrackerQueryID: uint64(tqID)}
		err = db.Create(&ti1).Error
	} else {
		ti2 := TrackerItem{ID: ti.ID, Item: item["content"], RemoteItemID: item["id"], BatchID: item["batch_id"], TrackerQueryID: uint64(tqID)}
		err = db.Save(&ti2).Error
	}
	return err
}
