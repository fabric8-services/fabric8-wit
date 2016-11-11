package account

import (
	"strings"
	"time"

	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/models"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

// Identity ddenDescribes a unique Person with the ALM
type Identity struct {
	gormsupport.Lifecycle
	ID       uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	Emails   []User    // has many Users
	FullName string    // The fullname of the Identity
	ImageURL string    // The image URL for this Identity
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m Identity) TableName() string {
	return "identities"

}

// GormIdentityRepository is the implementation of the storage interface for
// Identity.
type GormIdentityRepository struct {
	db *gorm.DB
}

// NewIdentityRepository creates a new storage type.
func NewIdentityRepository(db *gorm.DB) IdentityRepository {
	return &GormIdentityRepository{db: db}
}

// IdentityRepository represents the storage interface.
type IdentityRepository interface {
	Load(ctx context.Context, id uuid.UUID) (*Identity, error)
	Create(ctx context.Context, identity *Identity) error
	Save(ctx context.Context, identity *Identity) error
	Delete(ctx context.Context, id uuid.UUID) error
	Query(funcs ...func(*gorm.DB) *gorm.DB) ([]*Identity, error)
	SearchByFullName(ctx context.Context, q string, start int, limit int) ([]Identity, int, error)
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *GormIdentityRepository) TableName() string {
	return "identities"

}

// CRUD Functions

// Load returns a single Identity as a Database Model
// This is more for use internally, and probably not what you want in  your controllers
func (m *GormIdentityRepository) Load(ctx context.Context, id uuid.UUID) (*Identity, error) {
	defer goa.MeasureSince([]string{"goa", "db", "identity", "load"}, time.Now())

	var native Identity
	err := m.db.Table(m.TableName()).Where("id = ?", id).Find(&native).Error
	if err == gorm.ErrRecordNotFound {
		return nil, err
	}

	return &native, err
}

// Create creates a new record.
func (m *GormIdentityRepository) Create(ctx context.Context, model *Identity) error {
	defer goa.MeasureSince([]string{"goa", "db", "identity", "create"}, time.Now())

	model.ID = uuid.NewV4()

	err := m.db.Create(model).Error
	if err != nil {
		goa.LogError(ctx, "error adding Identity", "error", err.Error())
		return err
	}

	return nil
}

// Save modifies a single record.
func (m *GormIdentityRepository) Save(ctx context.Context, model *Identity) error {
	defer goa.MeasureSince([]string{"goa", "db", "identity", "save"}, time.Now())

	obj, err := m.Load(ctx, model.ID)
	if err != nil {
		goa.LogError(ctx, "error updating Identity", "error", err.Error())
		return err
	}
	err = m.db.Model(obj).Updates(model).Error

	return err
}

// Delete removes a single record.
func (m *GormIdentityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "identity", "delete"}, time.Now())

	var obj Identity

	err := m.db.Delete(&obj, id).Error

	if err != nil {
		goa.LogError(ctx, "error deleting Identity", "error", err.Error())
		return err
	}

	return nil
}

// Query expose an open ended Query model
func (m *GormIdentityRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]*Identity, error) {
	defer goa.MeasureSince([]string{"goa", "db", "identity", "query"}, time.Now())
	var objs []*Identity

	err := m.db.Scopes(funcs...).Table(m.TableName()).Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return objs, nil
}

// SearchByFullName searches for Identites where FullNmae like q%
func (m *GormIdentityRepository) SearchByFullName(ctx context.Context, q string, start int, limit int) ([]Identity, int, error) {

	db := m.db.Model(&Identity{}).Where("LOWER(full_name) like ?", strings.ToLower(q)+"%")
	db = db.Offset(start)
	db = db.Limit(limit)
	db = db.Select("count(*) over () as cnt2 , *")

	rows, err := db.Rows()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := []Identity{}
	value := Identity{}
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, models.NewInternalError(err.Error())
	}

	// need to set up a result for Scan() in order to extract total count.
	var count int
	var ignore interface{}
	columnValues := make([]interface{}, len(columns))

	for index := range columnValues {
		columnValues[index] = &ignore
	}
	columnValues[0] = &count
	first := true

	for rows.Next() {
		db.ScanRows(rows, &value)
		if first {
			first = false
			if err = rows.Scan(columnValues...); err != nil {
				return nil, 0, models.NewInternalError(err.Error())
			}
		}
		result = append(result, value)
	}
	return result, count, nil
}
