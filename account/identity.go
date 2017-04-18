package account

import (
	"database/sql/driver"
	"time"

	"github.com/almighty/almighty-core/app"
	errs "github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/log"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

const (
	// KeycloakIDP is the name of the main Keycloak Identity Provider
	KeycloakIDP string = "kc"
)

// NullUUID can be used with the standard sql package to represent a
// UUID value that can be NULL in the database
type NullUUID struct {
	UUID  uuid.UUID
	Valid bool
}

// Scan implements the sql.Scanner interface.
func (u *NullUUID) Scan(src interface{}) error {
	if src == nil {
		u.UUID, u.Valid = uuid.Nil, false
		return nil
	}

	// Delegate to UUID Scan function
	u.Valid = true

	switch src := src.(type) {
	case uuid.UUID:
		return u.UUID.Scan(src.Bytes())
	}

	return u.UUID.Scan(src)
}

// Value implements the driver.Valuer interface.
func (u NullUUID) Value() (driver.Value, error) {
	if !u.Valid {
		return nil, nil
	}
	// Delegate to UUID Value function
	return u.UUID.Value()
}

// Identity describes a federated identity provided by Identity Provider (IDP) such as Keycloak, GitHub, OSO, etc.
// One User account can have many Identities
type Identity struct {
	gormsupport.Lifecycle
	// This is the ID PK field. For identities provided by Keyclaok this ID equals to the Keycloak. For other types of IDP (github, oso, etc) this ID is generated automaticaly
	ID uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	// The username of the Identity
	Username string
	// ProviderType The type of provider, such as "keycloak", "github", "oso", etc
	ProviderType string `gorm:"column:provider_type"`
	// the URL of the profile on the remote work item service
	ProfileURL *string `gorm:"column:profile_url"`
	// Link to User
	UserID NullUUID `sql:"type:uuid"`
	User   User
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m Identity) TableName() string {
	return "identities"
}

// TODO: Remove. Data layer should not know about the REST layer. Moved to /users.go
// ConvertIdentityFromModel convert identity from model to app representation
func (m Identity) ConvertIdentityFromModel() *app.Identity {
	id := m.ID.String()
	converted := app.Identity{
		Data: &app.IdentityData{
			ID:   &id,
			Type: "identities",
			Attributes: &app.IdentityDataAttributes{
				Username:     &m.Username,
				ProviderType: &m.ProviderType,
			},
		},
	}
	return &converted
}

// GormIdentityRepository is the implementation of the storage interface for
// Identity.
type GormIdentityRepository struct {
	db *gorm.DB
}

// NewIdentityRepository creates a new storage type.
func NewIdentityRepository(db *gorm.DB) *GormIdentityRepository {
	return &GormIdentityRepository{db: db}
}

// IdentityRepository represents the storage interface.
type IdentityRepository interface {
	Load(ctx context.Context, id uuid.UUID) (*Identity, error)
	Create(ctx context.Context, identity *Identity) error
	Lookup(ctx context.Context, username, profileURL, providerType string) (*Identity, error)
	Save(ctx context.Context, identity *Identity) error
	Delete(ctx context.Context, id uuid.UUID) error
	Query(funcs ...func(*gorm.DB) *gorm.DB) ([]*Identity, error)
	List(ctx context.Context) (*app.IdentityArray, error)
	IsValid(context.Context, uuid.UUID) bool
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
		return nil, errors.WithStack(err)
	}

	return &native, errors.WithStack(err)
}

// Create creates a new record.
func (m *GormIdentityRepository) Create(ctx context.Context, model *Identity) error {
	defer goa.MeasureSince([]string{"goa", "db", "identity", "create"}, time.Now())

	if model.ID == uuid.Nil {
		model.ID = uuid.NewV4()
	}
	err := m.db.Create(model).Error
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"identity_id": model.ID,
			"err":         err,
		}, "unable to create the identity")
		return errors.WithStack(err)
	}

	log.Debug(ctx, map[string]interface{}{
		"identity_id": model.ID,
	}, "Identity created!")

	return nil
}

// Lookup looks for an existing identity with the given `profileURL` or creates a new one
func (m *GormIdentityRepository) Lookup(ctx context.Context, username, profileURL, providerType string) (*Identity, error) {
	if username == "" || profileURL == "" || providerType == "" {
		return nil, errors.New("Cannot lookup identity with empty username, profile URL or provider type")
	}
	log.Debug(nil, nil, "Looking for identity of user with profile URL=%s\n", profileURL)
	// bind the assignee to an existing identity, or create a new one
	identity, err := m.First(IdentityFilterByProfileURL(profileURL))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to lookup identity by profileURL '%s'", profileURL)
	}
	if identity == nil {
		// create the identity if it does not exist yet
		log.Debug(nil, nil, "Creating an identity for username '%s' with profile '%s' on '%s'\n", username, profileURL, providerType)
		identity = &Identity{
			ProviderType: providerType,
			Username:     username,
			ProfileURL:   &profileURL,
		}
		err = m.Create(context.Background(), identity)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create identity during lookup")
		}
	} else {
		// use existing identity
		log.Debug(nil, nil, "Using existing identity with ID: %v", identity.ID.String())
	}
	log.Debug(nil, nil, "Found identity of user with profile URL=%s: %s", profileURL, identity.ID)
	return identity, nil
}

// Save modifies a single record.
func (m *GormIdentityRepository) Save(ctx context.Context, model *Identity) error {
	defer goa.MeasureSince([]string{"goa", "db", "identity", "save"}, time.Now())

	obj, err := m.Load(ctx, model.ID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"identity_id": model.ID,
			"ctx":         ctx,
			"err":         err,
		}, "unable to update the identity")
		return errors.WithStack(err)
	}
	err = m.db.Model(obj).Updates(model).Error

	log.Debug(ctx, map[string]interface{}{
		"identity_id": model.ID,
	}, "Identity saved!")

	return errors.WithStack(err)
}

// Delete removes a single record.
func (m *GormIdentityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "identity", "delete"}, time.Now())

	obj := Identity{ID: id}
	db := m.db.Delete(obj)

	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"identity_id": id,
			"err":         db.Error,
		}, "unable to delete the identity")
		return errors.WithStack(db.Error)
	}
	if db.RowsAffected == 0 {
		return errs.NewNotFoundError("identity", id.String())
	}

	log.Debug(ctx, map[string]interface{}{
		"identity_id": id,
	}, "Identity deleted!")

	return nil
}

// Query expose an open ended Query model
func (m *GormIdentityRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]*Identity, error) {
	defer goa.MeasureSince([]string{"goa", "db", "identity", "query"}, time.Now())
	var objs []*Identity

	err := m.db.Scopes(funcs...).Table(m.TableName()).Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.WithStack(err)
	}

	log.Debug(nil, map[string]interface{}{
		"identity_list": objs,
	}, "Identity query executed successfully!")

	return objs, nil
}

// First returns the first Identity element that matches the given criteria
func (m *GormIdentityRepository) First(funcs ...func(*gorm.DB) *gorm.DB) (*Identity, error) {
	defer goa.MeasureSince([]string{"goa", "db", "identity", "first"}, time.Now())
	var objs []*Identity
	log.Debug(nil, nil, "Looking for identity matching: %v", funcs)

	err := m.db.Scopes(funcs...).Table(m.TableName()).First(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.WithStack(err)
	}
	if len(objs) != 0 && objs[0] != nil {
		log.Debug(nil, map[string]interface{}{
			"identity_list": objs,
		}, "Found matching identity: %v", *objs[0])
		return objs[0], nil
	}
	log.Debug(nil, map[string]interface{}{
		"identity_list": objs,
	}, "No matching identity found")
	return nil, nil
}

// IdentityFilterByUserID is a gorm filter for a Belongs To relationship.
func IdentityFilterByUserID(userID uuid.UUID) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ?", userID)
	}
}

// IdentityFilterByUsername is a gorm filter by 'username'
func IdentityFilterByUsername(username string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("username = ?", username).Limit(1)
	}
}

// IdentityFilterByProfileURL is a gorm filter by 'profile_url'
func IdentityFilterByProfileURL(profileURL string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("profile_url = ?", profileURL).Limit(1)
	}
}

// IdentityFilterByID is a gorm filter for Identity ID.
func IdentityFilterByID(identityID uuid.UUID) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", identityID)
	}
}

// IdentityWithUser is a gorm filter for preloading the User relationship.
func IdentityWithUser() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Preload("User")
	}
}

// IdentityFilterByProviderType is a gorm filter by 'provider_type'
func IdentityFilterByProviderType(providerType string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("provider_type = ?", providerType)
	}
}

// IdentityFilterByRegistrationCompleted is a gorm filter by 'registration_completed'
func IdentityFilterByRegistrationCompleted(registrationCompleted bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("registration_completed = ?", registrationCompleted)
	}
}

// List return all user identities
func (m *GormIdentityRepository) List(ctx context.Context) (*app.IdentityArray, error) {
	defer goa.MeasureSince([]string{"goa", "db", "identity", "list"}, time.Now())
	var rows []Identity

	err := m.db.Model(&Identity{}).Order("username").Find(&rows).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.WithStack(err)
	}
	res := app.IdentityArray{}
	res.Data = make([]*app.IdentityData, len(rows))
	for index, value := range rows {
		ident := value.ConvertIdentityFromModel()
		res.Data[index] = ident.Data
	}

	log.Debug(ctx, map[string]interface{}{
		"identity_list": &res,
	}, "Identity List executed successfully!")

	return &res, nil
}

// IsValid returns true if the identity exists
func (m *GormIdentityRepository) IsValid(ctx context.Context, id uuid.UUID) bool {
	_, err := m.Load(ctx, id)
	if err != nil {
		return false
	}
	return true
}
