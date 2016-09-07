package remoteworkitem

import "github.com/jinzhu/gorm"

// upload imports the items into database
func upload(db *gorm.DB, tID int, item map[string]string) error {
	remoteID := item["id"]
	content := item["content"]

	var ti TrackerItem
	if db.Where("remote_item_id = ? AND tracker_id = ?", remoteID, tID).Find(&ti).RecordNotFound() {
		ti = TrackerItem{
			Item:         content,
			RemoteItemID: remoteID,
			TrackerID:    uint64(tID)}
		return db.Create(&ti).Error
	}
	ti.Item = content
	return db.Save(&ti).Error
}
