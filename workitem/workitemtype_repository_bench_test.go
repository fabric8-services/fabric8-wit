package workitem_test

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/gormsupport/cleaner"
	gormbench "github.com/almighty/almighty-core/gormtestsupport/benchmark"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/path"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"
)

type BenchWorkItemTypeRepository struct {
	gormbench.DBBenchSuite
	clean func()
	repo  workitem.WorkItemTypeRepository
	ctx   context.Context
}

func BenchmarkRunWorkItemTypeRepository(b *testing.B) {
	testsupport.Run(b, &BenchWorkItemTypeRepository{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

func (s *BenchWorkItemTypeRepository) SetupSuite() {
	s.DBBenchSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBBenchSuite.PopulateDBBenchSuite(s.ctx)
}

func (s *BenchWorkItemTypeRepository) SetupBenchmark() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.repo = workitem.NewWorkItemTypeRepository(s.DB)
}

func (s *BenchWorkItemTypeRepository) TearDownBenchmark() {
	s.clean()
}

func (r *BenchWorkItemTypeRepository) BenchmarkLoad() {
	r.B().ResetTimer()
	for n := 0; n < r.B().N; n++ {
		res := workitem.WorkItemType{}
		db := r.DB.Model(&res).Where("id=? AND space_id=?", workitem.SystemExperience, space.SystemSpace).First(&res)
		if db.RecordNotFound() {
			r.B().Fail()
		}
		if err := db.Error; err != nil {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkLoadTypeFromDB() {
	r.B().ResetTimer()
	for n := 0; n < r.B().N; n++ {
		res := workitem.WorkItemType{}
		db := r.DB.Model(&res).Where("id=?", workitem.SystemExperience).First(&res)
		if db.RecordNotFound() {
			r.B().Fail()
		}
		if err := db.Error; err != nil {
			r.B().Fail()
		}
	}
}

/*
func (r *BenchWorkItemTypeRepository) Create(ctx context.Context, spaceID uuid.UUID, id *uuid.UUID, extendedTypeID *uuid.UUID, name string, description *string, icon string, fields map[string]FieldDefinition) (*WorkItemType, error) {
	// Make sure this WIT has an ID
	if id == nil {
		tmpID := uuid.NewV4()
		id = &tmpID
	}

	allFields := map[string]FieldDefinition{}
	path := LtreeSafeID(*id)
	if extendedTypeID != nil {
		extendedType := WorkItemType{}
		db := r.db.Model(&extendedType).Where("id=?", extendedTypeID).First(&extendedType)
		if db.RecordNotFound() {
			return nil, errors.NewBadParameterError("extendedTypeID", *extendedTypeID)
		}
		if err := db.Error; err != nil {
			return nil, errors.NewInternalError(err.Error())
		}
		// copy fields from extended type
		for key, value := range extendedType.Fields {
			allFields[key] = value
		}
		path = extendedType.Path + pathSep + path
	}
	// now process new fields, checking whether they are already there.
	for field, definition := range fields {
		existing, exists := allFields[field]
		if exists && !compatibleFields(existing, definition) {
			return nil, fmt.Errorf("incompatible change for field %s", field)
		}
		allFields[field] = definition
	}

	created := WorkItemType{
		Version:     0,
		ID:          *id,
		Name:        name,
		Description: description,
		Icon:        icon,
		Path:        path,
		Fields:      allFields,
		SpaceID:     spaceID,
	}

	if err := r.db.Create(&created).Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}

	log.Debug(ctx, map[string]interface{}{"witID": created.ID}, "Work item type created successfully!")
	return &created, nil
}
*/
func (r *BenchWorkItemTypeRepository) BenchmarkListPlannerItems() {
	r.B().ResetTimer()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.WorkItemType
		path := path.Path{}
		db := r.DB.Select("id").Where("space_id = ? AND path::text LIKE '"+path.ConvertToLtree(workitem.SystemPlannerItem)+".%'", space.SystemSpace.String())

		if err := db.Find(&rows).Error; err != nil {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListFind() {
	r.B().ResetTimer()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.WorkItemType
		db := r.DB.Where("space_id = ?", space.SystemSpace)
		if err := db.Find(&rows).Error; err != nil {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListRawScan() {
	r.B().ResetTimer()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.WorkItemType
		result, err := r.DB.Raw("select  from work_item_types where space_id = ?", space.SystemSpace).Rows()
		if err != nil {
			r.B().Fail()
		}
		defer result.Close()
		for result.Next() {
			wit := workitem.WorkItemType{}
			result.Scan(&wit)
			rows = append(rows, wit)
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListRawScanAll() {
	r.B().ResetTimer()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.WorkItemType
		result, err := r.DB.Raw("select * from work_item_types where space_id = ?", space.SystemSpace).Rows()
		if err != nil {
			r.B().Fail()
		}
		defer result.Close()
		for result.Next() {
			wit := workitem.WorkItemType{}
			result.Scan(&wit)
			rows = append(rows, wit)
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListRawScanName() {
	r.B().ResetTimer()
	for n := 0; n < r.B().N; n++ {
		var rows []string
		result, err := r.DB.Raw("select name from work_item_types where space_id = ?", space.SystemSpace).Rows()
		if err != nil {
			r.B().Fail()
		}
		defer result.Close()
		for result.Next() {
			var witName string
			result.Scan(&witName)
			rows = append(rows, witName)
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListRawScanFields() {
	r.B().ResetTimer()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.FieldDefinition
		result, err := r.DB.Raw("select fields from work_item_types where space_id = ?", space.SystemSpace).Rows()
		if err != nil {
			r.B().Fail()
		}
		defer result.Close()
		for result.Next() {
			var field workitem.FieldDefinition
			result.Scan(&field)
			rows = append(rows, field)
		}
	}
}
