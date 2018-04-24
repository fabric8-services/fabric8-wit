package workitem

import (
	"context"
	"time"

	"github.com/fabric8-services/fabric8-wit/spacetemplate"

	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// WorkItemTypeGroupRepository encapsulates storage & retrieval of work item
// type groups
type WorkItemTypeGroupRepository interface {
	repository.Exister
	Create(ctx context.Context, group WorkItemTypeGroup) (*WorkItemTypeGroup, error)
	Load(ctx context.Context, groupID uuid.UUID) (*WorkItemTypeGroup, error)
	LoadByName(ctx context.Context, spaceTemplateID uuid.UUID, name string) (*WorkItemTypeGroup, error)
	List(ctx context.Context, spaceTemplateID uuid.UUID) ([]*WorkItemTypeGroup, error)
}

// NewWorkItemTypeGroupRepository creates a wi type group repository based on
// gorm
func NewWorkItemTypeGroupRepository(db *gorm.DB) *GormWorkItemTypeGroupRepository {
	return &GormWorkItemTypeGroupRepository{db}
}

// GormWorkItemTypeGroupRepository implements WorkItemTypeGroupRepository using
// gorm
type GormWorkItemTypeGroupRepository struct {
	db *gorm.DB
}

// Load returns the work item type group for the given id
func (r *GormWorkItemTypeGroupRepository) Load(ctx context.Context, groupID uuid.UUID) (*WorkItemTypeGroup, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtypegroup", "load"}, time.Now())
	log.Debug(ctx, map[string]interface{}{"witg_id": groupID}, "loading work item type group ")
	res := WorkItemTypeGroup{}
	db := r.db.Model(&res).Where("id=?", groupID).First(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{"witg_id": groupID}, "work item type group not found")
		return nil, errors.NewNotFoundError("work item type group", groupID.String())
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	typeList, err := r.loadTypeList(ctx, res.ID)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	res.TypeList = typeList
	return &res, nil
}

// LoadByName returns the work item type group for the given name and space
// template
func (r *GormWorkItemTypeGroupRepository) LoadByName(ctx context.Context, spaceTemplateID uuid.UUID, name string) (*WorkItemTypeGroup, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtypegroup", "load_by_name"}, time.Now())
	log.Debug(ctx, map[string]interface{}{"witg_name": name, "space_template_id": spaceTemplateID}, "loading work item type group by name")
	res := WorkItemTypeGroup{}
	db := r.db.Model(&res).Where("space_template_id=? AND name=?", spaceTemplateID, name).First(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{"witg_name": name, "space_template_id": spaceTemplateID}, "work item type group not found")
		return nil, errors.NewNotFoundError("work item type group", name)
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	typeList, err := r.loadTypeList(ctx, res.ID)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	res.TypeList = typeList
	return &res, nil
}

// loadTypeList loads all work item type associated with the given group and
func (r *GormWorkItemTypeGroupRepository) loadTypeList(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	members := []typeGroupMember{}
	db := r.db.Model(&members).Where("type_group_id=?", groupID).Order("position ASC").Find(&members)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{"witg_id": groupID}, "work item type group members not found")
		return nil, errors.NewNotFoundError("work item type group members of group", groupID.String())
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	res := make([]uuid.UUID, len(members))
	for i, member := range members {
		res[i] = member.WorkItemTypeID
	}
	return res, nil
}

// List returns all work item type groups for the given space template ID
// ordered by their position value.
func (r *GormWorkItemTypeGroupRepository) List(ctx context.Context, spaceTemplateID uuid.UUID) ([]*WorkItemTypeGroup, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtypegroup", "list"}, time.Now())
	log.Debug(ctx, map[string]interface{}{"space_template_id": spaceTemplateID}, "loading work item type groups for space template")

	// check space template exists
	if err := spacetemplate.NewRepository(r.db).CheckExists(ctx, spaceTemplateID); err != nil {
		return nil, errors.NewNotFoundError("space template", spaceTemplateID.String())
	}

	res := []*WorkItemTypeGroup{}
	db := r.db.Model(&res).Where("space_template_id=?", spaceTemplateID).Order("position ASC").Find(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{"space_template_id": spaceTemplateID}, "work item type groups not found")
		return nil, errors.NewNotFoundError("work item type groups for space template", spaceTemplateID.String())
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	for _, group := range res {
		typeList, err := r.loadTypeList(ctx, group.ID)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		group.TypeList = typeList
	}
	return res, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormWorkItemTypeGroupRepository) CheckExists(ctx context.Context, id uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtypegroup", "exists"}, time.Now())
	log.Debug(ctx, map[string]interface{}{"witg_id": id}, "checking if work item type group exists")
	return repository.CheckExists(ctx, r.db, WorkItemTypeGroup{}.TableName(), id)
}

// Create creates a new work item type group in the repository
func (r *GormWorkItemTypeGroupRepository) Create(ctx context.Context, g WorkItemTypeGroup) (*WorkItemTypeGroup, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtypegroup", "create"}, time.Now())
	if len(g.TypeList) <= 0 {
		return nil, errors.NewBadParameterError("type_list", g.TypeList).Expected("not empty")
	}
	if g.ID == uuid.Nil {
		g.ID = uuid.NewV4()
	}
	db := r.db.Create(&g)
	if db.Error != nil {
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	log.Debug(ctx, map[string]interface{}{"witg_id": g.ID}, "created work item type group")
	// Create entries for each member in the type list
	for idx, ID := range g.TypeList {
		member := typeGroupMember{
			TypeGroupID:    g.ID,
			WorkItemTypeID: ID,
			Position:       idx,
		}
		db = db.Create(&member)
		if db.Error != nil {
			return nil, errors.NewInternalError(ctx, db.Error)
		}
	}
	return &g, nil
}
