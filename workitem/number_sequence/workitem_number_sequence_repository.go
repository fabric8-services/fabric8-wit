package numbersequence

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// WorkItemNumberSequenceRepository the interface for the work item number sequence repository
type WorkItemNumberSequenceRepository interface {
	Create(ctx context.Context, spaceID uuid.UUID) (*WorkItemNumberSequence, error)
	NextVal(ctx context.Context, spaceID uuid.UUID) (*WorkItemNumberSequence, error)
}

// NewWorkItemNumberSequenceRepository creates a GormWorkItemNumberSequenceRepository
func NewWorkItemNumberSequenceRepository(db *gorm.DB) *GormWorkItemNumberSequenceRepository {
	repository := &GormWorkItemNumberSequenceRepository{db}
	return repository
}

// GormWorkItemNumberSequenceRepository implements WorkItemNumberSequenceRepository using gorm
type GormWorkItemNumberSequenceRepository struct {
	db *gorm.DB
}

// Create returns the next work item sequence number for the given space ID. Creates an entry in the DB if none was found before
func (r *GormWorkItemNumberSequenceRepository) Create(ctx context.Context, spaceID uuid.UUID) (*WorkItemNumberSequence, error) {
	// retrieve the current issue number in the given space
	numberSequence := WorkItemNumberSequence{SpaceID: spaceID, CurrentVal: 0}
	if err := r.db.Save(&numberSequence).Error; err != nil {
		return nil, errs.Wrapf(err, "failed to create work item with sequence number: `%s`", numberSequence.String())
	}
	log.Warn(nil, map[string]interface{}{"Sequence": numberSequence.String()}, "Creating sequence")
	return &numberSequence, nil
}

// NextVal returns the next work item sequence number for the given space ID. Creates an entry in the DB if none was found before
func (r *GormWorkItemNumberSequenceRepository) NextVal(ctx context.Context, spaceID uuid.UUID) (*WorkItemNumberSequence, error) {
	// retrieve the current issue number in the given space
	numberSequence := WorkItemNumberSequence{}
	tx := r.db.Model(&WorkItemNumberSequence{}).Set("gorm:query_option", "FOR UPDATE").Where("space_id = ?", spaceID).First(&numberSequence)
	if tx.RecordNotFound() {
		numberSequence.SpaceID = spaceID
		numberSequence.CurrentVal = 1
	} else {
		numberSequence.CurrentVal++
	}
	if err := r.db.Save(&numberSequence).Error; err != nil {
		return nil, errs.Wrapf(err, "failed to update work item with sequence number: `%s`", numberSequence.String())
	}
	log.Warn(nil, map[string]interface{}{"Sequence": numberSequence.String()}, "computing nextVal")
	return &numberSequence, nil
}
