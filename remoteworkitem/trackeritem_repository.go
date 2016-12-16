package remoteworkitem

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
)

// upload imports the items into database
func upload(db *gorm.DB, tID int, item TrackerItemContent) error {
	remoteID := item.ID
	content := string(item.Content)

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

func updateTrackerQuery(db *gorm.DB, tqID int, lu *time.Time) error {
	tq := TrackerQuery{}
	tx := db.First(&tq, tqID)
	if tx.RecordNotFound() {
		log.Printf("not found, res=%v", tq)
		return NotFoundError{entity: "tracker_query", ID: string(tq.ID)}
	}
	tq.LastUpdated = lu
	if err := tx.Save(&tq).Error; err != nil {
		log.Print(err.Error())
		return InternalError{simpleError{err.Error()}}
	}
	return nil
}

// Map a remote work item into an ALM work item and persist it into the database.
func convert(db *gorm.DB, tID int, item TrackerItemContent, provider string) (*app.WorkItem, error) {
	remoteID := item.ID
	content := string(item.Content)

	wir := workitem.NewWorkItemRepository(db)
	ti := TrackerItem{Item: content, RemoteItemID: remoteID, TrackerID: uint64(tID)}

	// Converting the remote item to a local work item
	remoteTrackerItemMethodRef, ok := RemoteWorkItemImplRegistry[provider]
	if !ok {
		return nil, BadParameterError{parameter: provider, value: provider}
	}
	remoteTrackerItem, err := remoteTrackerItemMethodRef(ti)
	if err != nil {
		return nil, InternalError{simpleError{message: " Error parsing the tracker data "}}
	}
	workItem, err := Map(remoteTrackerItem, WorkItemKeyMaps[provider])
	if err != nil {
		return nil, ConversionError{simpleError{message: " Error mapping to local work item "}}
	}

	// Get the remote item identifier ( which is currently the url ) to check if the work item exists in the database.
	workItemRemoteID := workItem.Fields[workitem.SystemRemoteItemID]

	sqlExpression := criteria.Equals(criteria.Field(workitem.SystemRemoteItemID), criteria.Literal(workItemRemoteID))

	var newWorkItem *app.WorkItem

	// Querying the database
	existingWorkItems, _, err := wir.List(context.Background(), sqlExpression, nil, nil)
	if err != nil {
		return nil, err
	}

	if len(existingWorkItems) != 0 {
		fmt.Println("Workitem exists, will be updated")
		existingWorkItem := existingWorkItems[0]
		for key, value := range workItem.Fields {
			existingWorkItem.Fields[key] = value
		}
		newWorkItem, err = wir.Save(context.Background(), *existingWorkItem)
		if err != nil {
			fmt.Println("Error updating work item : ", err)
		}
	} else {
		fmt.Println("Work item not found , will now create new work item")
		c := workItem.Fields[workitem.SystemCreator]
		var creator string
		if c != nil {
			creator = c.(string)
		}
		newWorkItem, err = wir.Create(context.Background(), workitem.SystemBug, workItem.Fields, creator)
		if err != nil {
			fmt.Println("Error creating work item : ", err)
		}
	}
	return newWorkItem, err
}
