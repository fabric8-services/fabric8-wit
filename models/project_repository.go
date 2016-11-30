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
func (r *GormProjectRepository) Load(ctx context.Context, ID satoriuuid.UUID) (*project.Project, error) {
	log.Printf("loading project %v", ID)
	res := project.Project{}
	tx := r.db.First(&res, ID)
	if tx.RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{"project", ID.String()}
	}
	if tx.Error != nil {
		return nil, InternalError{simpleError{tx.Error.Error()}}
	}
	return &res, nil
}

// Delete deletes the work item with the given id
// returns NotFoundError or InternalError
func (r *GormProjectRepository) Delete(ctx context.Context, ID satoriuuid.UUID) error {
	project := project.Project{ID: ID}
	tx := r.db.Delete(project)

	if err := tx.Error; err != nil {
		return InternalError{simpleError{err.Error()}}
	}
	if tx.RowsAffected == 0 {
		return NotFoundError{entity: "project", ID: ID.String()}
	}

	return nil
}

// Save updates the given work item in storage. Version must be the same as the one int the stored version
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormProjectRepository) Save(ctx context.Context, p project.Project) (*project.Project, error) {
	tx := r.db.Debug().First(&project.Project{}, p.ID)
	p.Version = p.Version + 1
	if tx.RecordNotFound() {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"project", p.ID.String()}
	}
	if err := tx.Error; err != nil {
		log.Print(err.Error())
		return nil, InternalError{simpleError{err.Error()}}
	}
	tx = tx.Where("Version = ?", p.Version).Save(&p)
	if err := tx.Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	if tx.RowsAffected == 0 {
		return nil, VersionConflictError{simpleError{"version conflict"}}
	}
	log.Printf("updated project to %v\n", p)
	return &p, nil
}

// Create creates a new work item in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormProjectRepository) Create(ctx context.Context, name string) (*project.Project, error) {
	newProject := project.Project{
		Name: name,
	}

	tx := r.db.Debug().Create(&newProject)
	if err := tx.Error; err != nil {
		log.Print(err.Error())
		log.Printf("err: %v", tx.Error)
		return nil, InternalError{simpleError{err.Error()}}
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
		return nil, 0, InternalError{simpleError{err.Error()}}
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
				return nil, 0, InternalError{simpleError{err.Error()}}
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
