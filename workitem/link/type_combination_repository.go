package link

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/workitem"

	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// WorkItemLinkTypeCombinationRepository encapsulates storage & retrieval of work item link types
type WorkItemLinkTypeCombinationRepository interface {
	Create(ctx context.Context, tc *WorkItemLinkTypeCombination) (*WorkItemLinkTypeCombination, error)
	Load(ctx context.Context, ID uuid.UUID) (*WorkItemLinkTypeCombination, error)
	List(ctx context.Context, linkTypeID uuid.UUID) ([]WorkItemLinkTypeCombination, error)
	// Save(ctx context.Context, tc *WorkItemLinkTypeCombination) (*WorkItemLinkTypeCombination, error)
}

// NewWorkItemLinkTypeCombinationRepository creates a work item link WorkItemLinkTypeCombination repository based on gorm
func NewWorkItemLinkTypeCombinationRepository(db *gorm.DB) *GormWorkItemLinkTypeCombinationRepository {
	return &GormWorkItemLinkTypeCombinationRepository{db}
}

// GormWorkItemLinkTypeCombinationRepository implements WorkItemLinkTypeCombinationRepository using gorm
type GormWorkItemLinkTypeCombinationRepository struct {
	db *gorm.DB
}

// Create creates a new work item link WorkItemLinkTypeCombination in the repository.
// Returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemLinkTypeCombinationRepository) Create(ctx context.Context, tc *WorkItemLinkTypeCombination) (*WorkItemLinkTypeCombination, error) {
	if err := tc.CheckValidForCreation(); err != nil {
		return nil, errs.WithStack(err)
	}
	// Check link type exists
	// TODO(kwk): Implement WILT repo method "Exists(ctx, id)" and use that here
	linkType := WorkItemLinkType{}
	db := r.db.Where("id=?", tc.LinkTypeID).Find(&linkType)
	if db.RecordNotFound() {
		return nil, errors.NewBadParameterError("work item link type", tc.LinkTypeID)
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("failed to find work item link type: %s", db.Error.Error()))
	}
	// Check source WIT exists
	// TODO(kwk): Implement WIT repo method "Exists(ctx, id)" and use that here
	sourceType := workitem.WorkItemType{}
	db = r.db.Where("id=?", tc.SourceTypeID).Find(&sourceType)
	if db.RecordNotFound() {
		return nil, errors.NewBadParameterError("source work item type", tc.SourceTypeID)
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("failed to find source work item type: %s", db.Error.Error()))
	}
	// Check target WIT exists
	// TODO(kwk): Implement WIT repo method "Exists(ctx, id)" and use that here
	targetType := workitem.WorkItemType{}
	db = r.db.Where("id=?", tc.TargetTypeID).Find(&targetType)
	if db.RecordNotFound() {
		return nil, errors.NewBadParameterError("target work item type", tc.TargetTypeID)
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("failed to find target work item type: %s", db.Error.Error()))
	}
	// Finally create the type combination record in the DB
	db = r.db.Create(tc)
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	return tc, nil
}

// Load returns the work item link type combination for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkTypeCombinationRepository) Load(ctx context.Context, ID uuid.UUID) (*WorkItemLinkTypeCombination, error) {
	log.Info(ctx, map[string]interface{}{
		"type_combination_id": ID,
	}, "loading work item link type")
	modelLinkType := WorkItemLinkTypeCombination{}
	db := r.db.Model(&modelLinkType).Where("id=?", ID).First(&modelLinkType)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"type_combination_id": ID,
		}, "work item link type combination not found")
		return nil, errors.NewNotFoundError("work item link type combination", ID.String())
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	return &modelLinkType, nil
}

// List returns all work item link type combinations for the given link type
func (r *GormWorkItemLinkTypeCombinationRepository) List(ctx context.Context, linkTypeID uuid.UUID) ([]WorkItemLinkTypeCombination, error) {
	log.Info(ctx, map[string]interface{}{
		"link_type_id": linkTypeID,
	}, "listing work item link type combinations by link type ID %s", linkTypeID.String())

	// We don't have any where clause or paging at the moment.
	var modelWorkItemLinkTypeCombinations []WorkItemLinkTypeCombination
	db := r.db.Where("link_type_id=?", linkTypeID)
	if err := db.Find(&modelWorkItemLinkTypeCombinations).Error; err != nil {
		return nil, errs.WithStack(err)
	}
	return modelWorkItemLinkTypeCombinations, nil
}

// // Save updates the given work item link type combination in storage. Version must be the same as the one int the stored version.
// // returns NotFoundError, VersionConflictError, ConversionError or InternalError
// func (r *GormWorkItemLinkTypeCombinationRepository) Save(ctx context.Context, modelToSave WorkItemLinkTypeCombination) (*WorkItemLinkTypeCombination, error) {
// 	existingModel := WorkItemLinkTypeCombination{}
// 	db := r.db.Model(&existingModel).Where("id=?", modelToSave.ID).First(&existingModel)
// 	if db.RecordNotFound() {
// 		log.Error(ctx, map[string]interface{}{
// 			"type_combination_id": modelToSave.ID,
// 		}, "work item link type combination not found")
// 		return nil, errors.NewNotFoundError("work item link type combination", modelToSave.ID.String())
// 	}
// 	if db.Error != nil {
// 		log.Error(ctx, map[string]interface{}{
// 			"type_combination_id": modelToSave.ID,
// 			"err": db.Error,
// 		}, "unable to find work item link type combination")
// 		return nil, errors.NewInternalError(db.Error.Error())
// 	}
// 	if existingModel.Version != modelToSave.Version {
// 		return nil, errors.NewVersionConflictError("version conflict")
// 	}
// 	modelToSave.Version = modelToSave.Version + 1
// 	db = db.Save(&modelToSave)
// 	if db.Error != nil {
// 		log.Error(ctx, map[string]interface{}{
// 			"type_combination_id": existingModel.ID,
// 			"type_combination":    existingModel,
// 			"err":                 db.Error,
// 		}, "unable to save work item link type combination")
// 		return nil, errors.NewInternalError(db.Error.Error())
// 	}
// 	log.Info(ctx, map[string]interface{}{
// 		"type_combination_id": existingModel.ID,
// 		"type_combination":    existingModel,
// 	}, "Work item link type combination updated %v", modelToSave)
// 	return &modelToSave, nil
// }
