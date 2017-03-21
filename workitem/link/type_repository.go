package link

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/space"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// WorkItemLinkTypeRepository encapsulates storage & retrieval of work item link types
type WorkItemLinkTypeRepository interface {
	Create(ctx context.Context, name string, description *string, sourceTypeID, targetTypeID uuid.UUID, forwardName, reverseName, topology string, linkCategory, spaceID uuid.UUID) (*app.WorkItemLinkTypeSingle, error)
	Load(ctx context.Context, ID uuid.UUID) (*app.WorkItemLinkTypeSingle, error)
	List(ctx context.Context) (*app.WorkItemLinkTypeList, error)
	Delete(ctx context.Context, ID uuid.UUID) error
	Save(ctx context.Context, linkCat app.WorkItemLinkTypeSingle) (*app.WorkItemLinkTypeSingle, error)
	// ListSourceLinkTypes returns the possible link types for where the given
	// WIT can be used in the source.
	ListSourceLinkTypes(ctx context.Context, witID uuid.UUID) (*app.WorkItemLinkTypeList, error)
	// ListSourceLinkTypes returns the possible link types for where the given
	// WIT can be used in the target.
	ListTargetLinkTypes(ctx context.Context, witID uuid.UUID) (*app.WorkItemLinkTypeList, error)
}

// NewWorkItemLinkTypeRepository creates a work item link type repository based on gorm
func NewWorkItemLinkTypeRepository(db *gorm.DB) *GormWorkItemLinkTypeRepository {
	return &GormWorkItemLinkTypeRepository{db}
}

// GormWorkItemLinkTypeRepository implements WorkItemLinkTypeRepository using gorm
type GormWorkItemLinkTypeRepository struct {
	db *gorm.DB
}

// Create creates a new work item link type in the repository.
// Returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemLinkTypeRepository) Create(ctx context.Context, name string, description *string, sourceTypeID, targetTypeID uuid.UUID, forwardName, reverseName, topology string, linkCategoryID, spaceID uuid.UUID) (*app.WorkItemLinkTypeSingle, error) {
	linkType := &WorkItemLinkType{
		Name:           name,
		Description:    description,
		SourceTypeID:   sourceTypeID,
		TargetTypeID:   targetTypeID,
		ForwardName:    forwardName,
		ReverseName:    reverseName,
		Topology:       topology,
		LinkCategoryID: linkCategoryID,
		SpaceID:        spaceID,
	}
	if err := linkType.CheckValidForCreation(); err != nil {
		return nil, errs.WithStack(err)
	}

	// Check link category exists
	linkCategory := WorkItemLinkCategory{}
	db := r.db.Where("id=?", linkType.LinkCategoryID).Find(&linkCategory)
	if db.RecordNotFound() {
		return nil, errors.NewBadParameterError("work item link category", linkType.LinkCategoryID)
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("Failed to find work item link category: %s", db.Error.Error()))
	}
	// Check space exists
	space := space.Space{}
	db = r.db.Where("id=?", linkType.SpaceID).Find(&space)
	if db.RecordNotFound() {
		return nil, errors.NewBadParameterError("work item link space", linkType.SpaceID)
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("Failed to find work item link space: %s", db.Error.Error()))
	}

	db = r.db.Create(linkType)
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkTypeFromModel(goa.ContextRequest(ctx), *linkType)
	return &result, nil
}

// Load returns the work item link type for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkTypeRepository) Load(ctx context.Context, ID uuid.UUID) (*app.WorkItemLinkTypeSingle, error) {
	log.Info(ctx, map[string]interface{}{
		"wiltID": ID,
	}, "Loading work item link type")
	res := WorkItemLinkType{}
	db := r.db.Model(&res).Where("id=?", ID).First(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wiltID": ID,
		}, "work item link type not found")
		return nil, errors.NewNotFoundError("work item link type", ID.String())
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkTypeFromModel(goa.ContextRequest(ctx), res)

	return &result, nil
}

// LoadTypeFromDB return work item link type for the given name in the correct link category
// NOTE: Two link types can coexist with different categoryIDs.
func (r *GormWorkItemLinkTypeRepository) LoadTypeFromDBByNameAndCategory(ctx context.Context, name string, categoryId uuid.UUID) (*WorkItemLinkType, error) {
	log.Info(ctx, map[string]interface{}{
		"wiltName":   name,
		"categoryId": categoryId,
	}, "Loading work item link type %s with category ID %s", name, categoryId.String())

	res := WorkItemLinkType{}
	db := r.db.Model(&res).Where("name=? AND link_category_id=?", name, categoryId.String()).First(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wiltName":   name,
			"categoryId": categoryId.String(),
		}, "work item link type not found")
		return nil, errors.NewNotFoundError("work item link type", name)
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	return &res, nil
}

// LoadTypeFromDB return work item link type for the given ID
func (r *GormWorkItemLinkTypeRepository) LoadTypeFromDBByID(ctx context.Context, ID uuid.UUID) (*WorkItemLinkType, error) {
	log.Info(ctx, map[string]interface{}{
		"wiltID": ID.String(),
	}, "Loading work item link type with ID ", ID)

	res := WorkItemLinkType{}
	db := r.db.Model(&res).Where("ID=?", ID.String()).First(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wiltID": ID.String(),
		}, "work item link type not found")
		return nil, errors.NewNotFoundError("work item link type", ID.String())
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	return &res, nil
}

// List returns all work item link types
// TODO: Handle pagination
func (r *GormWorkItemLinkTypeRepository) List(ctx context.Context) (*app.WorkItemLinkTypeList, error) {
	// We don't have any where clause or paging at the moment.
	var rows []WorkItemLinkType
	db := r.db.Find(&rows)
	if db.Error != nil {
		return nil, db.Error
	}
	res := app.WorkItemLinkTypeList{}
	res.Data = make([]*app.WorkItemLinkTypeData, len(rows))
	for index, value := range rows {
		linkType := ConvertLinkTypeFromModel(goa.ContextRequest(ctx), value)
		res.Data[index] = linkType.Data
	}
	// TODO: When adding pagination, this must not be len(rows) but
	// the overall total number of elements from all pages.
	res.Meta = &app.WorkItemLinkTypeListMeta{
		TotalCount: len(rows),
	}
	return &res, nil
}

// Delete deletes the work item link type with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemLinkTypeRepository) Delete(ctx context.Context, ID uuid.UUID) error {
	var cat = WorkItemLinkType{
		ID: ID,
	}
	log.Info(ctx, map[string]interface{}{
		"wiltID": ID,
	}, "Work item link type to delete %v", cat)

	db := r.db.Delete(&cat)
	if db.Error != nil {
		return errors.NewInternalError(db.Error.Error())
	}
	if db.RowsAffected == 0 {
		return errors.NewNotFoundError("work item link type", ID.String())
	}
	return nil
}

// Save updates the given work item link type in storage. Version must be the same as the one int the stored version.
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemLinkTypeRepository) Save(ctx context.Context, lt app.WorkItemLinkTypeSingle) (*app.WorkItemLinkTypeSingle, error) {
	res := WorkItemLinkType{}
	if lt.Data.ID == nil {
		return nil, errors.NewBadParameterError("work item link type", nil)
	}
	db := r.db.Model(&res).Where("id=?", *lt.Data.ID).First(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wiltID": *lt.Data.ID,
		}, "work item link type not found")
		return nil, errors.NewNotFoundError("work item link type", lt.Data.ID.String())
	}
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wiltID": *lt.Data.ID,
			"err":    db.Error,
		}, "unable to find work item link type repository")
		return nil, errors.NewInternalError(db.Error.Error())
	}
	if lt.Data.Attributes.Version == nil || res.Version != *lt.Data.Attributes.Version {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	if err := ConvertLinkTypeToModel(lt, &res); err != nil {
		return nil, errs.WithStack(err)
	}
	res.Version = res.Version + 1
	db = db.Save(&res)
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wiltID": res.ID,
			"wilt":   res,
			"err":    db.Error,
		}, "unable to save work item link type repository")
		return nil, errors.NewInternalError(db.Error.Error())
	}
	log.Info(ctx, map[string]interface{}{
		"wiltID": res.ID,
		"wilt":   res,
	}, "Work item link type updated %v", res)
	result := ConvertLinkTypeFromModel(goa.ContextRequest(ctx), res)
	return &result, nil
}

type fetchLinkTypesFunc func() ([]WorkItemLinkType, error)

func (r *GormWorkItemLinkTypeRepository) listLinkTypes(ctx context.Context, fetchFunc fetchLinkTypesFunc) (*app.WorkItemLinkTypeList, error) {
	rows, err := fetchFunc()
	if err != nil {
		return nil, errs.WithStack(err)
	}
	res := app.WorkItemLinkTypeList{}
	res.Data = make([]*app.WorkItemLinkTypeData, len(rows))
	for index, value := range rows {
		lt := ConvertLinkTypeFromModel(goa.ContextRequest(ctx), value)
		res.Data[index] = lt.Data
	}
	// TODO: When adding pagination, this must not be len(rows) but
	// the overall total number of elements from all pages.
	res.Meta = &app.WorkItemLinkTypeListMeta{
		TotalCount: len(rows),
	}
	return &res, nil
}

func (r *GormWorkItemLinkTypeRepository) ListSourceLinkTypes(ctx context.Context, witID uuid.UUID) (*app.WorkItemLinkTypeList, error) {
	return r.listLinkTypes(ctx, func() ([]WorkItemLinkType, error) {
		db := r.db.Model(WorkItemLinkType{})
		query := fmt.Sprintf(`
			-- Get link types we can use with a specific WIT if the WIT is at the
			-- source of the link.
			(SELECT path FROM %[2]s WHERE id = %[1]s.source_type_id LIMIT 1)
			@>
			(SELECT path FROM %[2]s WHERE id = ? LIMIT 1)`,
			WorkItemLinkType{}.TableName(),
			workitem.WorkItemType{}.TableName(),
		)
		db = db.Where(query, witID)
		var rows []WorkItemLinkType
		db = db.Find(&rows)
		if db.RecordNotFound() {
			return nil, nil
		}
		if db.Error != nil {
			return nil, errs.WithStack(db.Error)
		}
		return rows, nil
	})
}

func (r *GormWorkItemLinkTypeRepository) ListTargetLinkTypes(ctx context.Context, witID uuid.UUID) (*app.WorkItemLinkTypeList, error) {
	return r.listLinkTypes(ctx, func() ([]WorkItemLinkType, error) {
		db := r.db.Model(WorkItemLinkType{})
		query := fmt.Sprintf(`
			-- Get link types we can use with a specific WIT if the WIT is at the
			-- target of the link.
			(SELECT path FROM %[2]s WHERE id = %[1]s.target_type_id LIMIT 1)
			@>
			(SELECT path FROM %[2]s WHERE id = ? LIMIT 1)`,
			WorkItemLinkType{}.TableName(),
			workitem.WorkItemType{}.TableName(),
		)
		db = db.Where(query, witID)
		var rows []WorkItemLinkType
		db = db.Find(&rows)
		if db.RecordNotFound() {
			return nil, nil
		}
		if db.Error != nil {
			return nil, errs.WithStack(db.Error)
		}
		return rows, nil
	})
}
