package link

import (
	"context"
	"fmt"
	"time"

	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// WorkItemLinkTypeRepository encapsulates storage & retrieval of work item link types
type WorkItemLinkTypeRepository interface {
	repository.Exister
	Create(ctx context.Context, linkType WorkItemLinkType) (*WorkItemLinkType, error)
	Load(ctx context.Context, ID uuid.UUID) (*WorkItemLinkType, error)
	List(ctx context.Context, spaceTemplateID uuid.UUID) ([]WorkItemLinkType, error)
	Save(ctx context.Context, linkCat WorkItemLinkType) (*WorkItemLinkType, error)
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
func (r *GormWorkItemLinkTypeRepository) Create(ctx context.Context, linkType WorkItemLinkType) (*WorkItemLinkType, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlinktype", "create"}, time.Now())
	if err := linkType.CheckValidForCreation(); err != nil {
		log.Error(ctx, map[string]interface{}{
			"wilt_id": linkType.ID.String(),
			"err":     err,
		}, "failed to validate link type")
		return nil, errs.WithStack(err)
	}
	db := r.db.Create(&linkType)
	if db.Error != nil {
		if gormsupport.IsUniqueViolation(db.Error, "work_item_link_types_name_idx") {
			log.Error(ctx, map[string]interface{}{
				"err":               db.Error,
				"space_template_id": linkType.SpaceTemplateID,
				"wilt_name":         linkType.Name,
			}, "unable to create work item link type because a link already exists with the same space_template_id and name")
			return nil, errors.NewDataConflictError(fmt.Sprintf("work item link type already exists with the same space_template_id: %s; name: %s ", linkType.SpaceTemplateID, linkType.Name))
		}
		log.Error(ctx, map[string]interface{}{
			"wilt_id": linkType.ID.String(),
			"err":     db.Error,
		}, "failed to create work item link type")
		return nil, errors.NewInternalError(ctx, errs.Wrap(db.Error, "failed to create link type"))
	}
	log.Info(ctx, map[string]interface{}{
		"wilt_id": linkType.ID.String(),
	}, "created work item link type")
	return &linkType, nil
}

// Load returns the work item link type for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkTypeRepository) Load(ctx context.Context, ID uuid.UUID) (*WorkItemLinkType, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlinktype", "load"}, time.Now())
	log.Info(ctx, map[string]interface{}{
		"wilt_id": ID,
	}, "loading work item link type")
	modelLinkType := WorkItemLinkType{}
	db := r.db.Model(&modelLinkType).Where("id=?", ID).First(&modelLinkType)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wilt_id": ID,
		}, "work item link type not found")
		return nil, errors.NewNotFoundError("work item link type", ID.String())
	}
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wilt_id": ID,
			"err":     db.Error,
		}, "failed to create work item link type")
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	return &modelLinkType, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormWorkItemLinkTypeRepository) CheckExists(ctx context.Context, id uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlinktype", "exists"}, time.Now())
	return repository.CheckExists(ctx, r.db, WorkItemLinkType{}.TableName(), id)
}

// List returns all work item link types
func (r *GormWorkItemLinkTypeRepository) List(ctx context.Context, spaceTemplateID uuid.UUID) ([]WorkItemLinkType, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlinktype", "list"}, time.Now())
	log.Info(ctx, map[string]interface{}{
		"space_template_id": spaceTemplateID,
	}, "Listing work item link types by space template ID %s", spaceTemplateID)

	// check space template exists
	if err := spacetemplate.NewRepository(r.db).CheckExists(ctx, spaceTemplateID); err != nil {
		return nil, errors.NewNotFoundError("space template", spaceTemplateID.String())
	}

	// We don't have any where clause or paging at the moment.
	var modelLinkTypes []WorkItemLinkType
	db := r.db.Where("space_template_id IN (?, ?)", spaceTemplateID, spacetemplate.SystemBaseTemplateID).Order("name")
	if err := db.Find(&modelLinkTypes).Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":               err,
			"space_template_id": spaceTemplateID,
		}, "failed to list work item link types")
		return nil, errs.Wrapf(err, "failed to find link types")
	}
	return modelLinkTypes, nil
}

// Save updates the given work item link type in storage. Version must be the same as the one int the stored version.
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemLinkTypeRepository) Save(ctx context.Context, modelToSave WorkItemLinkType) (*WorkItemLinkType, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlinktype", "save"}, time.Now())
	existingModel := WorkItemLinkType{}
	db := r.db.Model(&existingModel).Where("id=?", modelToSave.ID).First(&existingModel)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wilt_id": modelToSave.ID,
		}, "work item link type not found")
		return nil, errors.NewNotFoundError("work item link type", modelToSave.ID.String())
	}
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wilt_id": modelToSave.ID,
			"err":     db.Error,
		}, "unable to find work item link type repository")
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	if err := modelToSave.CheckValidForCreation(); err != nil {
		log.Error(ctx, map[string]interface{}{
			"wilt_id": modelToSave.ID.String(),
			"err":     err,
		}, "failed to validate link type to save")
		return nil, errs.WithStack(err)
	}
	if existingModel.Version != modelToSave.Version {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	if existingModel.SpaceTemplateID != modelToSave.SpaceTemplateID {
		log.Error(ctx, map[string]interface{}{
			"wilt_id": modelToSave.ID,
		}, "you must not change the link types association to a space")
		return nil, errors.NewForbiddenError("you must not change the link types association to a space")
	}
	if err := modelToSave.Topology.CheckValid(); err != nil {
		log.Error(ctx, map[string]interface{}{
			"wilt_id": modelToSave.ID,
			"err":     err,
		}, "cannot update link type's topology to %s", modelToSave.Topology)
		return nil, errors.NewBadParameterError("topology", modelToSave.Topology)
	}
	modelToSave.Version = modelToSave.Version + 1
	if existingModel.SpaceTemplateID != modelToSave.SpaceTemplateID {
		return nil, errors.NewForbiddenError("one must not change the space template reference in a work item link")
	}
	db = db.Save(&modelToSave)
	if db.Error != nil {
		if gormsupport.IsUniqueViolation(db.Error, "work_item_link_types_name_idx") {
			log.Error(ctx, map[string]interface{}{
				"err":               db.Error,
				"space_template_id": existingModel.SpaceTemplateID,
				"wilt_name":         existingModel.Name,
				"wilt_id":           existingModel.ID,
			}, "unable to save work item link type because a link already exists with the same space_template_id and name")
			return nil, errors.NewDataConflictError(fmt.Sprintf("work item link type already exists within the same space: %s; name: %s", existingModel.SpaceTemplateID, existingModel.Name))
		}
		log.Error(ctx, map[string]interface{}{
			"wilt_id": existingModel.ID,
			"wilt":    existingModel,
			"err":     db.Error,
		}, "unable to save work item link type repository")
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	log.Info(ctx, map[string]interface{}{
		"wilt_id": existingModel.ID,
		"wilt":    existingModel,
	}, "Work item link type updated %v", modelToSave)
	return &modelToSave, nil
}
