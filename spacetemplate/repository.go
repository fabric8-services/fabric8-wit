package spacetemplate

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Repository describes interactions with space templates
type Repository interface {
	repository.Exister
	// Create creates a new space template and all the artifacts (e.g. work item
	// types, work item link types) in the system_
	Create(ctx context.Context, template SpaceTemplate) (*SpaceTemplate, error)
	// List returns an array with all space templates in it
	List(ctx context.Context) ([]SpaceTemplate, error)
	// Load returns a single space template by a given ID
	Load(ctx context.Context, templateID uuid.UUID) (*SpaceTemplate, error)
}

// NewRepository creates a new space template repository
func NewRepository(db *gorm.DB) Repository {
	return &GormRepository{db: db}
}

// GormRepository is the implementation of the repository interface for space
// templates.
type GormRepository struct {
	db *gorm.DB
}

// CheckExists returns nil if a spacetemplate exists with a given ID
func (r *GormRepository) CheckExists(ctx context.Context, id uuid.UUID) error {
	return repository.CheckExists(ctx, r.db, SpaceTemplate{}.TableName(), id)
}

// List returns an array with all space templates in it
func (r *GormRepository) List(ctx context.Context) ([]SpaceTemplate, error) {
	var objs []SpaceTemplate
	err := r.db.Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errs.Wrap(err, "failed to list space templates")
	}
	return objs, nil
}

// Load returns a single space template by a given ID
func (r *GormRepository) Load(ctx context.Context, spaceTemplateID uuid.UUID) (*SpaceTemplate, error) {
	var s SpaceTemplate
	tx := r.db.Where("id = ?", spaceTemplateID).First(&s)
	if tx.RecordNotFound() {
		log.Info(ctx, map[string]interface{}{
			"space_template_id": spaceTemplateID.String(),
		}, "space template not found")
		return nil, errors.NewNotFoundError("space_template", spaceTemplateID.String())
	}
	if tx.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"space_template_id": spaceTemplateID.String(),
			"err":               tx.Error,
		}, "failed to load space template")
		return nil, errors.NewInternalError(ctx, errs.Wrap(tx.Error, "failed to load space template"))
	}
	return &s, nil
}

// Create creates a new space template and all the artifacts (e.g. work item
// types, work item link types) in the system_
func (r *GormRepository) Create(ctx context.Context, s SpaceTemplate) (*SpaceTemplate, error) {
	if uuid.Equal(s.ID, uuid.Nil) {
		s.ID = uuid.NewV4()
	}

	if err := s.Validate(); err != nil {
		log.Error(ctx, map[string]interface{}{"space_template": s, "err": err}, "space template is invalid")
		return nil, errs.Wrap(err, "space template is invalid")
	}

	// Create space template
	db := r.db.Create(&s)
	if err := db.Error; err != nil {
		log.Error(ctx, map[string]interface{}{"space_template": s, "err": err}, "failed to create space template")
		// name needs to be unique
		if gormsupport.IsUniqueViolation(err, "space_templates_name_uidx") {
			return nil, errors.NewBadParameterError("name", s.Name).Expected("unique")
		}
		return nil, errs.Wrap(err, "failed to create space template")
	}
	log.Debug(ctx, map[string]interface{}{"space_template_id": s.ID}, "space template created successfully")
	return &s, nil
}
