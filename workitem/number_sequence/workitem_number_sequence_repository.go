package numbersequence

import (
	"context"
	"fmt"

	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// WorkItemNumberSequenceRepository the interface for the work item number sequence repository
type WorkItemNumberSequenceRepository interface {
	NextVal(ctx context.Context, spaceID uuid.UUID) (*int, error)
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

// NextVal returns the next work item sequence number for the given space ID. Creates an entry in the DB if none was found before
func (r *GormWorkItemNumberSequenceRepository) NextVal(ctx context.Context, spaceID uuid.UUID) (*int, error) {
	// upsert the next val, retrieves full row
	upsertStmt := fmt.Sprintf(`INSERT INTO %[1]s (space_id, current_val) VALUES ($1,1)
		ON CONFLICT (space_id) DO UPDATE SET current_val = %[1]s.current_val + EXCLUDED.current_val
		RETURNING current_val`, WorkItemNumberSequence{}.TableName())
	var currentVal int
	err := r.db.CommonDB().QueryRow(upsertStmt, spaceID).Scan(&currentVal)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to obtain next val for space with ID=`%s`", spaceID.String())
	}
	log.Debug(nil, map[string]interface{}{"space_id": spaceID, "next_val": currentVal}, "computed nextVal")
	return &currentVal, nil
}
