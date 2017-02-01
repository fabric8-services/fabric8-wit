package remoteworkitem

import (
	"log"
	"time"

	"github.com/almighty/almighty-core/models"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
	uuid "github.com/satori/go.uuid"
)

// TrackerSchedule capture all configuration
type trackerSchedule struct {
	TrackerID      int
	URL            string
	TrackerType    string
	TrackerQueryID int
	Query          string
	Schedule       string
	LastUpdated    *time.Time
}

// Scheduler represents scheduler
type Scheduler struct {
	db *gorm.DB
}

var cr *cron.Cron

// NewScheduler creates a new Scheduler
func NewScheduler(db *gorm.DB) *Scheduler {
	s := Scheduler{db: db}
	return &s
}

// Stop scheduler
// This should be called only from main
func (s *Scheduler) Stop() {
	cr.Stop()
}

func batchID() string {
	u1 := uuid.NewV4().String()
	return u1
}

// ScheduleAllQueries fetch and import of remote tracker items
func (s *Scheduler) ScheduleAllQueries() {
	cr.Stop()

	trackerQueries := fetchTrackerQueries(s.db)
	for _, tq := range trackerQueries {
		cr.AddFunc(tq.Schedule, func() {
			tr := lookupProvider(tq)
			for i := range tr.Fetch() {
				models.Transactional(s.db, func(tx *gorm.DB) error {
					// Save the remote items in a 'temporary' table.
					err := upload(tx, tq.TrackerID, i)
					if err != nil {
						return errors.WithStack(err)
					}
					if i.LastUpdated != nil {
						err = updateTrackerQuery(tx, tq.TrackerQueryID, i.LastUpdated)
						if err != nil {
							log.Println("Couldn't update the last updated value", err)
						}
						tq.LastUpdated = i.LastUpdated
					}
					// Convert the remote item into a local work item and persist in the DB.
					_, err = convert(tx, tq.TrackerID, i, tq.TrackerType)
					return errors.WithStack(err)
				})
			}
		})
	}
	cr.Start()
}

func fetchTrackerQueries(db *gorm.DB) []trackerSchedule {
	tsList := []trackerSchedule{}
	err := db.Table("tracker_queries").Select("trackers.id as tracker_id, trackers.url, trackers.type as tracker_type, tracker_queries.id as tracker_query_id, tracker_queries.query, tracker_queries.schedule, tracker_queries.last_updated").Joins("left join trackers on tracker_queries.tracker_id = trackers.id").Where("trackers.deleted_at is NULL AND tracker_queries.deleted_at is NULL").Scan(&tsList).Error
	if err != nil {
		log.Printf("Fetch failed %v\n", err)
	}
	return tsList
}

// lookupProvider provides the respective tracker based on the type
func lookupProvider(ts trackerSchedule) TrackerProvider {
	q := ts.Query
	switch ts.TrackerType {
	case ProviderGithub:
		if ts.LastUpdated != nil {
			// Use the special date for formatting: https://golang.org/pkg/time/#Time.Format
			q = ts.Query + " updated:\">=" + ts.LastUpdated.Format("2006-01-02T15:04:05Z") + "\""
		}
		return &GithubTracker{URL: ts.URL, Query: q}
	case ProviderJira:
		if ts.LastUpdated != nil {
			// Use the special date for formatting: https://golang.org/pkg/time/#Time.Format
			q = ts.Query + " and updated >= \"" + ts.LastUpdated.Format("2006-01-02 15:04") + "\""
		}
		return &JiraTracker{URL: ts.URL, Query: q}
	}
	return nil
}

// TrackerItemContent represents a remote tracker item with it's content and unique ID
type TrackerItemContent struct {
	ID          string
	Content     []byte
	LastUpdated *time.Time
}

// TrackerProvider represents a remote tracker
type TrackerProvider interface {
	Fetch() chan TrackerItemContent // TODO: Change to an interface to enforce the contract
}

func init() {
	cr = cron.New()
	cr.Start()
}
