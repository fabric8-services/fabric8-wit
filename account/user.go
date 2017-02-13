package account

import (
	"time"

	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/log"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

// User describes a User account. A few identities can be assosiated with one user account
type User struct {
	gormsupport.Lifecycle
	ID         uuid.UUID  `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	Email      string     `sql:"unique_index"`                                            // This is the unique email field
	FullName   string     // The fullname of the User
	ImageURL   string     // The image URL for the User
	Bio        string     // The bio of the User
	URL        string     // The URL of the User
	Identities []Identity // has many Identities from different IDPs
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m User) TableName() string {
	return "users"
}

// GormUserRepository is the implementation of the storage interface for User.
type GormUserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new storage type.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{db: db}
}

// UserRepository represents the storage interface.
type UserRepository interface {
	Load(ctx context.Context, ID uuid.UUID) (*User, error)
	Create(ctx context.Context, u *User) error
	Save(ctx context.Context, u *User) error
	List(ctx context.Context) ([]*User, error)
	Delete(ctx context.Context, ID uuid.UUID) error
	Query(funcs ...func(*gorm.DB) *gorm.DB) ([]*User, error)
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *GormUserRepository) TableName() string {
	return "users"
}

// CRUD Functions

// Load returns a single User as a Database Model
// This is more for use internally, and probably not what you want in  your controllers
func (m *GormUserRepository) Load(ctx context.Context, id uuid.UUID) (*User, error) {
	defer goa.MeasureSince([]string{"goa", "db", "user", "load"}, time.Now())

	var native User
	err := m.db.Table(m.TableName()).Where("id = ?", id).Find(&native).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	return &native, errors.WithStack(err)
}

// Create creates a new record.
func (m *GormUserRepository) Create(ctx context.Context, u *User) error {
	defer goa.MeasureSince([]string{"goa", "db", "user", "create"}, time.Now())

	u.ID = uuid.NewV4()

	err := m.db.Create(u).Error
	if err != nil {
		log.LoggerRuntimeContext().WithFields(map[string]interface{}{
			"userID": u.ID,
			"err":    err.Error(),
		}).Errorln("Unable to create the user")
		goa.LogError(ctx, "error adding User", "error", err.Error())
		return errors.WithStack(err)
	}

	log.Logger().WithFields(map[string]interface{}{
		"pkg":    "user",
		"userID": u.ID,
	}).Debugln("User created!")

	return nil
}

// Save modifies a single record
func (m *GormUserRepository) Save(ctx context.Context, model *User) error {
	defer goa.MeasureSince([]string{"goa", "db", "user", "save"}, time.Now())

	obj, err := m.Load(ctx, model.ID)
	if err != nil {
		log.Logger().WithFields(map[string]interface{}{
			"pkg":    "user",
			"userID": model.ID,
			"err":    err.Error(),
		}).Errorln("Error updating User")
		goa.LogError(ctx, "error updating User", "error", err.Error())
		return errors.WithStack(err)
	}
	err = m.db.Model(obj).Updates(model).Error
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger().WithFields(map[string]interface{}{
		"pkg":    "user",
		"userID": model.ID,
	}).Debugln("User saved!")
	return nil
}

// Delete removes a single record.
func (m *GormUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "user", "delete"}, time.Now())

	var obj User

	err := m.db.Delete(&obj, id).Error

	if err != nil {
		log.LoggerRuntimeContext().WithFields(map[string]interface{}{
			"userID": id,
			"err":    err.Error(),
		}).Errorln("Unable to delete the user")
		goa.LogError(ctx, "error deleting User", "error", err.Error())
		return errors.WithStack(err)
	}

	log.Logger().WithFields(map[string]interface{}{
		"pkg":    "user",
		"userID": id,
	}).Debugln("User deleted!")

	return nil
}

// List return all users
func (m *GormUserRepository) List(ctx context.Context) ([]*User, error) {
	defer goa.MeasureSince([]string{"goa", "db", "user", "list"}, time.Now())
	var rows []*User

	err := m.db.Model(&User{}).Order("email").Find(&rows).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.WithStack(err)
	}
	return rows, nil
}

// Query expose an open ended Query model
func (m *GormUserRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]*User, error) {
	defer goa.MeasureSince([]string{"goa", "db", "user", "query"}, time.Now())
	var objs []*User

	err := m.db.Scopes(funcs...).Table(m.TableName()).Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.LoggerRuntimeContext().WithFields(map[string]interface{}{
			"err": errors.WithStack(err),
		}).Errorln("Error querying Users")
		return nil, errors.WithStack(err)
	}

	log.Logger().WithFields(map[string]interface{}{
		"pkg":    "user",
		"result": objs,
	}).Debugln("User query!")

	return objs, nil
}
