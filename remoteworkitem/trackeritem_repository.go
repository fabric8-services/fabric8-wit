package remoteworkitem

import "github.com/jinzhu/gorm"

// upload imports the items into database
func upload(db *gorm.DB, tqID int, item map[string]string) error {
	ti := TrackerItem{Item: item["content"], RemoteItemID: item["id"], BatchID: item["batch_id"], TrackerQueryID: uint64(tqID)}
	//db.Where("remote_item_id = ?", item["id"]).Find(&user)
	err := db.Create(&ti).Error
	return err
}
