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
			tr := lookupProvider(tq, s.db)
			tid := tq.TrackerID
			tqid := tq.TrackerQueryID
			tt := tq.TrackerType
			beforeLU := tr.LastUpdatedTime()
			for i := range tr.Fetch() {
				models.Transactional(s.db, func(tx *gorm.DB) error {
					// Save the remote items in a 'temporary' table.
					err := upload(tx, tid, i)
					if err != nil {
						return errors.WithStack(err)
					}
					if beforeLU == nil {
						err = updateTrackerQuery(tx, tqid, i.LastUpdated)
						if err != nil {
							log.Println("Couldn't update the last updated value", err)
						}
					} else {
						lu1 := beforeLU.AddDate(0, 0, 7)
						if i.LastUpdated.Before(lu1) {
							n := time.Now().AddDate(0, 0, -1)
							if i.LastUpdated.Before(n) {
								err = updateTrackerQuery(tx, tqid, i.LastUpdated)
								if err != nil {
									log.Println("Couldn't update the last updated value", err)
								}
							}
						}
					}
					// Convert the remote item into a local work item and persist in the DB.
					_, err = convert(tx, tid, i, tt)
					return errors.WithStack(err)
				})
			}
			if beforeLU != nil {
				if beforeLU.Equal(*tr.LastUpdatedTime()) {
					n := time.Now().AddDate(0, 0, -1)
					if n.Before(*tr.LastUpdatedTime()) {
						models.Transactional(s.db, func(tx *gorm.DB) error {
							lu := beforeLU.AddDate(0, 0, 7)
							err := updateTrackerQuery(tx, tqid, &lu)
							if err != nil {
								log.Println("Couldn't update the last updated value", err)
							}
							return err
						})
					}
				}
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
func lookupProvider(ts trackerSchedule, db *gorm.DB) TrackerProvider {
	q := ts.Query
	tq := TrackerQuery{}
	tx := db.First(&tq, ts.TrackerQueryID)
	if tx.RecordNotFound() {
		log.Printf("not found, res=%v", tq)
	}

	switch ts.TrackerType {
	case ProviderGithub:
		if tq.LastUpdated != nil {
			// Use the special date for formatting: https://golang.org/pkg/time/#Time.Format
			q = ts.Query + " updated:\"" + tq.LastUpdated.Format("2006-01-02T15:04") + " .. " + tq.LastUpdated.AddDate(0, 0, 7).Format("2006-01-02T15:04") + "\""
			return &GithubTracker{URL: ts.URL, Query: q, LastUpdated: tq.LastUpdated}
		}

		return &GithubTracker{URL: ts.URL, Query: q}
	case ProviderJira:
		if tq.LastUpdated != nil {
			// Use the special date for formatting: https://golang.org/pkg/time/#Time.Format
			q = ts.Query + " and updated >= \"" + tq.LastUpdated.Format("2006-01-02 15:04") + "\""
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
	LastUpdatedTime() *time.Time
}

func init() {
	cr = cron.New()
	cr.Start()
}
