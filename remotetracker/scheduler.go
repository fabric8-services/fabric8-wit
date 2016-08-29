package remotetracker

import (
	"log"

	"github.com/jinzhu/gorm"
	"github.com/robfig/cron"
)

// TrackerSchedule capture all configuration
type TrackerSchedule struct {
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
	tq := fetchTrackerQueries(s.db)
	for _, v := range tq {
		s.ScheduleSingleQuery(v)
	}
}

func fetchTrackerQueries(db *gorm.DB) []TrackerSchedule {
	tsList := []TrackerSchedule{}
	err := db.Table("trackers").Select("trackers.url, trackers.type as tracker_type, tracker_queries.query, tracker_queries.schedule").Joins("left join tracker_queries on tracker_queries.tracker = trackers.id").Scan(&tsList).Error
	if err != nil {
		log.Printf("Fetch failed %v\n", err)
	}
	return tsList
}

// ScheduleSingleQuery schedule fetch and import
func (s *Scheduler) ScheduleSingleQuery(ts TrackerSchedule) {
	switch ts.TrackerType {
	case "github":
		cr.AddFunc(ts.Schedule, func() {
			item := make(chan map[string]interface{})
			go fetchGithub(ts.URL, ts.Query, item)
			for i := range item {
				uploadGithub(s.db, i)
			}
		})
		/*
			case "jira":
				cr.AddFunc(ts.Schedule, func() {
					j.Fetch(ts.URL, ts.Query)
					j.Import()
				})
		*/
	}
}

func init() {
	cr = cron.New()
	cr.Start()
}
