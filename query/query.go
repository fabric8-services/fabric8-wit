package query

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// APIStringTypeQuery helps to avoid string literal
const APIStringTypeQuery = "queries"

// Query describes a single Label
type Query struct {
	gormsupport.Lifecycle
	ID      uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	SpaceID uuid.UUID `sql:"type:uuid"`
	Creator uuid.UUID `sql:"type:uuid"`
	Title   string
	Fields  string
}

// GetETagData returns the field values to use to generate the ETag
// func (q Query) GetETagData() []interface{} {
// 	return []interface{}{q.ID, q.Version}
// }

// GetLastModified returns the last modification time
func (q Query) GetLastModified() time.Time {
	return q.UpdatedAt.Truncate(time.Second)
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (q Query) TableName() string {
	return "queries"
}

// Repository describes interactions with Labels
type Repository interface {
	Create(ctx context.Context, u *Query) error
	List(ctx context.Context, spaceID uuid.UUID) ([]Query, error)
	ListByCreator(ctx context.Context, spaceID uuid.UUID, creatorID uuid.UUID) ([]Query, error)
	Load(ctx context.Context, queryID uuid.UUID, spaceID uuid.UUID) (*Query, error)
	Delete(ctx context.Context, ID uuid.UUID) error
}

// NewQueryRepository creates a new storage type.
func NewQueryRepository(db *gorm.DB) Repository {
	return &GormQueryRepository{db: db}
}

// GormQueryRepository is the implementation of the storage interface for Labels.
type GormQueryRepository struct {
	db *gorm.DB
}

// GetETagData returns the field values to use to generate the ETag
func (q Query) GetETagData() []interface{} {
	return []interface{}{q.ID, strconv.FormatInt(q.UpdatedAt.Unix(), 10)}
}

// QueryTableName constant that holds table name of Labels
const QueryTableName = "queries"

// Create a new label
func (r *GormQueryRepository) Create(ctx context.Context, q *Query) error {
	defer goa.MeasureSince([]string{"goa", "db", "Query", "create"}, time.Now())
	q.ID = uuid.NewV4()
	if strings.TrimSpace(q.Title) == "" {
		return errors.NewBadParameterError("query name cannot be empty string", q.Title).Expected("non empty string")
	}
	if q.Creator == uuid.Nil {
		return errors.NewBadParameterError("creator cannot be nil", q.Creator).Expected("valid user ID")
	}
	err := r.db.Create(q).Error
	if err != nil {
		// combination of name and space ID should be unique
		if gormsupport.IsUniqueViolation(err, "queries_title_space_id_creator_unique") {
			log.Error(ctx, map[string]interface{}{
				"err":      err,
				"title":    q.Title,
				"space_id": q.SpaceID,
			}, "unable to create query because a query with same title already exists in the space by same creator")
			return errors.NewDataConflictError(fmt.Sprintf("query already exists with title = %s , space_id = %s, creator = %s", q.Title, q.SpaceID, q.Creator))
		}
		log.Error(ctx, map[string]interface{}{}, "error adding Query: %s", err.Error())
		return err
	}
	return nil
}

// List all labels in a space
func (r *GormQueryRepository) List(ctx context.Context, spaceID uuid.UUID) ([]Query, error) {
	defer goa.MeasureSince([]string{"goa", "db", "Query", "list"}, time.Now())
	var objs []Query
	err := r.db.Where("space_id = ?", spaceID).Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return objs, nil
}

// ListByCreator all labels in a space by a creator
func (r *GormQueryRepository) ListByCreator(ctx context.Context, spaceID uuid.UUID, creatorID uuid.UUID) ([]Query, error) {
	defer goa.MeasureSince([]string{"goa", "db", "Query", "list"}, time.Now())
	var objs []Query
	err := r.db.Where("space_id = ? AND creator=?", spaceID, creatorID).Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return objs, nil
}

// Load Query in a space
func (r *GormQueryRepository) Load(ctx context.Context, ID uuid.UUID, spaceID uuid.UUID) (*Query, error) {
	defer goa.MeasureSince([]string{"goa", "db", "query", "show"}, time.Now())
	q := Query{}
	tx := r.db.Where("id = ? and space_id = ?", ID, spaceID).First(&q)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"query_id": ID.String(),
		}, "record not found")
		return nil, errors.NewNotFoundError("query", ID.String())
	}
	if tx.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"err":      tx.Error,
			"query_id": ID.String(),
		}, "unable to load the query by ID")
		return nil, errors.NewInternalError(ctx, tx.Error)
	}
	return &q, nil
}

// Delete deletes the query with the given id, returns NotFoundError or InternalError
func (r *GormQueryRepository) Delete(ctx context.Context, ID uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "query", "delete"}, time.Now())
	if ID == uuid.Nil {
		log.Error(ctx, map[string]interface{}{
			"query_id": ID.String(),
		}, "unable to find the query by ID")
		return errors.NewNotFoundError("query", ID.String())
	}
	q := Query{ID: ID}
	tx := r.db.Delete(q)

	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"query_id": ID.String(),
		}, "unable to delete the query")
		return errors.NewInternalError(ctx, err)
	}
	if tx.RowsAffected == 0 {
		log.Error(ctx, map[string]interface{}{
			"query_id": ID.String(),
		}, "none row was affected by the delete operation")
		return errors.NewNotFoundError("query", ID.String())
	}
	return nil
}
