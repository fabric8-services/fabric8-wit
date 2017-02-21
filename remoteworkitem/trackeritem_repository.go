package remoteworkitem

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
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

// Map a remote work item into an ALM work item and persist it into the database.
func convert(db *gorm.DB, tID int, item TrackerItemContent, providerType string) (*app.WorkItem, error) {
	remoteID := item.ID
	content := string(item.Content)
	trackerItem := TrackerItem{Item: content, RemoteItemID: remoteID, TrackerID: uint64(tID)}

	// Converting the remote item to a local work item
	remoteTrackerItemConvertFunc, ok := RemoteWorkItemImplRegistry[providerType]
	if !ok {
		return nil, BadParameterError{parameter: providerType, value: providerType}
	}
	remoteTrackerItem, err := remoteTrackerItemConvertFunc(trackerItem)
	if err != nil {
		return nil, InternalError{simpleError{message: fmt.Sprintf(" Error parsing the tracker data: %s", err.Error())}}
	}
	remoteWorkItem, err := Map(remoteTrackerItem, RemoteWorkItemKeyMaps[providerType])
	if err != nil {
		return nil, ConversionError{simpleError{message: fmt.Sprintf("Error mapping to local work item: %s", err.Error())}}
	}
	workItem, err := bindAssignees(db, remoteWorkItem, providerType)
	if err != nil {
		return nil, InternalError{simpleError{message: fmt.Sprintf("Error bind assignees: %s", err.Error())}}
	}

	return upsert(db, *workItem)
}

func bindAssignees(db *gorm.DB, remoteWorkItem RemoteWorkItem, providerType string) (*app.WorkItem, error) {
	identityRepository := account.NewIdentityRepository(db)
	workItem := app.WorkItem{
		ID:     remoteWorkItem.ID,
		Type:   remoteWorkItem.Type,
		Fields: make(map[string]interface{}),
	}
	// copy all fields from remoteworkitem into result workitem
	for fieldName, fieldValue := range remoteWorkItem.Fields {
		if fieldName == remoteAssigneeProfileURLs {
			if fieldValue == nil {
				workItem.Fields[workitem.SystemAssignees] = make([]string, 0)
				continue
			}
			identities := make([]string, 0)
			assigneeLogins := fieldValue.([]string)
			assigneeProfileURLs := remoteWorkItem.Fields[remoteAssigneeProfileURLs].([]string)
			for i, assigneeLogin := range assigneeLogins {
				assigneeProfileURL := assigneeProfileURLs[i]
				log.Debug(nil, map[string]interface{}{
					"pkg":      "remoteworkitem",
					"workitem": workItem,
				}, "Looking for identity of user with profile URL=%s\n", assigneeProfileURL)
				// bind the assignee to an existing identity, or create a new one
				identity, err := identityRepository.First(account.IdentityFilterByProfileURL(assigneeProfileURL))
				if err != nil {
					return nil, err
				}
				if identity == nil {
					// create the identity if it does not exist yet
					log.Debug(nil, map[string]interface{}{
						"pkg":      "remoteworkitem",
						"workitem": workItem,
					}, "Creating an identity for username '%s' with profile '%s' on '%s'\n", assigneeLogin, assigneeProfileURL, providerType)
					identity := &account.Identity{
						ProviderType: providerType,
						Username:     assigneeLogin,
						ProfileURL:   assigneeProfileURL,
					}
					err = identityRepository.Create(context.Background(), identity)
					if err != nil {
						return nil, err
					}
					identities = append(identities, identity.ID.String())
				} else {
					// use existing identity
					log.Debug(nil, map[string]interface{}{
						"pkg":      "remoteworkitem",
						"workitem": workItem,
					}, "Using existing identity with ID: %v", identity.ID.String())
					identities = append(identities, identity.ID.String())
				}
			}
			// associate the identities to the work item
			workItem.Fields[workitem.SystemAssignees] = identities
		} else if fieldName == remoteAssigneeProfileURLs {
			// ignore here, it is being processed above
		} else {
			// copy other fields
			workItem.Fields[fieldName] = fieldValue
		}
	}
	return &workItem, nil
}

func upsert(db *gorm.DB, workItem app.WorkItem) (*app.WorkItem, error) {
	wir := workitem.NewWorkItemRepository(db)
	// Get the remote item identifier ( which is currently the url ) to check if the work item exists in the database.
	workItemRemoteID := workItem.Fields[workitem.SystemRemoteItemID]
	log.Info(nil, map[string]interface{}{
		"pkg":      "remoteworkitem",
		"workitem": workItem,
	}, "Upsert on workItemRemoteID=%s", workItemRemoteID)
	// Querying the database to fetch the work item (if it exists)
	sqlExpression := criteria.Equals(criteria.Field(workitem.SystemRemoteItemID), criteria.Literal(workItemRemoteID))
	existingWorkItem, err := wir.Fetch(context.Background(), sqlExpression)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var resultWorkItem *app.WorkItem
	if existingWorkItem != nil {
		log.Info(nil, map[string]interface{}{
			"pkg":      "remoteworkitem",
			"workitem": workItem,
		}, "Workitem exists, will be updated")
		for key, value := range workItem.Fields {
			existingWorkItem.Fields[key] = value
		}
		resultWorkItem, err = wir.Save(context.Background(), *existingWorkItem)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	} else {
		log.Info(nil, map[string]interface{}{
			"pkg":      "remoteworkitem",
			"workitem": workItem,
		}, "Workitem does not exist, will be created")
		c := workItem.Fields[workitem.SystemCreator]
		var creator string
		if c != nil {
			creator = c.(string)
		}
		resultWorkItem, err = wir.Create(context.Background(), workitem.SystemBug, workItem.Fields, creator)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}
	log.Info(nil, map[string]interface{}{
		"pkg":      "remoteworkitem",
		"workitem": workItem,
	}, "Result workitem: %v", resultWorkItem)

	return resultWorkItem, nil

}
