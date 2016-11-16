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
	identityRepo := account.NewIdentityRepository(db)
	identity := account.Identity{
		FullName: "Test User Integration Random",
		ImageURL: "http://images.com/42",
		ID:       newUUID,
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

	// Attributes is a string->string map hence need to add few conditions
	var version int
	// validate version attribute
	if _, ok := wi.Attributes["version"]; ok {
		version, err = strconv.Atoi(wi.Attributes["version"])
		if err != nil {
			return nil, NewBadParameterError("version", version)
		}
	} else {
		return nil, VersionConflictError{simpleError{"version is mandatory"}}
	}
	if res.Version != version {
		return nil, VersionConflictError{simpleError{"version conflict"}}
	}

	rel := wi.Relationships
	// take out workItemType from relationship
	// It is mandatory and type must be workitemtypes (only one enum provided in design)
	// Hence direct access is possible
	inputWIType := rel.BaseType.Data.ID

	wiType, err := r.wir.LoadTypeFromDB(ctx, inputWIType)
	if err != nil {
		return nil, NewBadParameterError("Type", wi.Type)
	}

	newWi := WorkItem{
		ID:      id,
		Type:    inputWIType,
		Version: version + 1,
		Fields:  res.Fields,
	}

	if rel != nil && rel.Assignee != nil && rel.Assignee.Data != nil {
		assigneeData := rel.Assignee.Data
		identityRepo := account.NewIdentityRepository(r.db)
		uuidStr := assigneeData.ID
		assigneeUUID, err := uuid.FromString(uuidStr)
		if err != nil {
			return nil, NewBadParameterError("data.relationships.assignee.data.id should be UUID", uuidStr)
		}
		_, err = identityRepo.Load(ctx, assigneeUUID)
		if err != nil {
			return nil, NewBadParameterError("data.relationships.assignee.data.id not found", uuidStr)
		}

		// overwrite assignee for now;
		wi.Attributes[SystemAssignee] = uuidStr
		//  ToDO : make it a list and append
		// existingAssignees := res.Fields[SystemAssignee]
		// wi.Attributes.Fields[SystemAssignee] = append(existingAssignees, uuidStr)
	}

	for fieldName, fieldDef := range wiType.Fields {
		fieldValue, exist := wi.Attributes[fieldName]
		if !exist {
			// skip non-mentioned Attributes because this is a PATCH request.
			continue
		}
		var err error
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
