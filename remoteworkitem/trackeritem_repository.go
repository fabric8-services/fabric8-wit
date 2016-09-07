package remoteworkitem

import "github.com/jinzhu/gorm"

// upload imports the items into database
func upload(db *gorm.DB, tqID string, item map[string]string) error {
	ti := TrackerItem{Item: item["content"], RemoteItemID: item["id"], BatchID: item["batch_id"], TrackerQueryID: tqID}
	err := db.Create(&ti).Error
	return err
}
