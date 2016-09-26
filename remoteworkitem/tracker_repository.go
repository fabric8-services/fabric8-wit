package remoteworkitem

import (
	"log"
	"strconv"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/models"
	"golang.org/x/net/context"
)

// GormTrackerRepository implements TrackerRepository using gorm
type GormTrackerRepository struct {
	ts *models.GormTransactionSupport
}

// NewTrackerRepository constructs a TrackerRepository
func NewTrackerRepository(ts *models.GormTransactionSupport) *GormTrackerRepository {
	return &GormTrackerRepository{ts}
}

// Create creates a new tracker configuration in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormTrackerRepository) Create(ctx context.Context, url string, typeID string) (*app.Tracker, error) {
	_, present := RemoteWorkItemImplRegistry[typeID]
	// Ensure we support this remote tracker.
	if present != true {
		return nil, BadParameterError{parameter: "type", value: typeID}
	}
	t := Tracker{
		URL:  url,
		Type: typeID}
	tx := r.ts.TX()
	if err := tx.Create(&t).Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Printf("created tracker %v\n", t)
	t2 := app.Tracker{
		ID:   strconv.FormatUint(t.ID, 10),
		URL:  url,
		Type: typeID}

	return &t2, nil
}

// Load returns the tracker configuration for the given id
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerRepository) Load(ctx context.Context, ID string) (*app.Tracker, error) {
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"tracker", ID}
	}

	log.Printf("loading tracker %d", id)
	res := Tracker{}
	if r.ts.TX().First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{"tracker", ID}
	}
	t := app.Tracker{
		ID:   strconv.FormatUint(res.ID, 10),
		URL:  res.URL,
		Type: res.Type}

	return &t, nil
}

// List returns tracker selected by the given criteria.Expression, starting with start (zero-based) and returning at most limit items
func (r *GormTrackerRepository) List(ctx context.Context, criteria criteria.Expression, start *int, limit *int) ([]*app.Tracker, error) {
	where, parameters, err := models.Compile(criteria)
	if err != nil {
		return nil, BadParameterError{"expression", criteria}
	}

	log.Printf("executing query: %s", where)

	var rows []Tracker
	db := r.ts.TX().Where(where, parameters)
	if start != nil {
		db = db.Offset(*start)
	}
	if limit != nil {
		db = db.Limit(*limit)
	}
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]*app.Tracker, len(rows))

	for i, tracker := range rows {
		t := app.Tracker{
			ID:   strconv.FormatUint(tracker.ID, 10),
			URL:  tracker.URL,
			Type: tracker.Type}
		result[i] = &t
	}
	return result, nil
}

// Save updates the given tracker in storage.
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerRepository) Save(ctx context.Context, t app.Tracker) (*app.Tracker, error) {
	res := Tracker{}
	id, err := strconv.ParseUint(t.ID, 10, 64)
	if err != nil {
		return nil, NotFoundError{entity: "tracker", ID: t.ID}
	}

	log.Printf("looking for id %d", id)
	tx := r.ts.TX()
	if tx.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{entity: "tracker", ID: t.ID}
	}
	_, present := RemoteWorkItemImplRegistry[t.Type]
	// Ensure we support this remote tracker.
	if present != true {
		return nil, BadParameterError{parameter: "type", value: t.Type}
	}

	newT := Tracker{
		ID:   id,
		URL:  t.URL,
		Type: t.Type}

	if err := tx.Save(&newT).Error; err != nil {
		log.Print(err.Error())
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Printf("updated tracker to %v\n", newT)
	t2 := app.Tracker{
		ID:   strconv.FormatUint(id, 10),
		URL:  t.URL,
		Type: t.Type}

	return &t2, nil
}

// Delete deletes the tracker with the given id
// returns NotFoundError or InternalError
func (r *GormTrackerRepository) Delete(ctx context.Context, ID string) error {
	var t = Tracker{}
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil {
		// treat as not found: clients don't know it must be a number
		return NotFoundError{entity: "tracker", ID: ID}
	}
	t.ID = id
	tx := r.ts.TX()
	tx = tx.Delete(t)
	if err = tx.Error; err != nil {
		return InternalError{simpleError{err.Error()}}
	}
	if tx.RowsAffected == 0 {
		return NotFoundError{entity: "tracker", ID: ID}
	}
	return nil
}
