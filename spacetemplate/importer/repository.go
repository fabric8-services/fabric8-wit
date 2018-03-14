package importer

import (
	"context"
	"fmt"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Repository describes interactions with space templates
type Repository interface {
	// Import creates a new space template and all the artifacts (e.g.
	// work item types, work item link types) in the system. In case a space
	// template or a work item exists, we will update its description, label,
	// icon, title. We don't touch the work item type fields or IDs of any kind.
	Import(ctx context.Context, template ImportHelper) (*ImportHelper, error)
}

// NewRepository creates a new importer repository
func NewRepository(db *gorm.DB) Repository {
	return &GormRepository{db: db}
}

// GormRepository is the implementation of the repository interface for
// importer.
type GormRepository struct {
	db *gorm.DB
}

// Import creates a new space template and all the artifacts (e.g. work item
// types, work item link types) in the system. In case a space template or a
// work item exists, we will update its description, label, icon, title. We
// don't touch the work item type fields or IDs of any kind.
func (r *GormRepository) Import(ctx context.Context, s ImportHelper) (*ImportHelper, error) {
	if err := s.Validate(); err != nil {
		log.Error(ctx, map[string]interface{}{"space_template": s, "err": err}, "space template is invalid")
		return nil, errs.Wrap(err, "space template is invalid")
	}

	res := &s
	stRepo := spacetemplate.NewRepository(r.db)

	// load or create space template
	loadedSpaceTemplate, err := stRepo.Load(ctx, s.Template.ID)
	if err != nil {
		cause := errs.Cause(err)
		switch cause.(type) {
		case errors.NotFoundError:
			created, err := stRepo.Create(ctx, s.Template)
			if err != nil {
				return nil, errs.Wrap(err, "failed to create space template")
			}
			res.Template = *created
		default:
			log.Error(ctx, map[string]interface{}{"space_template_id": s.Template.ID.String(), "err": err}, "failed to load space template")
			return nil, errs.Wrapf(err, "failed to load space template", s.Template.ID)
		}
	} else {
		// Update space template
		loadedSpaceTemplate.Name = s.Template.Name
		loadedSpaceTemplate.Description = s.Template.Description
		if err := loadedSpaceTemplate.Validate(); err != nil {
			log.Error(ctx, map[string]interface{}{"space_template": loadedSpaceTemplate, "err": err}, "update space template is not valid")
			return nil, errs.Wrapf(err, "update space template is not valid %s", s.Template.ID)
		}
		db := r.db.Save(&loadedSpaceTemplate)
		if err := db.Error; err != nil {
			log.Error(ctx, map[string]interface{}{"space_template": loadedSpaceTemplate, "err": err}, "failed to update space template")
			return nil, errs.Wrapf(err, "failed to update space template %s", s.Template.ID)
		}
		res.Template = *loadedSpaceTemplate
	}

	res.WILTs = s.WILTs
	res.WITs = s.WITs
	res.WITGs = s.WITGs

	// Create or update work item types
	if err := r.createOrUpdateWITs(ctx, res); err != nil {
		log.Error(ctx, map[string]interface{}{"space_template": res, "err": err}, "failed to create or update work item types")
		return nil, errs.Wrapf(err, "failed to create or update work item types")
	}

	// Create or update work item link types
	if err := r.createOrUpdateWILTs(ctx, res); err != nil {
		log.Error(ctx, map[string]interface{}{"space_template": res, "err": err}, "failed to create or update work item link types")
		return nil, errs.Wrapf(err, "failed to create or update work item link types")
	}

	// Create or update work item type groups
	if err := r.createOrUpdateWITGs(ctx, res); err != nil {
		log.Error(ctx, map[string]interface{}{"space_template": res, "err": err}, "failed to create or update work item type groups")
		return nil, errs.Wrapf(err, "failed to create or update work item type groups")
	}

	log.Info(ctx, map[string]interface{}{"space_template_id": s.Template.ID}, "space template imorted successfully")
	return res, nil
}

func (r *GormRepository) createOrUpdateWITs(ctx context.Context, s *ImportHelper) error {
	err := r.checkNoWITIsMissing(ctx, s)
	if err != nil {
		return errs.WithStack(err)
	}
	witRepo := workitem.NewWorkItemTypeRepository(r.db)
	for _, wit := range s.WITs {
		loadedWIT, err := witRepo.Load(ctx, wit.ID)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case errors.NotFoundError:
				// Create WIT
				_, err := witRepo.Create(ctx, s.Template.ID, &wit.ID, &wit.Extends, wit.Name, wit.Description, wit.Icon, wit.Fields)
				if err != nil {
					return errs.Wrapf(err, "failed to create work item type '%s' from space template '%s'", wit.Name, s.Template.ID)
				}
			default:
				log.Error(ctx, map[string]interface{}{"wit_id": wit.ID.String(), "err": err}, "failed to work item type")
				return errs.Wrapf(err, "failed to load work item type %s", wit.ID)
			}
		} else {
			if loadedWIT.SpaceTemplateID != s.Template.ID {
				return errs.Errorf("work item type %s exists and is bound to space template %s instead of the new one %s", loadedWIT.ID, loadedWIT.SpaceTemplateID, s.Template.ID)
			}

			// Update work item type
			loadedWIT.Name = wit.Name
			loadedWIT.Description = wit.Description
			loadedWIT.Icon = wit.Icon

			//--------------------------------------------------------------------------------
			// Double check all existing fields are still present in new fields with same type
			//--------------------------------------------------------------------------------
			// verify that FieldTypes are same as loadedWIT
			toBeFoundFields := map[string]workitem.FieldType{}
			for k, fd := range loadedWIT.Fields {
				toBeFoundFields[k] = fd.Type
			}
			// Remove fields directly defined in WIT
			for fieldName, fd := range wit.Fields {
				// verify FieldType with original value
				if originalType, ok := toBeFoundFields[fieldName]; ok {
					if fd.Type.Equal(originalType) == false {
						return errs.Errorf("type of the field %s changed from %s to %s", fieldName, originalType, fd.Type)
					}
				}
				delete(toBeFoundFields, fieldName)
			}
			// Remove fields defined by extended type
			var extendedType *workitem.WorkItemType
			if wit.Extends != uuid.Nil {
				extendedType, err = witRepo.Load(ctx, wit.Extends)
				if err != nil {
					return errs.Wrapf(err, "failed to load WIT to be extended: %s", wit.Extends)
				}
				for k := range extendedType.Fields {
					delete(toBeFoundFields, k)
				}
			}
			if len(toBeFoundFields) > 0 {
				return errs.Errorf("you must not remove these fields from the new work item type definition of \"%s\": %+v", wit.Name, toBeFoundFields)
			}

			// TODO(kwk): Check that fields have not changed types.

			// Update fields
			if extendedType != nil {
				loadedWIT.Fields = extendedType.Fields
			}
			for name, field := range wit.Fields {
				loadedWIT.Fields[name] = field
			}
			db := r.db.Save(&loadedWIT)
			if err := db.Error; err != nil {
				return errs.Wrapf(err, "failed to update work item type %s", wit.ID)
			}
			workitem.ClearGlobalWorkItemTypeCache()
		}
	}

	// Now that we have created all work item types we can wire them up to
	// create their child types.
	for _, wit := range s.WITs {
		// Delete old child work item types (if any) associated with this work
		// item. There's no need to retain information about old child types as
		// it is just a linkage of work item types.
		db := r.db.Unscoped().Delete(workitem.ChildType{}, "parent_work_item_type_id = ?", wit.ID)
		if db.Error != nil {
			return errors.NewInternalError(ctx, errs.Wrapf(db.Error, "failed to deleted previous work item child types for WIT '%s'", wit.Name))
		}
		err := witRepo.AddChildTypes(ctx, wit.ID, wit.ChildTypeIDs)
		if err != nil {
			return errs.Wrapf(err, `failed to add child types to work item type "%s" (%s)`, wit.Name, wit.ID)
		}
	}

	return nil
}

// checkNoWITIsMissing returns an error if currently imported work item types
// are missing already existing work item types.
func (r *GormRepository) checkNoWITIsMissing(ctx context.Context, s *ImportHelper) error {
	type idType struct {
		ID uuid.UUID `gorm:"column:id" sql:"type:uuid"`
	}
	var IDs []idType
	query := fmt.Sprintf(`SELECT id FROM "%s" WHERE space_template_id = ?`, workitem.WorkItemType{}.TableName())
	db := r.db.Raw(query, s.Template.ID.String()).Scan(&IDs)
	if db.Error != nil {
		return errs.Wrapf(db.Error, "failed to load all work item types for space template '%s'", s.Template.ID)
	}
	toBeFoundIDs := id.Map{}
	for _, i := range IDs {
		toBeFoundIDs[i.ID] = struct{}{}
	}
	for _, wit := range s.WITs {
		delete(toBeFoundIDs, wit.ID)
	}
	if len(toBeFoundIDs) > 0 {
		return errs.Errorf("work item types to be imported must not remove these existing work item types: %s", toBeFoundIDs)
	}
	return nil
}

func (r *GormRepository) createOrUpdateWILTs(ctx context.Context, s *ImportHelper) error {
	err := r.checkNoWILTIsMissing(ctx, s)
	if err != nil {
		return errs.WithStack(err)
	}
	wiltRepo := link.NewWorkItemLinkTypeRepository(r.db)
	for _, wilt := range s.WILTs {
		if wilt.LinkCategoryID == uuid.Nil {
			wilt.LinkCategoryID = link.SystemWorkItemLinkCategorySystemID
		}
		loadedWILT, err := wiltRepo.Load(ctx, wilt.ID)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case errors.NotFoundError:
				// Create WILT
				if uuid.Equal(wilt.ID, uuid.Nil) {
					wilt.ID = uuid.NewV4()
				}
				_, err := wiltRepo.Create(ctx, &link.WorkItemLinkType{
					ID:              wilt.ID,
					Name:            wilt.Name,
					Description:     wilt.Description,
					ForwardName:     wilt.ForwardName,
					ReverseName:     wilt.ReverseName,
					Topology:        wilt.Topology,
					LinkCategoryID:  link.SystemWorkItemLinkCategoryUserID,
					SpaceTemplateID: s.Template.ID,
				})
				if err != nil {
					return errs.Wrapf(err, "failed to create work item link type '%s' from space template '%s'", wilt.Name, s.Template.ID)
				}
			default:
				return errs.Wrapf(err, "failed to load work item link type %s", wilt.ID)
			}
		} else {
			if loadedWILT.SpaceTemplateID != s.Template.ID {
				return errs.Errorf("work item link type %s exists and is bound to space template %s instead of the new one %s", loadedWILT.ID, loadedWILT.SpaceTemplateID, s.Template.ID)
			}
			loadedWILT.Name = wilt.Name
			loadedWILT.Description = wilt.Description
			loadedWILT.ForwardName = wilt.ForwardName
			loadedWILT.ReverseName = wilt.ReverseName
			db := r.db.Save(&loadedWILT)
			if err := db.Error; err != nil {
				return errs.Wrapf(err, "failed to update work item link type %s", wilt.ID)
			}
		}
	}
	return nil
}

// checkNoWILTIsMissing returns an error if currently imported work item link
// types are missing already existing work item link types.
func (r *GormRepository) checkNoWILTIsMissing(ctx context.Context, s *ImportHelper) error {
	type idType struct {
		ID uuid.UUID `gorm:"column:id" sql:"type:uuid"`
	}
	var IDs []idType
	query := fmt.Sprintf(`SELECT id FROM "%s" WHERE space_template_id = ?`, link.WorkItemLinkType{}.TableName())
	db := r.db.Raw(query, s.Template.ID.String()).Scan(&IDs)
	if db.Error != nil {
		return errs.Wrapf(db.Error, "failed to load all work item link types for space template '%s'", s.Template.ID)
	}
	toBeFoundIDs := id.Map{}
	for _, i := range IDs {
		toBeFoundIDs[i.ID] = struct{}{}
	}
	for _, wilt := range s.WILTs {
		delete(toBeFoundIDs, wilt.ID)
	}
	if len(toBeFoundIDs) > 0 {
		return errs.Errorf("work item link types to be imported must not remove these existing work item link types: %s", toBeFoundIDs)
	}
	return nil
}

func (r *GormRepository) createOrUpdateWITGs(ctx context.Context, s *ImportHelper) error {
	// Delete old work item type groups (if any) associated with this space
	// template. There's no need to retain information about old type groups as
	// it is just a linkage of work item types.
	db := r.db.Unscoped().Delete(workitem.WorkItemTypeGroup{}, "space_template_id = ?", s.Template.ID)
	if db.Error != nil {
		return errors.NewInternalError(ctx, errs.Wrapf(db.Error, "failed to deleted previous work item type groups for space template '%s'", s.Template.ID))
	}
	repo := workitem.NewWorkItemTypeGroupRepository(r.db)
	for pos, group := range s.WITGs {
		group.Position = pos
		_, err := repo.Create(ctx, *group)
		if err != nil {
			return errs.Wrapf(err, "failed to create work item type group '%s' from space template '%s'", group.Name, s.Template.ID)
		}
	}
	return nil
}
