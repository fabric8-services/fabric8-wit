package space

import (
	"strings"

	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/log"

	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	satoriuuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

var SystemSpace = satoriuuid.FromStringOrNil("2e0698d8-753e-4cef-bb7c-f027634824a2")

// Space represents a Space on the domain and db layer
type Space struct {
	gormsupport.Lifecycle
	ID          satoriuuid.UUID
	Version     int
	Name        string
	Description string
	OwnerId     satoriuuid.UUID `sql:"type:uuid"` // Belongs To Identity
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = Space{}
var _ convert.Equaler = (*Space)(nil)

// Equal returns true if two Space objects are equal; otherwise false is returned.
func (p Space) Equal(u convert.Equaler) bool {
	other, ok := u.(Space)
	if !ok {
		return false
	}
	lfEqual := p.Lifecycle.Equal(other.Lifecycle)
	if !lfEqual {
		return false
	}
	if p.Version != other.Version {
		return false
	}
	if p.Name != other.Name {
		return false
	}
	if p.Description != other.Description {
		return false
	}
	if !satoriuuid.Equal(p.OwnerId, other.OwnerId) {
		return false
	}
	return true
}

// Repository encapsulate storage & retrieval of spaces
type Repository interface {
	Create(ctx context.Context, space *Space) (*Space, error)
	Save(ctx context.Context, space *Space) (*Space, error)
	Load(ctx context.Context, ID satoriuuid.UUID) (*Space, error)
	Delete(ctx context.Context, ID satoriuuid.UUID) error
	LoadByOwnerAndName(ctx context.Context, userId *satoriuuid.UUID, spaceName *string) (*Space, error)
	List(ctx context.Context, start *int, length *int) ([]*Space, uint64, error)
	Search(ctx context.Context, q *string, start *int, length *int) ([]*Space, uint64, error)
}

// NewRepository creates a new space repo
func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db}
}

// GormRepository implements SpaceRepository using gorm
type GormRepository struct {
	db *gorm.DB
}

// Load returns the space for the given id
// returns NotFoundError or InternalError
func (r *GormRepository) Load(ctx context.Context, ID satoriuuid.UUID) (*Space, error) {
	res := Space{}
	tx := r.db.Where("id=?", ID).First(&res)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"spaceID": ID.String(),
		}, "state or known referer was empty")
		return nil, errors.NewNotFoundError("space", ID.String())
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(tx.Error.Error())
	}
	return &res, nil
}

// Delete deletes the space with the given id
// returns NotFoundError or InternalError
func (r *GormRepository) Delete(ctx context.Context, ID satoriuuid.UUID) error {
	if ID == satoriuuid.Nil {
		log.Error(ctx, map[string]interface{}{
			"spaceID": ID.String(),
		}, "unable to find the space by ID")
		return errors.NewNotFoundError("space", ID.String())
	}
	space := Space{ID: ID}
	tx := r.db.Delete(space)

	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"spaceID": ID.String(),
		}, "unable to delete the space")
		return errors.NewInternalError(err.Error())
	}
	if tx.RowsAffected == 0 {
		log.Error(ctx, map[string]interface{}{
			"spaceID": ID.String(),
		}, "none row was affected by the deletion operation")
		return errors.NewNotFoundError("space", ID.String())
	}

	return nil
}

// Save updates the given space in the db. Version must be the same as the one in the stored version
// returns NotFoundError, BadParameterError, VersionConflictError or InternalError
func (r *GormRepository) Save(ctx context.Context, p *Space) (*Space, error) {
	pr := Space{}
	tx := r.db.Where("id=?", p.ID).First(&pr)
	oldVersion := p.Version
	p.Version++
	if tx.RecordNotFound() {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, errors.NewNotFoundError("space", p.ID.String())
	}
	if err := tx.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	tx = tx.Where("Version = ?", oldVersion).Save(p)
	if err := tx.Error; err != nil {
		if gormsupport.IsCheckViolation(tx.Error, "spaces_name_check") {
			return nil, errors.NewBadParameterError("Name", p.Name).Expected("not empty")
		}
		if gormsupport.IsUniqueViolation(tx.Error, "spaces_name_idx") {
			return nil, errors.NewBadParameterError("Name", p.Name).Expected("unique")
		}
		return nil, errors.NewInternalError(err.Error())
	}
	if tx.RowsAffected == 0 {
		return nil, errors.NewVersionConflictError("version conflict")
	}

	log.Info(ctx, map[string]interface{}{
		"spaceID": p.ID,
	}, "space updated successfully")
	return p, nil
}

// Create creates a new Space in the db
// returns BadParameterError or InternalError
func (r *GormRepository) Create(ctx context.Context, space *Space) (*Space, error) {
	// We might want to create a space with a specific ID, e.g. space.SystemSpace
	if space.ID == satoriuuid.Nil {
		space.ID = satoriuuid.NewV4()
	}

	tx := r.db.Create(space)
	if err := tx.Error; err != nil {
		if gormsupport.IsCheckViolation(tx.Error, "spaces_name_check") {
			return nil, errors.NewBadParameterError("Name", space.Name).Expected("not empty")
		}
		if gormsupport.IsUniqueViolation(tx.Error, "spaces_name_idx") {
			return nil, errors.NewBadParameterError("Name", space.Name).Expected("unique")
		}
		return nil, errors.NewInternalError(err.Error())
	}

	log.Info(ctx, map[string]interface{}{
		"spaceID": space.ID,
	}, "Space created successfully")
	return space, nil
}

// extracted this function from List() in order to close the rows object with "defer" for more readability
// workaround for https://github.com/lib/pq/issues/81
func (r *GormRepository) listSpaceFromDB(ctx context.Context, q *string, start *int, limit *int) ([]*Space, uint64, error) {

	db := r.db.Model(&Space{})
	orgDB := db
	if start != nil {
		if *start < 0 {
			return nil, 0, errors.NewBadParameterError("start", *start)
		}
		db = db.Offset(*start)
	}
	if limit != nil {
		if *limit <= 0 {
			return nil, 0, errors.NewBadParameterError("limit", *limit)
		}
		db = db.Limit(*limit)
	}
	db = db.Select("count(*) over () as cnt2 , *")
	if q != nil {
		db = db.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(*q)+"%")
		db = db.Or("LOWER(description) LIKE ?", "%"+strings.ToLower(*q)+"%")
	}

	rows, err := db.Rows()
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}
	defer rows.Close()

	result := []*Space{}
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, errors.NewInternalError(err.Error())
	}

	// need to set up a result for Scan() in order to extract total count.
	var count uint64
	var ignore interface{}
	columnValues := make([]interface{}, len(columns))

	for index := range columnValues {
		columnValues[index] = &ignore
	}
	columnValues[0] = &count
	first := true

	for rows.Next() {
		value := Space{}
		db.ScanRows(rows, &value)
		if first {
			first = false
			if err = rows.Scan(columnValues...); err != nil {
				return nil, 0, errors.NewInternalError(err.Error())
			}
		}
		result = append(result, &value)
	}
	if first {
		if q != nil {
			// If 0 rows were returned from first query during search, then total is 0
			count = 0
		} else {
			// means 0 rows were returned from the first query (maybe becaus of offset outside of total count),
			// need to do a count(*) to find out total
			orgDB := orgDB.Select("count(*)")
			rows2, err := orgDB.Rows()
			defer rows2.Close()
			if err != nil {
				return nil, 0, errs.WithStack(err)
			}
			rows2.Next() // count(*) will always return a row
			rows2.Scan(&count)
		}
	}
	return result, count, nil
}

// List returns work item selected by the given criteria.Expression, starting with start (zero-based) and returning at most limit items
func (r *GormRepository) List(ctx context.Context, start *int, limit *int) ([]*Space, uint64, error) {
	result, count, err := r.listSpaceFromDB(ctx, nil, start, limit)
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}

	return result, count, nil
}

func (r *GormRepository) Search(ctx context.Context, q *string, start *int, limit *int) ([]*Space, uint64, error) {
	result, count, err := r.listSpaceFromDB(ctx, q, start, limit)
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}

	return result, count, nil
}

func (r *GormRepository) LoadByOwnerAndName(ctx context.Context, userId *satoriuuid.UUID, spaceName *string) (*Space, error) {
	res := Space{}
	tx := r.db.Where("spaces.owner_id=? AND spaces.name=?", *userId, *spaceName).First(&res)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"spaceName": *spaceName,
			"userId":    *userId,
		}, "Could not find space under owner")
		return nil, errors.NewNotFoundError("space", *spaceName)
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(tx.Error.Error())
	}
	return &res, nil
}
