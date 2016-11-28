package models

import (
	"golang.org/x/net/context"

	"log"
	"strconv"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// Following constants define "Type" value to be used in jsonapi specification based APIStinrgTypeAssignee
// e.g> Workitem.2 Update/List API
const (
	APIStinrgTypeAssignee = "identities"
	APIStinrgTypeWorkItem = "workitems"
)

type GormWorkItem2Repository struct {
	db  *gorm.DB
	wir *GormWorkItemTypeRepository
}

// Save updates the given work item in storage. Version must be the same as the one int the stored version
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItem2Repository) Save(ctx context.Context, wi app.WorkItemDataForUpdate) (*app.WorkItem, error) {
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
		versionStr := wi.Attributes["version"].(string)
		version, err = strconv.Atoi(versionStr)
		if err != nil {
			return nil, NewBadParameterError("version", version)
		}
	} else {
		return nil, VersionConflictError{simpleError{"version is mandatory"}}
	}
	if res.Version != version {
		return nil, VersionConflictError{simpleError{"version conflict"}}
	}

	newWi := WorkItem{
		ID:      id,
		Type:    res.Type, // read WIT from DB object and not from payload relationship
		Version: version + 1,
		Fields:  res.Fields,
	}

	wiType, err := r.wir.LoadTypeFromDB(ctx, newWi.Type)
	if err != nil {
		// ideally should not reach this, if reach it means something went wrong while CREATE WI
		return nil, NewBadParameterError("Type", newWi.Type)
	}

	rel := wi.Relationships
	// TODO
	// if rel.Assignee.Data == nil then remove the relationship for WI
	if rel != nil && rel.Assignee != nil && rel.Assignee.Data != nil {
		assigneeData := rel.Assignee.Data
		identityRepo := account.NewIdentityRepository(r.db)
		uuidStr := assigneeData.ID
		if uuidStr == nil {
			// remove Assignee
			wi.Attributes[SystemAssignee] = nil
		} else {
			assigneeUUID, err := uuid.FromString(*uuidStr)
			if err != nil {
				return nil, NewBadParameterError("data.relationships.assignee.data.id", uuidStr)
			}
			_, err = identityRepo.Load(ctx, assigneeUUID)
			if err != nil {
				return nil, NewBadParameterError("data.relationships.assignee.data.id", uuidStr)
			}
			wi.Attributes[SystemAssignee] = *uuidStr
			//  ToDO : make it a list and append
			// existingAssignees := res.Fields[SystemAssignee]
			// wi.Attributes.Fields[SystemAssignee] = append(existingAssignees, uuidStr)
		}
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
		return nil, NewInternalError(err.Error())
	}
	return result, nil
}
