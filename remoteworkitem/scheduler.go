package remoteworkitem

import (
	"log"

	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/transaction"
	"github.com/jinzhu/gorm"
	"github.com/robfig/cron"
)

// TrackerSchedule capture all configuration
type trackerSchedule struct {
	TrackerID   int
	URL         string
	TrackerType string
	Query       string
	Schedule    string
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

// ScheduleAllQueries fetch and import of remote tracker items
func (s *Scheduler) ScheduleAllQueries() {
	cr.Stop()
	ts := models.NewGormTransactionSupport(s.db)

	trackerQueries := fetchTrackerQueries(s.db)
	for _, tq := range trackerQueries {
		cr.AddFunc(tq.Schedule, func() {
			tr := LookupProvider(tq)
			for i := range tr.Fetch() {
				transaction.Do(ts, func() error {

					// Save the remote items in a 'temporary' table.
					err := upload(ts.TX(), tq.TrackerID, i)
					if err != nil {
						return err
					}

					// Convert the remote item into a local work item and persist in the DB.
					_, err = convert(ts, tq.TrackerID, i, tq.TrackerType)
					return err

				})
			}
		})
	}
	cr.Start()
}

func fetchTrackerQueries(db *gorm.DB) []trackerSchedule {
	tsList := []trackerSchedule{}
	err := db.Table("tracker_queries").Select("trackers.id as tracker_id, trackers.url, trackers.type as tracker_type, tracker_queries.query, tracker_queries.schedule").Joins("left join trackers on tracker_queries.tracker_id = trackers.id").Where("trackers.deleted_at is NULL AND tracker_queries.deleted_at is NULL").Scan(&tsList).Error
	if err != nil {
		log.Printf("Fetch failed %v\n", err)
	}
	return tsList
}

// LookupProvider provides the respective tracker based on the type
func LookupProvider(ts trackerSchedule) TrackerProvider {
	switch ts.TrackerType {
	case ProviderGithub:
		return &Github{URL: ts.URL, Query: ts.Query}
	case ProviderJira:
		return &Jira{URL: ts.URL, Query: ts.Query}
	}
	return nil
}

// TrackerProvider represents a remote tracker
type TrackerProvider interface {
	Fetch() chan map[string]string
}

func init() {
	cr = cron.New()
	cr.Start()
}
