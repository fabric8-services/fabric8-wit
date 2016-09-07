package remoteworkitem

import (
	"log"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/models"
	"golang.org/x/net/context"
)

// GormTrackerRepository implements TrackerRepository using gorm
type GormTrackerRepository struct {
	ts *GormTransactionSupport
}

// NewTrackerRepository constructs a TrackerRepository
func NewTrackerRepository(ts *GormTransactionSupport) *GormTrackerRepository {
	return &GormTrackerRepository{ts}
}

var trackerTypes = map[string]string{
	"github": "Github",
	"jira":   "Jira"}

// Create creates a new tracker configuration in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormTrackerRepository) Create(ctx context.Context, url string, typeID string) (*app.Tracker, error) {
	_, present := trackerTypes[typeID]
	if present != true {
		return nil, BadParameterError{parameter: "type", value: typeID}
	}
	t := Tracker{
		URL:  url,
		Type: typeID}
	tx := r.ts.tx
	if err := tx.Create(&t).Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Printf("created tracker %v\n", t)
	t2 := app.Tracker{
		ID:   string(t.ID),
		URL:  url,
		Type: typeID}

	return &t2, nil
}

// Load returns the tracker configuration for the given id
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerRepository) Load(ctx context.Context, ID string) (*app.Tracker, error) {
	id := ID

	log.Printf("loading tracker %d", id)
	res := Tracker{}
	if r.ts.tx.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{"tracker", ID}
	}
	t := app.Tracker{
		ID:   res.ID,
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
	db := r.ts.tx.Where(where, parameters)
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

	return result, nil
}

// Save updates the given tracker in storage.
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerRepository) Save(ctx context.Context, t app.Tracker) (*app.Tracker, error) {
	res := Tracker{}
	id := t.ID

	log.Printf("looking for id %d", id)
	tx := r.ts.tx
	if tx.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{entity: "tracker", ID: t.ID}
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
		ID:   string(id),
		URL:  t.URL,
		Type: t.Type}

	return &t2, nil
}

// Delete deletes the tracker with the given id
// returns NotFoundError or InternalError
func (r *GormTrackerRepository) Delete(ctx context.Context, ID string) error {
	var t = Tracker{}
	id := ID
	t.ID = id
	tx := r.ts.tx

	if err := tx.Delete(t).Error; err != nil {
		if tx.RecordNotFound() {
			return NotFoundError{entity: "tracker", ID: ID}
		}
		return InternalError{simpleError{err.Error()}}
	}

	return nil
}
