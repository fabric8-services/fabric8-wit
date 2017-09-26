package workitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	gormbench "github.com/fabric8-services/fabric8-wit/gormtestsupport/benchmark"
	"github.com/fabric8-services/fabric8-wit/path"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"golang.org/x/net/context"
)

type BenchWorkItemTypeRepository struct {
	gormbench.DBBenchSuite
	repo workitem.WorkItemTypeRepository
}

func BenchmarkRunWorkItemTypeRepository(b *testing.B) {
	testsupport.Run(b, &BenchWorkItemTypeRepository{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

func (s *BenchWorkItemTypeRepository) SetupBenchmark() {
	s.DBBenchSuite.SetupBenchmark()
	s.repo = workitem.NewWorkItemTypeRepository(s.DB)
}

func (r *BenchWorkItemTypeRepository) BenchmarkLoad() {
	// given
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItemTypes(1))

	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		res := workitem.WorkItemType{}
		db := r.DB.Model(&res).Where("id=? AND space_id=?", fxt.WorkItemTypes[0].ID, fxt.WorkItemTypes[0].SpaceID).First(&res)
		if db.RecordNotFound() {
			r.B().Fail()
		}
		if err := db.Error; err != nil {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkLoadTypeFromDB() {
	// given
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItemTypes(1))

	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		res := workitem.WorkItemType{}
		db := r.DB.Model(&res).Where("id=?", fxt.WorkItemTypes[0].ID).First(&res)
		if db.RecordNotFound() {
			r.B().Fail()
		}
		if err := db.Error; err != nil {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkLoadWorkItemType() {
	// given
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItemTypes(1))

	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		if s, err := r.repo.Load(context.Background(), fxt.WorkItemTypes[0].SpaceID, fxt.WorkItemTypes[0].ID); err != nil || (err == nil && s == nil) {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListWorkItemTypes() {
	// given
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItemTypes(1))

	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		if s, err := r.repo.List(context.Background(), fxt.WorkItemTypes[0].SpaceID, nil, nil); err != nil || (err == nil && s == nil) {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListWorkItemTypesTransaction() {
	// given
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItemTypes(1))

	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		if err := application.Transactional(gormapplication.NewGormDB(r.DB), func(app application.Application) error {
			_, err := r.repo.List(context.Background(), fxt.WorkItemTypes[0].SpaceID, nil, nil)
			return err
		}); err != nil {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListPlannerItems() {
	// given
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItemTypes(1))

	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.WorkItemType
		path := path.Path{}
		db := r.DB.Select("id").Where("space_id = ? AND path::text LIKE '"+path.ConvertToLtree(fxt.WorkItemTypes[0].ID)+".%'", fxt.WorkItemTypes[0].SpaceID.String())

		if err := db.Find(&rows).Error; err != nil {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListFind() {
	// given
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItemTypes(1))

	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.WorkItemType
		db := r.DB.Where("space_id = ?", fxt.WorkItemTypes[0].SpaceID)
		if err := db.Find(&rows).Error; err != nil {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListRawScan() {
	// given
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItemTypes(1))

	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.WorkItemType
		result, err := r.DB.Raw("select  from work_item_types where space_id = ?", fxt.WorkItemTypes[0].SpaceID).Rows()
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
	// given
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItemTypes(1))

	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.WorkItemType
		result, err := r.DB.Raw("select * from work_item_types where space_id = ?", fxt.WorkItemTypes[0].SpaceID).Rows()
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
	// given
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItemTypes(1))

	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		var rows []string
		result, err := r.DB.Raw("select name from work_item_types where space_id = ?", fxt.WorkItemTypes[0].SpaceID).Rows()
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
	// given
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItemTypes(1))

	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.FieldDefinition
		result, err := r.DB.Raw("select fields from work_item_types where space_id = ?", fxt.WorkItemTypes[0].SpaceID).Rows()
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
