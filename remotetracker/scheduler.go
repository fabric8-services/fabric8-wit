package remotetracker

import (
	"log"

	"github.com/jinzhu/gorm"
	"github.com/robfig/cron"
)

// TrackerSchedule capture all configuration
type trackerSchedule struct {
	TrackerQueryID int
	URL            string
	TrackerType    string
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

// ScheduleAllQueries fetch and import of remote tracker items
func (s *Scheduler) ScheduleAllQueries() {
	cr.Stop()
	tq := fetchTrackerQueries(s.db)
	for _, v := range tq {
		s.scheduleSingleQuery(v)
	}
	cr.Start()
}

func fetchTrackerQueries(db *gorm.DB) []trackerSchedule {
	tsList := []trackerSchedule{}
	err := db.Table("trackers").Select("tracker_queries.id as tracker_query_id, trackers.url, trackers.type as tracker_type, tracker_queries.query, tracker_queries.schedule").Joins("left join tracker_queries on tracker_queries.tracker = trackers.id").Scan(&tsList).Error
	if err != nil {
		log.Printf("Fetch failed %v\n", err)
	}
	return tsList
}

// ScheduleSingleQuery schedule fetch and import
func (s *Scheduler) scheduleSingleQuery(ts trackerSchedule) {
	switch ts.TrackerType {
	case "github":
		cr.AddFunc(ts.Schedule, func() {
			item := make(chan map[string]interface{})
			go fetchGithub(ts.URL, ts.Query, item)
			for i := range item {
				uploadGithub(s.db, ts.TrackerQueryID, i)
			}
		})
	case "jira":
		cr.AddFunc(ts.Schedule, func() {
			item := make(chan map[string]interface{})
			go fetchJira(ts.URL, ts.Query, item)
			for i := range item {
				uploadJira(s.db, ts.TrackerQueryID, i)
			}
		})
	}
}

func init() {
	cr = cron.New()
	cr.Start()
}
