package comment

import (
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/rendering"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Comment describes a single comment
type Comment struct {
	gormsupport.Lifecycle
	ID        uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	ParentID  string
	CreatedBy uuid.UUID `sql:"type:uuid"` // Belongs To Identity
	Body      string
	Markup    string
}

// Repository describes interactions with comments
type Repository interface {
	Create(ctx context.Context, u *Comment) error
	Save(ctx context.Context, comment *Comment) (*Comment, error)
	List(ctx context.Context, parent string, start *int, limit *int) ([]*Comment, uint64, error)
	Load(ctx context.Context, id uuid.UUID) (*Comment, error)
	Count(ctx context.Context, parent string) (int, error)
}

// NewCommentRepository creates a new storage type.
func NewCommentRepository(db *gorm.DB) Repository {
	return &GormCommentRepository{db: db}
}

// GormCommentRepository is the implementation of the storage interface for Comments.
type GormCommentRepository struct {
	db *gorm.DB
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *GormCommentRepository) TableName() string {
	return "comments"
}

// Create creates a new record.
func (m *GormCommentRepository) Create(ctx context.Context, comment *Comment) error {
	defer goa.MeasureSince([]string{"goa", "db", "comment", "create"}, time.Now())
	comment.ID = uuid.NewV4()
	// make sure no comment is created with an empty 'markup' value
	if comment.Markup == "" {
		comment.Markup = rendering.SystemMarkupDefault
	}
	if err := m.db.Create(comment).Error; err != nil {
		goa.LogError(ctx, "error adding Comment", "error", err.Error())
		return errs.WithStack(err)
	}

	return nil
}

// Save a single comment
func (m *GormCommentRepository) Save(ctx context.Context, comment *Comment) (*Comment, error) {
	c := Comment{}
	tx := m.db.Where("id=?", comment.ID).First(&c)
	if tx.RecordNotFound() {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, errors.NewNotFoundError("comment", comment.ID.String())
	}
	if err := tx.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	// make sure no comment is created with an empty 'markup' value
	if comment.Markup == "" {
		comment.Markup = rendering.SystemMarkupDefault
	}
	tx = tx.Save(comment)
	if err := tx.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	log.Printf("updated comment to %v\n", comment)
	return comment, nil
}

// List all comments related to a single item
func (m *GormCommentRepository) List(ctx context.Context, parent string, start *int, limit *int) ([]*Comment, uint64, error) {
	defer goa.MeasureSince([]string{"goa", "db", "comment", "query"}, time.Now())

	db := m.db.Model(&Comment{}).Where("parent_id = ?", parent)
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
	db = db.Select("count(*) over () as cnt2 , *").Order("created_at desc")

	rows, err := db.Rows()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := []*Comment{}
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
		value := &Comment{}
		db.ScanRows(rows, value)
		if first {
			first = false
			if err = rows.Scan(columnValues...); err != nil {
				return nil, 0, errors.NewInternalError(err.Error())
			}
		}
		result = append(result, value)

	}
	if first {
		// means 0 rows were returned from the first query (maybe because of offset outside of total count),
		// need to do a count(*) to find out total
		orgDB := orgDB.Select("count(*)")
		rows2, err := orgDB.Rows()
		defer rows2.Close()
		if err != nil {
			return nil, 0, err
		}
		rows2.Next() // count(*) will always return a row
		rows2.Scan(&count)
	}
	return result, count, nil
}

// Count all comments related to a single item
func (m *GormCommentRepository) Count(ctx context.Context, parent string) (int, error) {
	defer goa.MeasureSince([]string{"goa", "db", "comment", "query"}, time.Now())
	var count int

	m.db.Model(&Comment{}).Where("parent_id = ?", parent).Count(&count)

	return count, nil
}

// Load a single comment regardless of parent
func (m *GormCommentRepository) Load(ctx context.Context, id uuid.UUID) (*Comment, error) {
	defer goa.MeasureSince([]string{"goa", "db", "comment", "get"}, time.Now())
	var obj Comment

	tx := m.db.Where("id=?", id).First(&obj)
	if tx.RecordNotFound() {
		return nil, errors.NewNotFoundError("comment", id.String())
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(tx.Error.Error())
	}
	return &obj, nil
}
