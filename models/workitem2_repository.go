package models

import (
	"golang.org/x/net/context"

	"log"
	"strconv"

	"fmt"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

const (
	AssigneeType = "identities"
)

type GormWorkItem2Repository struct {
	db  *gorm.DB
	wir *GormWorkItemTypeRepository
}

// Used for dev-testing. This is temparory method. Will be removed/moved_to_test package soon.
func createOneRandomUserIdentity(ctx context.Context, db *gorm.DB) {
	newUUID := uuid.NewV4()
	fmt.Println("new UUId is -> ", newUUID)
	identityRepo := account.NewIdentityRepository(db)
	identity := account.Identity{
		FullName: "Test User Integration Random",
		ImageURL: "http://images.com/42",
		ID:       uuid.NewV4(),
	}
	err := identityRepo.Create(ctx, &identity)
	if err != nil {
		fmt.Println("should not happen off.")
	}
}

// Save updates the given work item in storage. Version must be the same as the one int the stored version
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItem2Repository) Save(ctx context.Context, wi app.WorkItemDataForUpdate) (*app.WorkItem, error) {
	createOneRandomUserIdentity(ctx, r.db) // will be removed too.
	res := WorkItem{}
	id, err := strconv.ParseUint(wi.ID, 10, 64)
	if err != nil {
		return nil, NotFoundError{entity: "work item", ID: wi.ID}
	}

	log.Printf("looking for id %d", id)
	tx := r.db
	if tx.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NewNotFoundError("work item", wi.ID)
	}

	if res.Version != wi.Attributes.Version {
		return nil, VersionConflictError{simpleError{"version conflict"}}
	}
	wiType, err := r.wir.LoadTypeFromDB(ctx, wi.Attributes.Type)
	if err != nil {
		return nil, NewBadParameterError("Type", wi.Type)
	}

	newWi := WorkItem{
		ID:      id,
		Type:    wi.Attributes.Type,
		Version: wi.Attributes.Version + 1,
		Fields:  Fields{},
	}

	rel := wi.Relationships
	if rel != nil && rel.Assignee != nil && rel.Assignee.Data != nil {
		fmt.Println("checking for relationships started !")
		assigneeData := rel.Assignee.Data
		identityRepo := account.NewIdentityRepository(r.db)
		uuidStr := assigneeData.ID
		assigneeUUID, err := uuid.FromString(uuidStr)
		if err != nil {
			return nil, NewBadParameterError("data.relationships.assignee.data.id", uuidStr)
		}
		_, err = identityRepo.Load(ctx, assigneeUUID)
		if err != nil {
			return nil, NewBadParameterError("data.relationships.assignee.data.id", uuidStr)
		}

		// overwrite assignee for now;
		wi.Attributes.Fields[SystemAssignee] = uuidStr
		//  ToDO : make it a list and append
		// existingAssignees := res.Fields[SystemAssignee]
		// wi.Attributes.Fields[SystemAssignee] = append(existingAssignees, uuidStr)
	}

	for fieldName, fieldDef := range wiType.Fields {
		fieldValue := wi.Attributes.Fields[fieldName]
		var err error
		fmt.Println("Now setting", fieldName, fieldValue)
		newWi.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			return nil, NewBadParameterError(fieldName, fieldValue)
		}
	}

	if err := tx.Save(&newWi).Error; err != nil {
		log.Print(err.Error())
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Printf("updated item to %v\n", newWi)
	result, err := wiType.ConvertFromModel(newWi)
	if err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	return result, nil
}
