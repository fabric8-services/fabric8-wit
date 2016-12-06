package models

import (
	"log"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/project"
	"github.com/jinzhu/gorm"
	satoriuuid "github.com/satori/go.uuid"
)

// NewProjectRepository creates a new project repo
func NewProjectRepository(db *gorm.DB) *GormProjectRepository {
	return &GormProjectRepository{db}
}

// GormProjectRepository implements ProjectRepository using gorm
type GormProjectRepository struct {
	db *gorm.DB
}

// Load returns the project for the given id
// returns NotFoundError or InternalError
func (r *GormProjectRepository) Load(ctx context.Context, ID satoriuuid.UUID) (*project.Project, error) {
	res := project.Project{}
	tx := r.db.Where("id=?", ID).First(&res)
	if tx.RecordNotFound() {
		return nil, NewNotFoundError("project", ID.String())
	}
	if tx.Error != nil {
		return nil, NewInternalError(tx.Error.Error())
	}
	return &res, nil
}

// Delete deletes the project with the given id
// returns NotFoundError or InternalError
func (r *GormProjectRepository) Delete(ctx context.Context, ID satoriuuid.UUID) error {
	project := project.Project{ID: ID}
	tx := r.db.Delete(project)

	if err := tx.Error; err != nil {
		return NewInternalError(err.Error())
	}
	if tx.RowsAffected == 0 {
		return NewNotFoundError("project", ID.String())
	}

	return nil
}

// Save updates the given project in the db. Version must be the same as the one in the stored version
// returns NotFoundError, BadParameterError, VersionConflictError or InternalError
func (r *GormProjectRepository) Save(ctx context.Context, p project.Project) (*project.Project, error) {
	pr := project.Project{}
	tx := r.db.Where("id=?", p.ID).First(&pr)
	oldVersion := p.Version
	p.Version++
	if tx.RecordNotFound() {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"project", p.ID.String()}
	}
	if err := tx.Error; err != nil {
		return nil, NewInternalError(err.Error())
	}
	tx = tx.Where("Version = ?", oldVersion).Save(&p)
	if err := tx.Error; err != nil {
		if isCheckViolation(tx.Error, "projects_name_check") {
			return nil, NewBadParameterError("Name", p.Name).Expected("not empty")
		}
		if isUniqueViolation(tx.Error, "projects_name_idx") {
			return nil, NewBadParameterError("Name", p.Name).Expected("unique")
		}
		return nil, NewInternalError(err.Error())
	}
	if tx.RowsAffected == 0 {
		return nil, NewVersionConflictError("version conflict")
	}
	log.Printf("updated project to %v\n", p)
	return &p, nil
}

// Create creates a new Project in the db
// returns BadParameterError or InternalError
func (r *GormProjectRepository) Create(ctx context.Context, name string) (*project.Project, error) {
	newProject := project.Project{
		Name: name,
	}

	tx := r.db.Create(&newProject)
	if err := tx.Error; err != nil {
		if isCheckViolation(tx.Error, "projects_name_check") {
			return nil, NewBadParameterError("Name", name).Expected("not empty")
		}
		if isUniqueViolation(tx.Error, "projects_name_idx") {
			return nil, NewBadParameterError("Name", name).Expected("unique")
		}
		return nil, NewInternalError(err.Error())
	}
	log.Printf("created project %v\n", newProject)
	return &newProject, nil
}

// extracted this function from List() in order to close the rows object with "defer" for more readability
// workaround for https://github.com/lib/pq/issues/81
func (r *GormProjectRepository) listProjectFromDB(ctx context.Context, start *int, limit *int) ([]project.Project, uint64, error) {

	db := r.db.Model(&project.Project{})
	orgDB := db
	if start != nil {
		if *start < 0 {
			return nil, 0, NewBadParameterError("start", *start)
		}
		db = db.Offset(*start)
	}
	if limit != nil {
		if *limit <= 0 {
			return nil, 0, NewBadParameterError("limit", *limit)
		}
		db = db.Limit(*limit)
	}
	db = db.Select("count(*) over () as cnt2 , *")

	rows, err := db.Rows()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := []project.Project{}
	value := project.Project{}
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, NewInternalError(err.Error())
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
		db.ScanRows(rows, &value)
		if first {
			first = false
			if err = rows.Scan(columnValues...); err != nil {
				return nil, 0, NewInternalError(err.Error())
			}
		}
		result = append(result, value)

	}
	if first {
		// means 0 rows were returned from the first query (maybe becaus of offset outside of total count),
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

// List returns work item selected by the given criteria.Expression, starting with start (zero-based) and returning at most limit items
func (r *GormProjectRepository) List(ctx context.Context, start *int, limit *int) ([]project.Project, uint64, error) {
	result, count, err := r.listProjectFromDB(ctx, start, limit)
	if err != nil {
		return nil, 0, err
	}

	return result, count, nil
}
