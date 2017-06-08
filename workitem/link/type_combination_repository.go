package link

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
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
	linkTypeExists, err := NewWorkItemLinkTypeRepository(r.db).Exists(ctx, tc.LinkTypeID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"wilt_id": tc.LinkTypeID,
			"err":     err,
		}, "error checking for work item link type")
		return nil, errors.NewInternalError(fmt.Sprintf("failed to find link type: %s", err.Error()))
	}
	if !linkTypeExists {
		log.Error(ctx, map[string]interface{}{
			"wilt_id": tc.LinkTypeID,
		}, "work item link type not found")
		return nil, errors.NewNotFoundError("wilt_id", tc.LinkTypeID.String())
	}
	// Check source WIT exists
	// TODO(kwk): Implement WIT repo method "Exists(ctx, id)" and use that here
	sourceType := workitem.WorkItemType{}
	db := r.db.Where("id=?", tc.SourceTypeID).Find(&sourceType)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wit_id": tc.SourceTypeID,
		}, "source work item type not found")
		return nil, errors.NewNotFoundError("source_wit_id", tc.SourceTypeID.String())
	}
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wit_id": tc.SourceTypeID,
			"err":    db.Error,
		}, "failed to find source work item type")
		return nil, errors.NewInternalError(fmt.Sprintf("failed to find source work item type: %s", db.Error.Error()))
	}
	// Check target WIT exists
	// TODO(kwk): Implement WIT repo method "Exists(ctx, id)" and use that here
	targetType := workitem.WorkItemType{}
	db = r.db.Where("id=?", tc.TargetTypeID).Find(&targetType)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wit_id": tc.TargetTypeID,
		}, "target work item type not found")
		return nil, errors.NewNotFoundError("target_wit_id", tc.TargetTypeID.String())
	}
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wit_id": tc.TargetTypeID,
			"err":    db.Error,
		}, "failed to find target work item type")
		return nil, errors.NewInternalError(fmt.Sprintf("failed to find target work item type: %s", db.Error.Error()))
	}
	// Finally create the type combination record in the DB
	db = r.db.Create(tc)
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"err": db.Error,
		}, "failed to create work item link type combination")
	}
	if gormsupport.IsUniqueViolation(db.Error, "work_item_link_type_combinations_uniq") {
		return nil, errors.NewBadParameterError("space+link+source+target", tc).Expected("unique")
	}
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

	// Check if link type even exists
	wiltExists, err := NewWorkItemLinkTypeRepository(r.db).Exists(ctx, linkTypeID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"wilt_id": linkTypeID,
			"err":     err,
		}, "failed to check if work item link type exists")
		return nil, errs.WithStack(err)
	}
	if !wiltExists {
		log.Error(ctx, map[string]interface{}{
			"wilt_id": linkTypeID,
		}, "work item link type not found")
		return nil, errors.NewNotFoundError("wilt_id", linkTypeID.String())
	}

	// We don't have any where clause or paging at the moment.
	var modelWorkItemLinkTypeCombinations []WorkItemLinkTypeCombination
	db := r.db.Where("link_type_id=?", linkTypeID)
	if err := db.Find(&modelWorkItemLinkTypeCombinations).Error; err != nil {
		return nil, errs.WithStack(err)
	}
	return modelWorkItemLinkTypeCombinations, nil
}
